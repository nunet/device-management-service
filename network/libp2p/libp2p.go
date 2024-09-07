package libp2p

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	kbucket "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"gitlab.com/nunet/device-management-service/types"

	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
	"github.com/spf13/afero"
	"google.golang.org/protobuf/proto"

	dht "github.com/libp2p/go-libp2p-kad-dht"

	libp2pdiscovery "github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"

	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"

	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"

	commonproto "gitlab.com/nunet/device-management-service/proto/generated/v1/common"
)

const (
	MB                 = 1024 * 1024
	MaxMessageLengthMB = 10

	ValidationAccept = pubsub.ValidationAccept
	ValidationReject = pubsub.ValidationReject
	ValidationIgnore = pubsub.ValidationIgnore
)

type (
	ValidationResult = pubsub.ValidationResult
	Validator        func([]byte, interface{}) (ValidationResult, interface{})
)

// Libp2p contains the configuration for a Libp2p instance.
//
// TODO-suggestion: maybe we should call it something else like Libp2pPeer,
// Libp2pHost or just Peer (callers would use libp2p.Peer...)
type Libp2p struct {
	Host              host.Host
	DHT               *dht.IpfsDHT
	PS                peerstore.Peerstore
	pubsub            *pubsub.PubSub
	pubsubTopics      map[string]*pubsub.Topic
	topicValidators   map[string]map[uint64]Validator
	topicSubscription map[string]map[uint64]*pubsub.Subscription
	nextTopicSubID    uint64
	topicMux          sync.RWMutex

	// a list of peers discovered by discovery
	discoveredPeers []peer.AddrInfo
	discovery       libp2pdiscovery.Discovery

	// services
	pingService *ping.PingService

	// tasks
	discoveryTask *bt.Task

	handlerRegistry *HandlerRegistry

	config *types.Libp2pConfig

	// dependencies (db, filesystem...)
	fs afero.Fs
}

// New creates a libp2p instance.
//
// TODO-Suggestion: move types.Libp2pConfig to here for better readability.
// Unless there is a reason to keep within types.
func New(config *types.Libp2pConfig, fs afero.Fs) (*Libp2p, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}

	if config.Scheduler == nil {
		return nil, errors.New("scheduler is nil")
	}

	return &Libp2p{
		config:            config,
		discoveredPeers:   make([]peer.AddrInfo, 0),
		pubsubTopics:      make(map[string]*pubsub.Topic),
		topicSubscription: make(map[string]map[uint64]*pubsub.Subscription),
		topicValidators:   make(map[string]map[uint64]Validator),
		fs:                fs,
	}, nil
}

// Init initializes a libp2p host with its dependencies.
func (l *Libp2p) Init(context context.Context) error {
	host, dht, pubsub, err := NewHost(context, l.config)
	if err != nil {
		zlog.Sugar().Error(err)
		return err
	}

	l.Host = host
	l.DHT = dht
	l.PS = host.Peerstore()
	l.discovery = drouting.NewRoutingDiscovery(dht)
	l.pubsub = pubsub
	l.handlerRegistry = NewHandlerRegistry(host)

	return nil
}

// Start performs network bootstrapping, peer discovery and protocols handling.
func (l *Libp2p) Start(context context.Context) error {
	// set stream handlers
	l.registerStreamHandlers()

	// bootstrap should return error if it had an error
	err := l.Bootstrap(context, l.config.BootstrapPeers)
	if err != nil {
		zlog.Sugar().Errorf("failed to start network: %v", err)
		return err
	}

	// advertise randevouz discovery
	err = l.advertiseForRendezvousDiscovery(context)
	if err != nil {
		// TODO: the error might be misleading as a peer can normally work well if an error
		// is returned here (e.g.: the error is yielded in tests even though all tests pass).
		zlog.Sugar().Errorf("failed to start network with randevouz discovery: %v", err)
	}

	// discover
	err = l.DiscoverDialPeers(context)
	if err != nil {
		zlog.Sugar().Errorf("failed to discover peers: %v", err)
	}

	// register period peer discoveryTask task
	discoveryTask := &bt.Task{
		Name:        "Peer Discovery",
		Description: "Periodic task to discover new peers every 15 minutes",
		Function: func(_ interface{}) error {
			return l.DiscoverDialPeers(context)
		},
		Triggers: []bt.Trigger{&bt.PeriodicTrigger{Interval: 15 * time.Minute}},
	}

	l.discoveryTask = l.config.Scheduler.AddTask(discoveryTask)
	l.config.Scheduler.Start()

	return nil
}

