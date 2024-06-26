package libp2p

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-libp2p/p2p/protocol/ping"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/multiformats/go-multiaddr"
	mafilt "github.com/whyrusleeping/multiaddr-filter"

	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/utils/validate"
)

// Libp2p contains the configuration for a Libp2p instance.
type Libp2p struct {
	Host   host.Host
	DHT    *dht.IpfsDHT
	PS     peerstore.Peerstore
	peers  []peer.AddrInfo
	config Libp2pConfig
}

type Libp2pConfig struct {
	PrivateKey     crypto.PrivKey
	ListenAddr     []string
	BootstrapPeers []multiaddr.Multiaddr
	Rendezvous     string
	Server         bool
	Scheduler      *bt.Scheduler
}

func (p *Libp2p) Init(config models.NetConfig) error {
	libp2pConfig, err := DecodeSpec(&config.NetworkSpec)
	if err != nil {
		return fmt.Errorf("failed to decode libp2p config: %v", err)
	}

	host, dht, err := newHost(context.Background(), libp2pConfig.PrivateKey, libp2pConfig.Server)
	if err != nil {
		return err
	}
	p.Host = host
	p.DHT = dht
	p.PS = host.Peerstore()

	return nil
}

func (p *Libp2p) Start(ctx context.Context) error {
	err := p.BootstrapNode(ctx)
	if err != nil {
		return fmt.Errorf("bootstraping failed: %v", err)
	}

	p.Host.SetStreamHandler(protocol.ID("/ipfs/ping/1.0.0"), PingHandler)

	err = p.DiscoverDialPeers(ctx)
	if err != nil {
		return err
	}

	// register period peer discoveryTask task
	discoveryTask := &bt.Task{
		Name:        "Peer Discovery",
		Description: "Periodic task to discover new peers every 15 minutes",
		Function: func(args interface{}) error {
			return p.DiscoverDialPeers(ctx)
		},
		Triggers: []bt.Trigger{&bt.PeriodicTrigger{Interval: 15 * time.Minute}},
	}
	p.config.Scheduler.AddTask(discoveryTask)

	// register period offline peer cleanup task
	cleanupTask := &bt.Task{
		Name:        "Peer Discovery",
		Description: "Periodic task to discover new peers every 15 minutes",
		Function: func(args interface{}) error {
			return p.DiscoverDialPeers(ctx)
		},
		Triggers: []bt.Trigger{&bt.PeriodicTrigger{Interval: 5 * time.Minute}},
	}
	p.config.Scheduler.AddTask(cleanupTask)

	return nil
}

type Advertisement struct {
	PeerID    string `json:"peer_id"`
	Timestamp int64  `json:"timestamp,omitempty"`
	Data      []byte `json:"data"`
}

func (p *Libp2p) Advertise(adId string, data []byte) error {
	pubData := Advertisement{}
	pubData.PeerID = p.Host.ID().String()
	peerData, err := p.Host.Peerstore().Get(peer.ID(pubData.PeerID), adId)
	if err != nil {
		zlog.Sugar().Errorf("Advertise error: unable to read data from self store: %v", err)
	}

	if adData, ok := peerData.(Advertisement); ok {
		pubData = adData
	}

	signature, err := signData(p.Host.Peerstore().PrivKey(p.Host.ID()), data)
	if err != nil {
		zlog.Sugar().Infof("Unable to sign data to be advertised: %v", err)
	}
	type update struct {
		Data      []byte `json:"data"`
		Signature []byte `json:"signature"`
	}
	dhtUpdate := update{
		Data:      data,
		Signature: signature,
	}

	dht, err := json.Marshal(dhtUpdate)
	if err != nil {
		zlog.Sugar().Infof("UpdateDHT error: %v", err)
	}

	// Add custom namespace to the key
	namespacedKey := customNamespace + p.Host.ID().String()

	err = p.DHT.PutValue(context.Background(), namespacedKey, dht)
	if err != nil {
		zlog.Sugar().Infof("UpdateDHT error: %v", err)
	}
	return nil
}

func (p *Libp2p) Unadvertise(adId string) error {
	return p.DHT.PutValue(context.Background(), adId, nil)
}

func (p *Libp2p) Publish(topic string, data []byte) error {
	return nil
}

