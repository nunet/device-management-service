// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package libp2p

import (
	"bufio"
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/quic-go/quic-go/http3"

	"github.com/quic-go/quic-go"
	crypto "gitlab.com/nunet/device-management-service/lib/crypto"

	ic "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/network"

	"gitlab.com/nunet/device-management-service/utils/sys"

	cid "github.com/ipfs/go-cid"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	libp2pdiscovery "github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	multiaddr "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	multihash "github.com/multiformats/go-multihash"
	msmux "github.com/multiformats/go-multistream"
	"github.com/spf13/afero"
	"google.golang.org/protobuf/proto"

	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/observability"
	commonproto "gitlab.com/nunet/device-management-service/proto/generated/v1/common"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	MB                 = 1024 * 1024
	maxMessageLengthMB = 1

	ValidationAccept = pubsub.ValidationAccept
	ValidationReject = pubsub.ValidationReject
	ValidationIgnore = pubsub.ValidationIgnore

	readTimeout = 30 * time.Second

	sendSemaphoreLimit = 4096
)

type (
	PeerID            = peer.ID
	ProtocolID        = protocol.ID
	Topic             = pubsub.Topic
	PubSub            = pubsub.PubSub
	ValidationResult  = pubsub.ValidationResult
	Validator         func([]byte, interface{}) (ValidationResult, interface{})
	PeerScoreSnapshot = pubsub.PeerScoreSnapshot
)

// Libp2p contains the configuration for a Libp2p instance.
//
// TODO-suggestion: maybe we should call it something else like Libp2pPeer,
// Libp2pHost or just Peer (callers would use libp2p.Peer...)
type Libp2p struct {
	Host   host.Host
	DHT    *dht.IpfsDHT
	PS     peerstore.Peerstore
	pubsub *PubSub

	ctx    context.Context
	cancel func()

	mx             sync.Mutex
	pubsubAppScore func(peer.ID) float64
	pubsubScore    map[peer.ID]*PeerScoreSnapshot

	topicMux          sync.RWMutex
	pubsubTopics      map[string]*Topic
	topicValidators   map[string]map[uint64]Validator
	topicSubscription map[string]map[uint64]*pubsub.Subscription
	nextTopicSubID    uint64

	// send backpressure semaphore
	sendSemaphore chan struct{}

	// a list of peers discovered by discovery
	discoveredPeers []peer.AddrInfo
	discovery       libp2pdiscovery.Discovery

	// channel to signal when a public IP address has been confirmed
	observedAddrCh chan multiaddr.Multiaddr
	observedAddr   multiaddr.Multiaddr

	// services
	pingService *ping.PingService

	// tasks
	discoveryTask           *bt.Task
	advertiseRendezvousTask *bt.Task

	handlerRegistry *HandlerRegistry

	config *types.Libp2pConfig

	// dependencies (db, filesystem...)
	fs afero.Fs

	subnetsmx              sync.Mutex
	subnets                map[string]*subnet
	isHTTPServerRegistered int32

	// for ip proxying in subnets
	ipproxy          *http3.Server
	ipproxyCtx       context.Context
	ipproxyCtxCancel func()
	ipproxyConns     map[string]*quic.Conn
	ipproxyConnsMx   sync.Mutex

	udpln  *net.UDPConn
	rawqtr *RawQUICTransport

	NetIfaceFactory NetInterfaceFactory // Injected factory for creating NetInterface (for testing/mocking)
}

// This results in a cyclic dependency error
// var _ dmsNetwork.Network = (*Libp2p)(nil)
// TODO: remove this once we move the network types and interfaces to the types package

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

	var netIfaceFactory NetInterfaceFactory
	if config.NetIfaceFactory != nil {
		netIfaceFactory = func(name string) (sys.NetInterface, error) {
			iface, err := config.NetIfaceFactory(name)
			if err != nil {
				return nil, err
			}
			return iface.(sys.NetInterface), nil
		}
	} else if config.NetIfaceFactory == nil {
		netIfaceFactory = func(name string) (sys.NetInterface, error) {
			return sys.NewTunTapInterface(name, sys.NetTunMode, false)
		}
	}

	return &Libp2p{
		config:            config,
		discoveredPeers:   make([]peer.AddrInfo, 0),
		pubsubTopics:      make(map[string]*pubsub.Topic),
		topicSubscription: make(map[string]map[uint64]*pubsub.Subscription),
		topicValidators:   make(map[string]map[uint64]Validator),
		sendSemaphore:     make(chan struct{}, sendSemaphoreLimit),
		fs:                fs,
		subnets:           make(map[string]*subnet),
		observedAddrCh:    make(chan multiaddr.Multiaddr, 1), // buffer of 1 to avoid blocking
		NetIfaceFactory:   netIfaceFactory,                   // from config or nil
	}, nil
}