// RegisterStreamMessageHandler registers a stream handler for a specific protocol.
func (l *Libp2p) RegisterStreamMessageHandler(messageType types.MessageType, handler StreamHandler) error {
	if messageType == "" {
		return errors.New("message type is empty")
	}

	if err := l.handlerRegistry.RegisterHandlerWithStreamCallback(messageType, handler); err != nil {
		return fmt.Errorf("failed to register handler %s: %w", messageType, err)
	}

	return nil
}

// RegisterBytesMessageHandler registers a stream handler for a specific protocol and sends bytes to handler func.
func (l *Libp2p) RegisterBytesMessageHandler(messageType types.MessageType, handler func(data []byte)) error {
	if messageType == "" {
		return errors.New("message type is empty")
	}

	if err := l.handlerRegistry.RegisterHandlerWithBytesCallback(messageType, l.handleReadBytesFromStream, handler); err != nil {
		return fmt.Errorf("failed to register handler %s: %w", messageType, err)
	}

	return nil
}

// HandleMessage registers a stream handler for a specific protocol and sends bytes to handler func.
func (l *Libp2p) HandleMessage(messageType string, handler func(data []byte)) error {
	return l.RegisterBytesMessageHandler(types.MessageType(messageType), handler)
}

func (l *Libp2p) handleReadBytesFromStream(s network.Stream) {
	callback, ok := l.handlerRegistry.bytesHandlers[s.Protocol()]
	if !ok {
		s.Close()
		return
	}

	c := bufio.NewReader(s)
	defer s.Close()

	// read the first 8 bytes to determine the size of the message
	msgLengthBuffer := make([]byte, 8)
	_, err := c.Read(msgLengthBuffer)
	if err != nil {
		return
	}

	// create a buffer with the size of the message and then read until its full
	lengthPrefix := binary.LittleEndian.Uint64(msgLengthBuffer)
	buf := make([]byte, lengthPrefix)

	// read the full message
	_, err = io.ReadFull(c, buf)
	if err != nil {
		return
	}

	callback(buf)
}

// UnregisterMessageHandler unregisters a stream handler for a specific protocol.
func (l *Libp2p) UnregisterMessageHandler(messageType string) {
	l.handlerRegistry.UnregisterHandler(types.MessageType(messageType))
}

// SendMessage sends a message to a list of peers.
func (l *Libp2p) SendMessage(ctx context.Context, addrs []string, msg types.MessageEnvelope) error {
	var wg sync.WaitGroup
	errCh := make(chan error, len(addrs))

	for _, addr := range addrs {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()
			err := l.sendMessage(ctx, addr, msg)
			if err != nil {
				errCh <- err
			}
		}(addr)
	}
	wg.Wait()
	close(errCh)

	var result error
	for err := range errCh {
		if result == nil {
			result = err
		} else {
			result = fmt.Errorf("%v; %v", result, err)
		}
	}

	return result
}

// OpenStream opens a stream to a remote address and returns the stream for the caller to handle.
func (l *Libp2p) OpenStream(ctx context.Context, addr string, messageType types.MessageType) (network.Stream, error) {
	maddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid multiaddress: %w", err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(maddr)
	if err != nil {
		return nil, fmt.Errorf("could not resolve peer info: %w", err)
	}

	if err := l.Host.Connect(ctx, *peerInfo); err != nil {
		return nil, fmt.Errorf("failed to connect to peer: %w", err)
	}

	stream, err := l.Host.NewStream(ctx, peerInfo.ID, protocol.ID(messageType))
	if err != nil {
		return nil, fmt.Errorf("failed to open stream: %w", err)
	}

	return stream, nil
}