func (p *Libp2p) Subscribe(topic string, handler func(data []byte)) error {
	return nil
}

func (p *Libp2p) Stop() error {
	return nil
}

// ListPeers is a stub.
func (p *Libp2p) ListPeers() ([]peer.AddrInfo, error) {
	zlog.Warn("ListPeers: Stub")
	return nil, nil
}

// ListDHTPeers is a stub.
func (p *Libp2p) ListDHTPeers(ctx context.Context) ([]models.PeerData, error) {
	zlog.Warn("ListDHTPeers: Stub")
	return nil, nil
}

type OpenStream struct {
	ID         int    `json:"id"`
	StreamID   string `json:"stream_id"`
	FromPeer   string `json:"from_peer"`
	TimeOpened string `json:"time_opened"`
}

// IncomingChatRequests is a stub.
func (p *Libp2p) IncomingChatRequests() ([]OpenStream, error) {
	zlog.Warn("IncomingChatRequests: Stub")
	return nil, errors.New("not implemented")
}

// ClearIncomingChatRequests is a stub.
func (p *Libp2p) ClearIncomingChatRequests() error {
	zlog.Warn("ClearIncomingChatRequests: Stub")
	return errors.New("not implemented")
}

// CreateChatStream is a stub.
func (p *Libp2p) CreateChatStream(ctx context.Context, id peer.ID) (network.Stream, error) {
	zlog.Warn("CreateChatStream: Stub")
	return nil, errors.New("not implemented")
}

// StartChat is a stub.
func (p *Libp2p) StartChat(w http.ResponseWriter, r *http.Request, s network.Stream, id string) error {
	zlog.Warn("StartChat: Stub")
	return errors.New("not implemented")
}

// JoinChat is a stub.
func (p *Libp2p) JoinChat(w http.ResponseWriter, r *http.Request, id int) error {
	zlog.Warn("JoinChat: Stub")
	return errors.New("not implemented")
}

// DumpDHT is a stub.
func (p *Libp2p) DumpDHT(ctx context.Context) ([]models.PeerData, error) {
	zlog.Warn("DumpDHT: Stub")
	return nil, errors.New("not implemented")
}

// DefaultDepReqPeer is a stub.
func (p *Libp2p) DefaultDepReqPeer(ctx context.Context, id string) (string, error) {
	zlog.Warn("DefaultDepReqPeer: Stub")
	return "", errors.New("not implemented")
}

// ClearIncomingFileRequests is a stub.
func (p *Libp2p) ClearIncomingFileRequests() error {
	zlog.Warn("ClearIncomingFileRequests: Stub")
	return errors.New("not implemented")
}

// IncomingFileTransferRequests is a stub.
func (p *Libp2p) IncomingFileTransferRequests() ([]OpenStream, error) {
	zlog.Warn("IncomingFileTransferRequests: Stub")
	return nil, errors.New("not implemented")
}

func (p *Libp2p) InitiateTransferFile(ctx context.Context, w http.ResponseWriter, r *http.Request, id peer.ID, path string) error {
	zlog.Warn("InitiateTransferFile: Stub")
	return errors.New("not implemented")
}

func (p *Libp2p) AcceptPeerFileTransfer(ctx context.Context, w http.ResponseWriter, r *http.Request, id int) (string, error) {
	zlog.Warn("AcceptTransferFile: Stub")
	return "", errors.New("not implemented")
}

func PingHandler(s network.Stream) {
	// TODO any ping handling logic should be here
	pingSrv := ping.PingService{}
	pingSrv.PingHandler(s)
}

// Validate checks if the libp2p config is valid
func (lc Libp2pConfig) Validate() error {
	if validate.IsBlank(lc.PrivateKey.GetPublic().Type().String()) {
		return fmt.Errorf("invalid libp2p params: PrivateKey cannot be empty")
	}
	if len(lc.ListenAddr) == 0 {
		return fmt.Errorf("invalid libp2p params: ListenAddr cannot be empty")
	}
	return nil
}