// Init initializes a libp2p host with its dependencies.
func (l *Libp2p) Init(cfg *config.Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	host, dht, pubsub, udpConn, rqtr, err := NewHost(ctx, l.config, l.broadcastAppScore, l.broadcastScoreInspect)
	if err != nil {
		cancel()
		log.Error(err)
		return err
	}

	l.ctx = ctx
	l.cancel = cancel
	l.Host = host
	l.DHT = dht
	l.PS = host.Peerstore()
	l.discovery = drouting.NewRoutingDiscovery(dht)
	l.pubsub = pubsub
	l.handlerRegistry = NewHandlerRegistry(host)
	l.udpln = udpConn
	l.rawqtr = rqtr

	l.rawqtr.network = l

	// Extract the public key from the private key
	publicKey := l.config.PrivateKey.GetPublic()

	// Derive the DID from the public key
	didInstance := did.FromPublicKey(publicKey)
	if didInstance.Empty() {
		return fmt.Errorf("failed to derive a valid DID from public key")
	}

	log.Infof("Derived DID: %s", didInstance.URI)

	// Initialize the observability package with the host and DID
	if err := observability.Initialize(l.Host, didInstance, cfg); err != nil {
		return fmt.Errorf("failed to initialize observability: %w", err)
	}

	return nil
}

// Start performs network bootstrapping, peer discovery and protocols handling.
func (l *Libp2p) Start() error {
	if l.Host == nil {
		return ErrHostNotInitialized
	}

	// set stream handlers
	l.registerStreamHandlers()

	// connect to bootstrap nodes
	err := l.connectToBootstrapNodes(l.ctx)
	if err != nil {
		log.Errorw("libp2p_bootstrap_failure", "labels", string(observability.LabelNode), "error", err)
		return err
	}
	log.Infow("libp2p_bootstrap_success", "labels", string(observability.LabelNode))

	err = l.bootstrapDHT(l.ctx)
	if err != nil {
		log.Errorw("libp2p_bootstrap_failure", "labels", string(observability.LabelNode), "error", err)
		return err
	}
	log.Infow("libp2p_bootstrap_success", "labels", string(observability.LabelNode))

	// Start random walk
	l.startRandomWalk(l.ctx)

	// watch for local address change
	go l.watchForAddrsChange(l.ctx)

	// discover
	go func() {
		// wait for dht bootstrap
		time.Sleep(1 * time.Minute)

		// advertise randevouz discovery
		err = l.advertiseForRendezvousDiscovery(l.ctx)
		if err != nil {
			log.Warnf("libp2p_advertise_rendezvous_failure", "labels", string(observability.LabelNode), "error", err)
		} else {
			log.Infow("libp2p_advertise_rendezvous_success", "labels", string(observability.LabelNode))
		}

		err = l.discoverDialPeers(l.ctx)
		if err != nil {
			log.Warnf("libp2p_peer_discover_failure", "labels", string(observability.LabelNode), "error", err)
		} else {
			log.Infow("libp2p_peer_discover_success", "labels", string(observability.LabelNode), "foundPeers", len(l.discoveredPeers))
		}
	}()

	// register period peer discoveryTask task
	discoveryTask := &bt.Task{
		Name:        "Peer Discovery",
		Description: "Periodic task to discover new peers every 15 minutes",
		Function: func(_ interface{}) error {
			return l.discoverDialPeers(l.ctx)
		},
		Triggers: []bt.Trigger{&bt.PeriodicTrigger{Interval: 15 * time.Minute}},
	}

	l.discoveryTask = l.config.Scheduler.AddTask(discoveryTask)

	// register rendezvous advertisement task
	advertiseRendezvousTask := &bt.Task{
		Name:        "Rendezvous advertisement",
		Description: "Periodic task to advertise a rendezvous point every 6 hours",
		Function: func(_ interface{}) error {
			return l.advertiseForRendezvousDiscovery(l.ctx)
		},
		Triggers: []bt.Trigger{&bt.PeriodicTrigger{Interval: 6 * time.Hour}},
	}

	l.advertiseRendezvousTask = l.config.Scheduler.AddTask(advertiseRendezvousTask)

	l.config.Scheduler.Start()

	go l.watchForObservedAddr()

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
func (l *Libp2p) RegisterBytesMessageHandler(messageType types.MessageType, handler func(data []byte, peerId peer.ID)) error {
	if messageType == "" {
		return errors.New("message type is empty")
	}

	if err := l.handlerRegistry.RegisterHandlerWithBytesCallback(messageType, l.handleReadBytesFromStream, handler); err != nil {
		return fmt.Errorf("failed to register handler %s: %w", messageType, err)
	}

	return nil
}

// HandleMessage registers a stream handler for a specific protocol and sends bytes to handler func.
func (l *Libp2p) HandleMessage(messageType string, handler func(data []byte, peerId peer.ID)) error {
	return l.RegisterBytesMessageHandler(types.MessageType(messageType), handler)
}

func (l *Libp2p) handleReadBytesFromStream(s network.Stream) {
	l.handlerRegistry.mu.RLock()
	callback, ok := l.handlerRegistry.bytesHandlers[s.Protocol()]
	l.handlerRegistry.mu.RUnlock()
	if !ok {
		_ = s.Reset()
		return
	}

	if err := s.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		_ = s.Reset()
		log.Warnf("error setting read deadline: %s", err)
		return
	}

	c := bufio.NewReader(s)
	defer s.Close()

	// read the first 8 bytes to determine the size of the message
	msgLengthBuffer := make([]byte, 8)
	_, err := c.Read(msgLengthBuffer)
	if err != nil {
		log.Debugf("error reading message length: %s", err)
		_ = s.Reset()
		return
	}

	// create a buffer with the size of the message and then read until its full
	lengthPrefix := binary.LittleEndian.Uint64(msgLengthBuffer)

	// check if the message length is greater than max allowed
	if lengthPrefix > maxMessageLengthMB*MB {
		_ = s.Reset()
		log.Warnf("message length exceeds maximum: %d", lengthPrefix)
		return
	}

	buf := make([]byte, lengthPrefix)

	// read the full message
	_, err = io.ReadFull(c, buf)
	if err != nil {
		log.Debugf("error reading message: %s", err)
		_ = s.Reset()
		return
	}

	_ = s.Close()
	callback(buf, s.Conn().RemotePeer())
}