// GetMultiaddr returns the peer's multiaddr.
func (l *Libp2p) GetMultiaddr() ([]multiaddr.Multiaddr, error) {
	peerInfo := peer.AddrInfo{
		ID:    l.Host.ID(),
		Addrs: l.Host.Addrs(),
	}
	return peer.AddrInfoToP2pAddrs(&peerInfo)
}

// Stop performs a cleanup of any resources used in this package.
func (l *Libp2p) Stop() error {
	var errorMessages []string

	l.config.Scheduler.RemoveTask(l.discoveryTask.ID)

	if err := l.DHT.Close(); err != nil {
		errorMessages = append(errorMessages, err.Error())
	}
	if err := l.Host.Close(); err != nil {
		errorMessages = append(errorMessages, err.Error())
	}

	if len(errorMessages) > 0 {
		return errors.New(strings.Join(errorMessages, "; "))
	}

	return nil
}

// Stat returns the status about the libp2p network.
func (l *Libp2p) Stat() types.NetworkStats {
	lAddrs := make([]string, 0, len(l.Host.Addrs()))
	for _, addr := range l.Host.Addrs() {
		lAddrs = append(lAddrs, addr.String())
	}
	return types.NetworkStats{
		ID:         l.Host.ID().String(),
		ListenAddr: strings.Join(lAddrs, ", "),
	}
}

// Ping the remote address. The remote address is the encoded peer id which will be decoded and used here.
//
// TODO (Return error once): something that was confusing me when using this method is that the error is
// returned twice if any. Once as a field of PingResult and one as a return value.
func (l *Libp2p) Ping(ctx context.Context, peerIDAddress string, timeout time.Duration) (types.PingResult, error) {
	// avoid dial to self attempt
	if peerIDAddress == l.Host.ID().String() {
		err := errors.New("can't ping self")
		return types.PingResult{Success: false, Error: err}, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	remotePeer, err := peer.Decode(peerIDAddress)
	if err != nil {
		return types.PingResult{}, err
	}

	pingChan := ping.Ping(pingCtx, l.Host, remotePeer)

	select {
	case res := <-pingChan:
		if res.Error != nil {
			zlog.Sugar().Errorf("failed to ping peer %s: %v", peerIDAddress, res.Error)
			return types.PingResult{
				Success: false,
				RTT:     res.RTT,
				Error:   res.Error,
			}, res.Error
		}

		return types.PingResult{
			RTT:     res.RTT,
			Success: true,
		}, nil
	case <-pingCtx.Done():
		return types.PingResult{
			Error: pingCtx.Err(),
		}, pingCtx.Err()
	}
}

// ResolveAddress resolves the address by given a peer id.
func (l *Libp2p) ResolveAddress(ctx context.Context, id string) ([]string, error) {
	pid, err := peer.Decode(id)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve invalid peer: %w", err)
	}

	// resolve ourself
	if l.Host.ID().String() == id {
		multiAddrs, err := l.GetMultiaddr()
		if err != nil {
			return nil, fmt.Errorf("failed to resolve self: %w", err)
		}
		resolved := make([]string, len(multiAddrs))
		for i, v := range multiAddrs {
			resolved[i] = v.String()
		}

		return resolved, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	pi, err := l.DHT.FindPeer(ctx, pid)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve address %s: %w", id, err)
	}

	peerInfo := peer.AddrInfo{
		ID:    pi.ID,
		Addrs: pi.Addrs,
	}

	multiAddrs, err := peer.AddrInfoToP2pAddrs(&peerInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to p2p address: %w", err)
	}

	resolved := make([]string, len(multiAddrs))
	for i, v := range multiAddrs {
		resolved[i] = v.String()
	}

	return resolved, nil
}