// DecodeSpec decodes a spec config into a libp2p config
func DecodeSpec(spec *models.SpecConfig) (Libp2pConfig, error) {
	if !spec.IsType(string(models.NetP2P)) {
		return Libp2pConfig{}, fmt.Errorf(
			"invalid network type. expected %s, but recieved: %s",
			models.NetP2P,
			spec.Type,
		)
	}

	inputParams := spec.Params
	if inputParams == nil {
		return Libp2pConfig{}, fmt.Errorf("invalid libp2p params: params cannot be nil")
	}

	paramBytes, err := json.Marshal(inputParams)
	if err != nil {
		return Libp2pConfig{}, fmt.Errorf("failed to encode libp2p params: %w", err)
	}

	var libp2pConfig *Libp2pConfig
	err = json.Unmarshal(paramBytes, &libp2pConfig)
	if err != nil {
		return Libp2pConfig{}, fmt.Errorf("failed to decode libp2p params: %w", err)
	}

	return *libp2pConfig, libp2pConfig.Validate()
}

func newHost(ctx context.Context, priv crypto.PrivKey, server bool) (host.Host, *dht.IpfsDHT, error) {

	var idht *dht.IpfsDHT

	connmgr, err := connmgr.NewConnManager(
		100, // Lowwater
		400, // HighWater,
		connmgr.WithGracePeriod(time.Minute),
	)

	if err != nil {
		zlog.Sugar().Errorf("Error Creating Connection Manager: %v", err)
		return nil, nil, err
	}

	filter := multiaddr.NewFilters()
	for _, s := range defaultServerFilters {
		f, err := mafilt.NewMask(s)
		if err != nil {
			zlog.Sugar().Errorf("incorrectly formatted address filter in config: %s - %v", s, err)
		}
		filter.AddFilter(*f, multiaddr.ActionDeny)
	}

	ps, err := pstoremem.NewPeerstore()
	if err != nil {
		zlog.Sugar().Errorf("Couldn't Create Peerstore: %v", err)
		return nil, nil, err
	}

	var libp2pOpts []libp2p.Option
	baseOpts := []dht.Option{
		kadPrefix,
		dht.NamespacedValidator(strings.ReplaceAll(customNamespace, "/", ""), dhtValidator{PS: ps}),
		dht.Mode(dht.ModeServer),
	}

	libp2pOpts = append(libp2pOpts, libp2p.ListenAddrStrings(
		config.GetConfig().P2P.ListenAddress...),
		libp2p.Identity(priv),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			idht, err = dht.New(ctx, h, baseOpts...)
			return idht, err
		}),
		libp2p.Peerstore(ps),
		libp2p.EnableNATService(),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.DefaultTransports,
		libp2p.EnableNATService(),
		libp2p.ConnectionManager(connmgr),
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelayService(
			relay.WithResources(
				relay.Resources{
					MaxReservations:        256,
					MaxCircuits:            32,
					BufferSize:             4096,
					MaxReservationsPerPeer: 8,
					MaxReservationsPerIP:   16,
				},
			),
			relay.WithLimit(&relay.RelayLimit{
				Duration: 5 * time.Minute,
				Data:     1 << 21, // 2 MiB
			}),
		),
		libp2p.EnableAutoRelayWithPeerSource(
			func(ctx context.Context, num int) <-chan peer.AddrInfo {
				r := make(chan peer.AddrInfo)
				go func() {
					defer close(r)
					for i := 0; i < num; i++ {
						select {
						case p := <-newPeer:
							select {
							case r <- p:
							case <-ctx.Done():
								return
							}
						case <-ctx.Done():
							return
						}
					}
				}()
				return r
			},
			autorelay.WithBootDelay(time.Minute),
			autorelay.WithBackoff(30*time.Second),
			autorelay.WithMinCandidates(2),
			autorelay.WithMaxCandidates(3),
			autorelay.WithNumRelays(2),
		),
	)

	if server {
		libp2pOpts = append(libp2pOpts, libp2p.AddrsFactory(makeAddrsFactory([]string{}, []string{}, defaultServerFilters)))
		libp2pOpts = append(libp2pOpts, libp2p.ConnectionGater((*filtersConnectionGater)(filter)))
	} else {
		libp2pOpts = append(libp2pOpts, libp2p.NATPortMap())
	}

	host, err := libp2p.New(libp2pOpts...)

	if err != nil {
		zlog.Sugar().Errorf("Couldn't Create Host: %v", err)
		return nil, nil, err
	}

	zlog.Sugar().Infof("Self Peer Info %s -> %s", host.ID().String(), host.Addrs())

	return host, idht, nil
}