// UnregisterMessageHandler unregisters a stream handler for a specific protocol.
func (l *Libp2p) UnregisterMessageHandler(messageType string) {
	l.handlerRegistry.UnregisterHandler(types.MessageType(messageType))
}

// SendMessage asynchronously sends a message to a peer
func (l *Libp2p) SendMessage(ctx context.Context, hostID string, msg types.MessageEnvelope, expiry time.Time) error {
	pid, err := peer.Decode(hostID)
	if err != nil {
		return fmt.Errorf("send: invalid peer ID: %w", err)
	}

	// we are delivering a message to ourself
	// we should use the handler to send the message to the handler directly which has been previously registered.
	if pid == l.Host.ID() {
		l.handlerRegistry.SendMessageToLocalHandler(msg.Type, msg.Data, pid)
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, time.Until(expiry))
	select {
	case l.sendSemaphore <- struct{}{}:
		go func() {
			defer cancel()
			defer func() { <-l.sendSemaphore }()
			l.sendMessage(ctx, pid, msg, expiry, nil)
		}()
		return nil
	case <-ctx.Done():
		cancel()
		return ctx.Err()
	}
}

// SendMessageSync synchronously sends a message to a peer
func (l *Libp2p) SendMessageSync(ctx context.Context, hostID string, msg types.MessageEnvelope, expiry time.Time) error {
	pid, err := peer.Decode(hostID)
	if err != nil {
		return fmt.Errorf("send: invalid peer ID: %w", err)
	}

	if pid == l.Host.ID() {
		l.handlerRegistry.SendMessageToLocalHandler(msg.Type, msg.Data, pid)
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, time.Until(expiry))
	defer cancel()

	result := make(chan error, 1)
	l.sendMessage(ctx, pid, msg, expiry, result)

	return <-result
}

// workaround for https://github.com/libp2p/go-libp2p/issues/2983
func (l *Libp2p) newStream(ctx context.Context, pid peer.ID, proto protocol.ID) (network.Stream, error) {
	s, err := l.Host.Network().NewStream(network.WithNoDial(ctx, "already dialed"), pid)
	if err != nil {
		return nil, err
	}

	selected, err := msmux.SelectOneOf([]protocol.ID{proto}, s)
	if err != nil {
		_ = s.Reset()
		return nil, err
	}

	if err := s.SetProtocol(selected); err != nil {
		_ = s.Reset()
		return nil, err
	}

	return s, nil
}