// Query return all the advertisements in the network related to a key.
// The network is queried to find providers for the given key, and peers which we aren't connected to can be retrieved.
func (l *Libp2p) Query(ctx context.Context, key string) ([]*commonproto.Advertisement, error) {
	if key == "" {
		return nil, errors.New("advertisement key is empty")
	}

	customCID, err := createCIDFromKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cid for key %s: %w", key, err)
	}

	addrInfo, err := l.DHT.FindProviders(ctx, customCID)
	if err != nil {
		return nil, fmt.Errorf("failed to find providers for key %s: %w", key, err)
	}
	advertisements := make([]*commonproto.Advertisement, 0)
	for _, v := range addrInfo {
		// TODO: use go routines to get the values in parallel.
		bytesAdvertisement, err := l.DHT.GetValue(ctx, l.getCustomNamespace(key, v.ID.String()))
		if err != nil {
			continue
		}
		var ad commonproto.Advertisement
		if err := proto.Unmarshal(bytesAdvertisement, &ad); err != nil {
			return nil, fmt.Errorf("failed to unmarshal advertisement payload: %w", err)
		}
		advertisements = append(advertisements, &ad)
	}

	return advertisements, nil
}

// Advertise given data and a key pushes the data to the dht.
func (l *Libp2p) Advertise(ctx context.Context, key string, data []byte) error {
	if key == "" {
		return errors.New("advertisement key is empty")
	}

	pubKeyBytes, err := l.getPublicKey()
	if err != nil {
		return fmt.Errorf("failed to get public key: %w", err)
	}

	envelope := &commonproto.Advertisement{
		PeerId:    l.Host.ID().String(),
		Timestamp: time.Now().Unix(),
		Data:      data,
		PublicKey: pubKeyBytes,
	}

	concatenatedBytes := bytes.Join([][]byte{
		[]byte(envelope.PeerId),
		{byte(envelope.Timestamp)},
		envelope.Data,
		pubKeyBytes,
	}, nil)

	sig, err := l.sign(concatenatedBytes)
	if err != nil {
		return fmt.Errorf("failed to sign advertisement envelope content: %w", err)
	}

	envelope.Signature = sig

	envelopeBytes, err := proto.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("failed to marshal advertise envelope: %w", err)
	}

	customCID, err := createCIDFromKey(key)
	if err != nil {
		return fmt.Errorf("failed to create cid for key %s: %w", key, err)
	}

	err = l.DHT.PutValue(ctx, l.getCustomNamespace(key, l.DHT.PeerID().String()), envelopeBytes)
	if err != nil {
		return fmt.Errorf("failed to put key %s into the dht: %w", key, err)
	}

	err = l.DHT.Provide(ctx, customCID, true)
	if err != nil {
		return fmt.Errorf("failed to provide key %s into the dht: %w", key, err)
	}

	return nil
}

// Unadvertise removes the data from the dht.
func (l *Libp2p) Unadvertise(ctx context.Context, key string) error {
	err := l.DHT.PutValue(ctx, l.getCustomNamespace(key, l.DHT.PeerID().String()), nil)
	if err != nil {
		return fmt.Errorf("failed to remove key %s from the DHT: %w", key, err)
	}

	return nil
}

// Publish publishes data to a topic.
// The requirements are that only one topic handler should exist per topic.
func (l *Libp2p) Publish(ctx context.Context, topic string, data []byte) error {
	topicHandler, err := l.getOrJoinTopicHandler(topic)
	if err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	err = topicHandler.Publish(ctx, data)
	if err != nil {
		return fmt.Errorf("failed to publish to topic %s: %w", topic, err)
	}

	return nil
}

// Subscribe subscribes to a topic and sends the messages to the handler.
func (l *Libp2p) Subscribe(ctx context.Context, topic string, handler func(data []byte), validator Validator) (uint64, error) {
	topicHandler, err := l.getOrJoinTopicHandler(topic)
	if err != nil {
		return 0, fmt.Errorf("failed to subscribe to topic: %w", err)
	}

	sub, err := topicHandler.Subscribe()
	if err != nil {
		return 0, fmt.Errorf("failed to subscribe to topic %s: %w", topic, err)
	}

	l.topicMux.Lock()
	subID := l.nextTopicSubID
	l.nextTopicSubID++
	topicMap, ok := l.topicSubscription[topic]
	if !ok {
		topicMap = make(map[uint64]*pubsub.Subscription)
		l.topicSubscription[topic] = topicMap
	}
	if validator != nil {
		validatorMap, ok := l.topicValidators[topic]
		if !ok {
			if err := l.pubsub.RegisterTopicValidator(topic, l.validate); err != nil {
				sub.Cancel()
				return 0, fmt.Errorf("failed to register topic validator: %w", err)
			}
			validatorMap = make(map[uint64]Validator)
			l.topicValidators[topic] = validatorMap
		}
		validatorMap[subID] = validator
	}
	topicMap[subID] = sub
	l.topicMux.Unlock()

	go func() {
		for {
			msg, err := sub.Next(ctx)
			if err != nil {
				continue
			}
			handler(msg.Data)
		}
	}()

	return subID, nil
}

func (l *Libp2p) validate(_ context.Context, _ peer.ID, msg *pubsub.Message) ValidationResult {
	l.topicMux.Lock()
	validators, ok := l.topicValidators[msg.GetTopic()]
	l.topicMux.Unlock()

	if !ok {
		return ValidationAccept
	}

	for _, validator := range validators {
		result, validatorData := validator(msg.Data, msg.ValidatorData)
		if result != ValidationAccept {
			return result
		}
		msg.ValidatorData = validatorData
	}

	return ValidationAccept
}

func (l *Libp2p) sendMessage(ctx context.Context, addr string, msg types.MessageEnvelope) error {
	peerAddr, err := multiaddr.NewMultiaddr(addr)
	if err != nil {
		return fmt.Errorf("invalid multiaddr %s: %v", addr, err)
	}

	peerInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
	if err != nil {
		return fmt.Errorf("failed to get peer info %s: %v", addr, err)
	}

	// we are delivering a message to ourself
	// we should use the handler to send the message to the handler directly which has been previously registered.
	if peerInfo.ID.String() == l.Host.ID().String() {
		l.handlerRegistry.SendMessageToLocalHandler(msg.Type, msg.Data)
		return nil
	}

	if err := l.Host.Connect(ctx, *peerInfo); err != nil {
		return fmt.Errorf("failed to connect to peer %v: %v", peerInfo.ID, err)
	}

	stream, err := l.Host.NewStream(ctx, peerInfo.ID, protocol.ID(msg.Type))
	if err != nil {
		return fmt.Errorf("failed to open stream to peer %v: %v", peerInfo.ID, err)
	}
	defer stream.Close()

	requestBufferSize := 8 + len(msg.Data)
	if requestBufferSize > MaxMessageLengthMB*MB {
		return fmt.Errorf("message size %d is greater than limit %d bytes", requestBufferSize, MaxMessageLengthMB*MB)
	}

	requestPayloadWithLength := make([]byte, requestBufferSize)
	binary.LittleEndian.PutUint64(requestPayloadWithLength, uint64(len(msg.Data)))
	copy(requestPayloadWithLength[8:], msg.Data)

	_, err = stream.Write(requestPayloadWithLength)
	if err != nil {
		return fmt.Errorf("failed to send message to peer %v: %v", peerInfo.ID, err)
	}

	return nil
}

// getOrJoinTopicHandler gets the topic handler, it will be created if it doesn't exist.
// for publishing and subscribing its needed therefore its implemented in this function.
func (l *Libp2p) getOrJoinTopicHandler(topic string) (*pubsub.Topic, error) {
	l.topicMux.Lock()
	defer l.topicMux.Unlock()
	topicHandler, ok := l.pubsubTopics[topic]
	if !ok {
		t, err := l.pubsub.Join(topic)
		if err != nil {
			return nil, fmt.Errorf("failed to join topic %s: %w", topic, err)
		}
		topicHandler = t
		l.pubsubTopics[topic] = t
	}

	return topicHandler, nil
}