func (l *Libp2p) sendMessage(ctx context.Context, pid peer.ID, msg types.MessageEnvelope, expiry time.Time, result chan error) {
	var err error
	defer func() {
		if result != nil {
			result <- err
		}
	}()

	if !l.PeerConnected(pid) {
		var ai peer.AddrInfo
		ai, err = l.resolvePeerAddress(ctx, pid)
		if err != nil {
			log.Warnf("send: error resolving addresses for peer %s: %s", pid, err)
			return
		}

		if err = l.Host.Connect(ctx, ai); err != nil {
			log.Warnf("send: failed to connect to peer %s: %s", pid, err)
			return
		}
	}

	requestBufferSize := 8 + len(msg.Data)
	if requestBufferSize > maxMessageLengthMB*MB {
		log.Warnf("send: message size %d is greater than limit %d bytes", requestBufferSize, maxMessageLengthMB*MB)
		err = fmt.Errorf("message too large")
		return
	}

	ctx = network.WithAllowLimitedConn(ctx, "send message")

	stream, err := l.newStream(ctx, pid, protocol.ID(msg.Type))
	if err != nil {
		log.Warnf("send: failed to open stream to peer %s: %s", pid, err)
		return
	}
	defer stream.Close()

	if err = stream.SetWriteDeadline(expiry); err != nil {
		_ = stream.Reset()
		log.Warnf("send: failed to set write deadline to peer %s: %s", pid, err)
		return
	}

	requestPayloadWithLength := make([]byte, requestBufferSize)
	binary.LittleEndian.PutUint64(requestPayloadWithLength, uint64(len(msg.Data)))
	copy(requestPayloadWithLength[8:], msg.Data)

	if _, err = stream.Write(requestPayloadWithLength); err != nil {
		_ = stream.Reset()
		log.Warnf("send: failed to send message to peer %s: %s", pid, err)
	}

	if err = stream.CloseWrite(); err != nil {
		_ = stream.Reset()
		log.Warnf("send: failed to flush output to peer %s: %s", pid, err)
	}

	log.Debugf("send %d bytes to peer %s", len(requestPayloadWithLength), pid)
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

	// Cancel context if not nil
	if l.cancel != nil {
		l.cancel()
	}

	// Remove scheduled tasks if scheduler exists
	if l.config != nil && l.config.Scheduler != nil {
		// Only remove tasks if they exist
		if l.discoveryTask != nil {
			l.config.Scheduler.RemoveTask(l.discoveryTask.ID)
		}
		if l.advertiseRendezvousTask != nil {
			l.config.Scheduler.RemoveTask(l.advertiseRendezvousTask.ID)
		}
	}

	// Close DHT if not nil
	if l.DHT != nil {
		if err := l.DHT.Close(); err != nil {
			errorMessages = append(errorMessages, err.Error())
		}
	}

	// Close Host if not nil
	if l.Host != nil {
		if err := l.Host.Close(); err != nil {
			errorMessages = append(errorMessages, err.Error())
		}
	}

	// Close subnets
	if l.subnets != nil {
		for subnetID := range l.subnets {
			err := l.DestroySubnet(subnetID)
			if err != nil {
				errorMessages = append(errorMessages, err.Error())
			}
		}
	}

	if l.udpln != nil {
		if err := l.udpln.Close(); err != nil && !strings.Contains(err.Error(), "use of closed network connection") {
			errorMessages = append(errorMessages, err.Error())
		}
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

// Peers returns a list of peers from the peer store
func (l *Libp2p) Peers() []peer.ID {
	return l.PS.Peers()
}

// Connect connects to a peer by its multiaddress and returns an error if any
func (l *Libp2p) Connect(ctx context.Context, peerMultiAddr string) error {
	if peerMultiAddr == "" {
		return fmt.Errorf("peer multiaddress is empty")
	}

	log.Infof("Creating multiaddress from peerMultiAddr: %s", peerMultiAddr)
	peerAddr, err := multiaddr.NewMultiaddr(peerMultiAddr)
	if err != nil {
		log.Infof("Invalid multiaddress: %v", err)
		return fmt.Errorf("invalid multiaddress: %w", err)
	}

	log.Infof("Resolving peer info from multiaddress")
	addrInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
	if err != nil {
		log.Infof("Could not resolve peer info: %v", err)
		return fmt.Errorf("could not resolve peer info: %w", err)
	}

	log.Infof("Connecting to peer: %s", peerMultiAddr)
	if err := l.Host.Connect(ctx, *addrInfo); err != nil {
		log.Infof("Failed to connect to peer %s: %v", peerMultiAddr, err)
		return fmt.Errorf("failed to connect to peer %s: %w", peerMultiAddr, err)
	}

	return nil
}

// GetPeerIP gets the ip of the peer from the peer store
func (l *Libp2p) GetPeerIP(p PeerID) string {
	addrs := l.Host.Peerstore().Addrs(p)

	for _, addr := range addrs {
		addrParts := strings.Split(addr.String(), "/")

		for i, part := range addrParts {
			if part == "ip4" || part == "ip6" {
				return addrParts[i+1]
			}
		}
	}

	return ""
}

func (l *Libp2p) watchForObservedAddr() {
	sub, err := l.Host.EventBus().Subscribe(new(event.EvtPeerIdentificationCompleted))
	if err != nil {
		log.Debugf("could not subscribe to event: %w", err)
		return
	}
	defer sub.Close()

	// track address observations
	addrCount := make(map[string]int)
	var addrMux sync.Mutex

	for e := range sub.Out() {
		event := e.(event.EvtPeerIdentificationCompleted)

		if event.ObservedAddr.String() == "" {
			continue
		}

		// peer that reported the event
		isPeerPublic := slices.ContainsFunc(event.ListenAddrs, manet.IsPublicAddr)
		if !isPeerPublic {
			continue
		}

		if !manet.IsPublicAddr(event.ObservedAddr) {
			continue
		}
		// skip relays
		addrStr := event.ObservedAddr.String()
		if strings.Contains(addrStr, "p2p-circuit") {
			continue
		}

		ip, err := manet.ToIP(event.ObservedAddr)
		if err != nil {
			continue
		}

		addrMux.Lock()
		addrCount[ip.String()]++
		count := addrCount[ip.String()]
		addrMux.Unlock()

		log.Debugf("got public ip: %s (seen %d times)", ip.String(), count)

		if count >= 3 {
			l.mx.Lock()
			l.observedAddr = event.ObservedAddr
			l.mx.Unlock()
			log.Debugf("confirmed public address after seeing it %d times: %s", count, addrStr)

			// send the observed address on the channel
			select {
			case l.observedAddrCh <- event.ObservedAddr:
				log.Debugf("sent observed address signal: %s", addrStr)
			default:
				log.Debugf("channel full, couldn't send observed address signal: %s", addrStr)
			}
			return
		}
	}
}

// GetHostID returns the host ID.
func (l *Libp2p) GetHostID() PeerID {
	return l.Host.ID()
}

// GetPeerPubKey returns the public key for the given peerID.
func (l *Libp2p) GetPeerPubKey(peerID PeerID) crypto.PubKey {
	return l.Host.Peerstore().PubKey(peerID)
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
			log.Errorf("failed to ping peer %s: %v", peerIDAddress, res.Error)
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
	ai, err := l.resolveAddress(ctx, id)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0, len(ai.Addrs))
	for _, addr := range ai.Addrs {
		result = append(result, fmt.Sprintf("%s/p2p/%s", addr, id))
	}

	return result, nil
}

func (l *Libp2p) resolveAddress(ctx context.Context, id string) (peer.AddrInfo, error) {
	pid, err := peer.Decode(id)
	if err != nil {
		return peer.AddrInfo{}, fmt.Errorf("failed to resolve invalid peer: %w", err)
	}

	return l.resolvePeerAddress(ctx, pid)
}

func (l *Libp2p) resolvePeerAddress(ctx context.Context, pid peer.ID) (peer.AddrInfo, error) {
	// resolve ourself
	if l.Host.ID() == pid {
		addrs, err := l.GetMultiaddr()
		if err != nil {
			return peer.AddrInfo{}, fmt.Errorf("failed to resolve self: %w", err)
		}

		return peer.AddrInfo{ID: pid, Addrs: addrs}, nil
	}

	if l.PeerConnected(pid) {
		addrs := l.Host.Peerstore().Addrs(pid)
		return peer.AddrInfo{
			ID:    pid,
			Addrs: addrs,
		}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	pi, err := l.DHT.FindPeer(ctx, pid)
	if err != nil {
		return peer.AddrInfo{}, fmt.Errorf("failed to resolve address for peer %s: %w", pid, err)
	}

	return pi, nil
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

	log.Infof("advertised key: %s", key)
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
	l.topicMux.RLock()
	validators, ok := l.topicValidators[msg.GetTopic()]
	l.topicMux.RUnlock()

	if !ok {
		return ValidationAccept
	}

	for _, validator := range validators {
		result, _ := validator(msg.Data, msg.ValidatorData)
		if result != ValidationAccept {
			return result
		}
	}

	return ValidationAccept
}

func (l *Libp2p) SetupBroadcastTopic(topic string, setup func(*Topic) error) error {
	t, ok := l.pubsubTopics[topic]
	if !ok {
		return fmt.Errorf("%w: %s", ErrNotSubscribed, topic)
	}

	return setup(t)
}

func (l *Libp2p) SetBroadcastAppScore(f func(peer.ID) float64) {
	l.mx.Lock()
	defer l.mx.Unlock()

	l.pubsubAppScore = f
}

func (l *Libp2p) broadcastAppScore(p peer.ID) float64 {
	f := func(peer.ID) float64 { return 0 }

	l.mx.Lock()
	if l.pubsubAppScore != nil {
		f = l.pubsubAppScore
	}
	l.mx.Unlock()

	return f(p)
}

func (l *Libp2p) GetBroadcastScore() map[peer.ID]*PeerScoreSnapshot {
	l.mx.Lock()
	defer l.mx.Unlock()

	return l.pubsubScore
}

func (l *Libp2p) broadcastScoreInspect(score map[peer.ID]*PeerScoreSnapshot) {
	l.mx.Lock()
	defer l.mx.Unlock()

	l.pubsubScore = score
}

func (l *Libp2p) watchForAddrsChange(ctx context.Context) {
	sub, err := l.Host.EventBus().Subscribe(&event.EvtLocalAddressesUpdated{})
	if err != nil {
		log.Errorf("failed to subscribe to event bus: %v", err)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-sub.Out():
			log.Debug("network address changed. trying to be bootstrap again.")
			if err = l.connectToBootstrapNodes(l.ctx); err != nil {
				log.Errorf("failed to start network: %v", err)
			}
		}
	}
}

func (l *Libp2p) Notify(
	ctx context.Context,
	preconnected func(peer.ID, []protocol.ID, int),
	connected, disconnected func(peer.ID),
	identified, updated func(peer.ID, []protocol.ID),
) error {
	sub, err := l.Host.EventBus().Subscribe([]interface{}{
		&event.EvtPeerConnectednessChanged{},
		&event.EvtPeerIdentificationCompleted{},
		&event.EvtPeerProtocolsUpdated{},
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to event bus: %w", err)
	}

	for _, p := range l.Host.Network().Peers() {
		switch l.Host.Network().Connectedness(p) {
		case network.Limited:
			fallthrough
		case network.Connected:
			protos, _ := l.Host.Peerstore().GetProtocols(p)
			preconnected(p, protos, len(l.Host.Network().ConnsToPeer(p)))
		}
	}

	go func() {
		defer sub.Close()

		for ctx.Err() == nil {
			var ev any
			select {
			case <-ctx.Done():
				return
			case ev = <-sub.Out():
				switch evt := ev.(type) {
				case event.EvtPeerConnectednessChanged:
					switch evt.Connectedness {
					case network.Limited:
						fallthrough
					case network.Connected:
						connected(evt.Peer)
					case network.NotConnected:
						disconnected(evt.Peer)
					}
				case event.EvtPeerIdentificationCompleted:
					identified(evt.Peer, evt.Protocols)
				case event.EvtPeerProtocolsUpdated:
					updated(evt.Peer, evt.Added)
				}
			}
		}
	}()

	return nil
}

func (l *Libp2p) PeerConnected(p PeerID) bool {
	switch l.Host.Network().Connectedness(p) {
	case network.Limited:
		return true
	case network.Connected:
		return true
	default:
		return false
	}
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

func (l *Libp2p) HostPublicIP() (net.IP, error) {
	if l.config.Env == "dev" || l.config.Env == "test" {
		return l.listeningIP()
	}
	addr, err := l.waitForObservedAddr(l.ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve observed addr: %w", err)
	}
	return manet.ToIP(addr)
}

func (l *Libp2p) listeningIP() (net.IP, error) {
	var privIP net.IP
	var hasPrivIP bool
	if len(l.Host.Addrs()) > 4 && l.config.Env != "production" {
		for _, addr := range l.Host.Addrs() {
			if manet.IsPrivateAddr(addr) && !strings.Contains(addr.String(), "/ip4/127.0.0.1/udp") {
				ip, err := manet.ToIP(addr)
				if err != nil {
					return nil, fmt.Errorf("failed to convert multiaddr to IP: %w", err)
				}
				return ip, nil
			}
		}
	} else {
		for _, addr := range l.Host.Addrs() {
			if manet.IsPublicAddr(addr) {
				ip, err := manet.ToIP(addr)
				if err != nil {
					return nil, fmt.Errorf("failed to convert multiaddr to IP: %w", err)
				}
				return ip, nil
			} else if manet.IsPrivateAddr(addr) {
				ip, err := manet.ToIP(addr)
				if err != nil {
					return nil, fmt.Errorf("failed to convert multiaddr to IP: %w", err)
				}
				privIP = ip
				hasPrivIP = true
			}
		}
		if hasPrivIP {
			return privIP, nil
		}
	}
	return net.ParseIP("127.0.0.1"), nil
}

// WaitForObservedAddr waits for the node to confirm its public IP address
// Returns the observed multiaddress or an error if the context expires
func (l *Libp2p) waitForObservedAddr(ctx context.Context) (multiaddr.Multiaddr, error) {
	// if we already have an observed address, return it immediately
	l.mx.Lock()
	if l.observedAddr != nil {
		addr := l.observedAddr
		l.mx.Unlock()
		return addr, nil
	}
	l.mx.Unlock()

	// otherwise wait for the signal
	select {
	case addr := <-l.observedAddrCh:
		return addr, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *Libp2p) registerStreamHandlers() {
	l.pingService = ping.NewPingService(l.Host)
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

func (l *Libp2p) RawQUICConnect(target peer.ID, serverName string) (*quic.Conn, *netip.AddrPort, error) {
	if l.config.Env == "dev" || l.config.Env == "test" {
		return l.rawQUICConnect(target, serverName, false)
	}
	return l.rawQUICConnect(target, serverName, true)
}

func (l *Libp2p) RawQUICConnectLocal(target peer.ID, serverName string) (*quic.Conn, *netip.AddrPort, error) {
	return l.rawQUICConnect(target, serverName, false)
}

func (l *Libp2p) rawQUICConnect(target peer.ID, serverName string, onlyPublicAddress bool) (*quic.Conn, *netip.AddrPort, error) {
	connected := make(chan struct{}, 1)
	go func() {
		sub, err := l.Host.EventBus().Subscribe(new(event.EvtPeerConnectednessChanged))
		if err != nil {
			log.Fatal("failed to subscribe to peer connectedness changed event: ", err)
		}
		defer sub.Close()
		for ev := range sub.Out() {
			e := ev.(event.EvtPeerConnectednessChanged)
			if e.Peer == target {
				msg := fmt.Sprintf("peer connectedness changed: %s\n", e.Connectedness)
				for _, c := range l.Host.Network().ConnsToPeer(target) {
					msg += fmt.Sprintf("\t%s <-> %s\n", c.LocalMultiaddr(), c.RemoteMultiaddr())
				}
				log.Debug(msg)
				if e.Connectedness == network.Connected {
					connected <- struct{}{}
				}
			}
		}
	}()

	ai, err := l.DHT.FindPeer(context.Background(), target)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to find peer: %w", err)
	}

	// 1st step: check if we have a public QUIC address for the target
	// If we do, we can directly dial it.
	var udpAddr *net.UDPAddr
	rawQuicAddrs := getRawQUICAddrs(l.Host.Peerstore().Addrs(target))
	for _, a := range rawQuicAddrs {
		fmt.Println("found address", a, "for target", target)
		if onlyPublicAddress {
			if manet.IsPublicAddr(a) && isQUICAddr(a) {
				udpAddr, err = quicAddrToNetAddr(a)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to convert multiaddr to net.UDPAddr: %w", err)
				}
			}
		} else {
			if isQUICAddr(a) {
				udpAddr, err = quicAddrToNetAddr(a)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to convert multiaddr to net.UDPAddr: %w", err)
				}

				break
			}
		}
	}

	if udpAddr != nil {
		addr := udpAddr.AddrPort()
		conn, err := dialSubnetQUICLayer(l, l.rawqtr.Transport, udpAddr, serverName)
		return conn, &addr, err
	}

	// 2nd step: connect to the target via a relay address.
	// If we don't have a public QUIC address for the target,
	// we need to connect to it via a relay address.
	if err := l.Host.Connect(context.Background(), ai); err != nil {
		return nil, nil, fmt.Errorf("failed to connect to peer: %w", err)
	}

	// As soon as the relayed peer accepts the connection via the relay,
	// it tries to establish a direction connection back to us using the DCUtR protocol.
	// We wait for this connection to be established.
	select {
	case <-connected:
	case <-time.After(2 * time.Minute):
		return nil, nil, fmt.Errorf("timed out waiting for direct (e.g. hole-punched) connection")
	}

	// Now that we have a direct connection to the target, we can dial another
	// QUIC connection on the same 4-tupe. This works since QUIC demultiplexes connections
	// based on their connection ID.
	var directAddr *net.UDPAddr
	for _, c := range l.Host.Network().ConnsToPeer(target) {
		if a := c.RemoteMultiaddr(); isQUICAddr(a) {
			directAddr, err = quicAddrToNetAddr(a)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to convert multiaddr to net.UDPAddr: %w", err)
			}
			log.Debugf("found QUIC address: %s", a)
			break
		}
	}

	// Due to https://github.com/libp2p/go-libp2p/issues/3101, we can't rely on the Connectedness connection state,
	// as it doesn't distinguish between direct and connections via an unlimited relay.
	start := time.Now()
	ticker := time.NewTicker(25 * time.Millisecond)
	defer ticker.Stop()
connectLoop:
	for now := range ticker.C {
		if now.Sub(start) > 5*time.Second {
			break
		}
		for _, c := range l.Host.Network().ConnsToPeer(target) {
			if a := c.RemoteMultiaddr(); isQUICAddr(a) {
				directAddr, err = quicAddrToNetAddr(a)
				if err != nil {
					return nil, nil, fmt.Errorf("failed to convert multiaddr to net.UDPAddr: %w", err)
				}
				if directAddr.Port == 10000 {
					break
				}
				break connectLoop
			}
		}
	}
	if directAddr == nil {
		return nil, nil, fmt.Errorf("failed to find a direct QUIC address for peer %s after hole punching", target)
	}

	log.Debugf("dialing QUIC address: %s", directAddr)
	log.Debugf("found hole punched connection, addr: %s:%d", directAddr.IP.String(), directAddr.Port)

	addr := directAddr.AddrPort()
	conn, err := dialSubnetQUICLayer(l, l.rawqtr.Transport, directAddr, serverName)
	return conn, &addr, err
}

func dialSubnetQUICLayer(l *Libp2p, tr *quic.Transport, addr *net.UDPAddr, servName string) (*quic.Conn, error) {
	priv, err := l.config.PrivateKey.Raw()
	if err != nil {
		return nil, fmt.Errorf("failed to get private key: %w", err)
	}
	cert, err := generateSelfSignedCert(ed25519.PrivateKey(priv), []string{fmt.Sprintf("%s.nunet.internal", servName)})
	if err != nil {
		return nil, fmt.Errorf("failed to generate self signed certificate: %w", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	log.Debugf("dialing QUIC address: %s", addr)
	conn, err := tr.Dial(
		ctx,
		addr,
		&tls.Config{
			InsecureSkipVerify:       true,
			VerifyPeerCertificate:    makeVerifySubnetPeerCertificateFn(l),
			PreferServerCipherSuites: true,
			ClientAuth:               tls.RequireAndVerifyClientCert,
			Certificates:             []tls.Certificate{*cert},
			NextProtos:               []string{"raw"},
			ServerName:               fmt.Sprintf("%s.nunet.internal", servName),
		},
		&quic.Config{
			EnableDatagrams: true,
		},
	)
	if err != nil {
		return nil, fmt.Errorf("failed to dial QUIC address: %w", err)
	}
	return conn, nil
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

func makeVerifySubnetPeerCertificateFn(l *Libp2p) func([][]byte, [][]*x509.Certificate) error {
	return func(rawCerts [][]byte, _ [][]*x509.Certificate) error {
		if len(l.subnets) == 0 {
			return fmt.Errorf("peer not a member of any subnet, invalidating cert")
		}

		cert, err := x509.ParseCertificate(rawCerts[0])
		if err != nil {
			return fmt.Errorf("failed to parse certificate: %v", err)
		}

		// Check expiration
		if time.Now().Before(cert.NotBefore) || time.Now().After(cert.NotAfter) {
			return fmt.Errorf("certificate is expired or not yet valid")
		}

		subnetID := strings.Split(cert.Subject.CommonName, ".")[0]
		// Check server name
		if _, ok := l.subnets[subnetID]; ok && slices.Contains(cert.DNSNames, cert.Subject.CommonName) {
			return fmt.Errorf("either server name does not match certificate or peer not a member of provided subnets")
		}

		var pubKey crypto.PubKey
		switch algoType := strings.ToLower(cert.PublicKeyAlgorithm.String()); algoType {
		case "ecdsa":
			key, ok := cert.PublicKey.(ecdsa.PublicKey)
			if !ok {
				return fmt.Errorf("failed to cast public key to ecdsa type=%T", cert.PublicKey)
			}
			pubkey, err := key.ECDH()
			if err != nil {
				return fmt.Errorf("failed to get ecdh public key: %v", err)
			}
			pubKey, err = ic.UnmarshalECDSAPublicKey(pubkey.Bytes())
			if err != nil {
				return fmt.Errorf("failed to unmarshal ecdsa public key: %v", err)
			}
		case "rsa":
			key, ok := cert.PublicKey.(rsa.PublicKey)
			if !ok {
				return fmt.Errorf("failed to cast public key to rsa, type=%T", cert.PublicKey)
			}
			rawBytes, err := x509.MarshalPKIXPublicKey(key)
			if err != nil {
				return fmt.Errorf("failed to marshal PKIX public key: %v", err)
			}
			pubKey, err = ic.UnmarshalRsaPublicKey(rawBytes)
			if err != nil {
				return fmt.Errorf("failed to unmarshal rsa public key: %v", err)
			}

		case "ed25519":
			key, ok := cert.PublicKey.(ed25519.PublicKey)
			if !ok {
				return fmt.Errorf("failed to cast public key to ed25519, type=%T", cert.PublicKey)
			}
			pubkey, err := ic.UnmarshalEd25519PublicKey([]byte(key))
			if err != nil {
				return fmt.Errorf("failed to unmarshal ed25519 public key: %v", err)
			}

			pubKey = pubkey

		default:
			return fmt.Errorf("unsupported public key type: %T, %s", cert.PublicKey, cert.PublicKeyAlgorithm.String())
		}

		peerID, err := peer.IDFromPublicKey(pubKey)
		if err != nil {
			return fmt.Errorf("failed to get peer id from public key: %v", err)
		}
		for _, subnet := range l.subnets {
			peerMap := subnet.info.rtable.All()
			if _, ok := peerMap[peerID]; ok {
				log.Debugf("peer is a member of subnet, allowing raw quic connection (peerID=%S, subnet=%s)", peerID.String(), subnet.info.id)
				goto done
			}
		}

		return fmt.Errorf("peer not a member of any subnet, invalidating cert")

	done:
		return nil
	}
}

func getRawQUICAddrs(multiaddrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
	rawQuicAddrs := make([]multiaddr.Multiaddr, 0)
	for _, a := range multiaddrs {
		if isQUICAddr(a) {
			_, err := quicAddrToNetAddr(a)
			if err != nil {
				log.Errorf("failed to convert multiaddr to net.UDPAddr: %v", err)
				continue
			}
			rawQuicAddrs = append(rawQuicAddrs, a)
		}
	}
	return rawQuicAddrs
}