// Unsubscribe cancels the subscription to a topic
func (l *Libp2p) Unsubscribe(topic string, subID uint64) error {
	l.topicMux.Lock()
	defer l.topicMux.Unlock()

	topicHandler, ok := l.pubsubTopics[topic]
	if !ok {
		return fmt.Errorf("not subscribed to topic: %s", topic)
	}

	topicValidators, ok := l.topicValidators[topic]
	if ok {
		delete(topicValidators, subID)
	}

	// delete subscription handler and subscription
	topicSubscriptions, ok := l.topicSubscription[topic]
	if ok {
		sub, ok := topicSubscriptions[subID]
		if ok {
			sub.Cancel()
			delete(topicSubscriptions, subID)
		}
	}

	if len(topicSubscriptions) == 0 {
		delete(l.pubsubTopics, topic)
		if err := topicHandler.Close(); err != nil {
			return fmt.Errorf("failed to close topic handler: %w", err)
		}
	}

	return nil
}

func (l *Libp2p) VisiblePeers() []peer.AddrInfo {
	return l.discoveredPeers
}

func (l *Libp2p) KnownPeers() ([]peer.AddrInfo, error) {
	knownPeers := l.Host.Peerstore().Peers()
	peers := make([]peer.AddrInfo, 0, len(knownPeers))
	for _, p := range knownPeers {
		peers = append(peers, peer.AddrInfo{ID: p})
	}
	return peers, nil
}

func (l *Libp2p) DumpDHTRoutingTable() ([]kbucket.PeerInfo, error) {
	rt := l.DHT.RoutingTable()
	return rt.GetPeerInfos(), nil
}

func (l *Libp2p) registerStreamHandlers() {
	l.pingService = ping.NewPingService(l.Host)
	l.Host.SetStreamHandler(protocol.ID("/ipfs/ping/1.0.0"), l.pingService.PingHandler)
}

func (l *Libp2p) sign(data []byte) ([]byte, error) {
	privKey := l.Host.Peerstore().PrivKey(l.Host.ID())
	if privKey == nil {
		return nil, errors.New("private key not found for the host")
	}

	signature, err := privKey.Sign(data)
	if err != nil {
		return nil, fmt.Errorf("failed to sign data: %w", err)
	}

	return signature, nil
}

func (l *Libp2p) getPublicKey() ([]byte, error) {
	privKey := l.Host.Peerstore().PrivKey(l.Host.ID())
	if privKey == nil {
		return nil, errors.New("private key not found for the host")
	}

	pubKey := privKey.GetPublic()
	return pubKey.Raw()
}

func (l *Libp2p) getCustomNamespace(key, peerID string) string {
	return fmt.Sprintf("%s-%s-%s", l.config.CustomNamespace, key, peerID)
}

func createCIDFromKey(key string) (cid.Cid, error) {
	hash := sha256.Sum256([]byte(key))
	mh, err := multihash.Encode(hash[:], multihash.SHA2_256)
	if err != nil {
		return cid.Cid{}, err
	}
	return cid.NewCidV1(cid.Raw, mh), nil
}

func CleanupPeer(_ peer.ID) error {
	zlog.Warn("CleanupPeer: Stub")
	return nil
}

func PingPeer(_ context.Context, _ peer.ID) (bool, *ping.Result) {
	zlog.Warn("PingPeer: Stub")
	return false, nil
}

func DumpKademliaDHT(_ context.Context) ([]types.PeerData, error) {
	zlog.Warn("DumpKademliaDHT: Stub")
	return nil, nil
}

func OldPingPeer(_ context.Context, _ peer.ID) (bool, *types.PingResult) {
	zlog.Warn("OldPingPeer: Stub")
	return false, nil
}
