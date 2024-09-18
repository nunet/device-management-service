package libp2p

import (
	"context"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	quic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	webtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	mafilt "github.com/whyrusleeping/multiaddr-filter"
	"gitlab.com/nunet/device-management-service/types"
)

// NewHost returns a new libp2p host with dht and other related settings.
func NewHost(ctx context.Context, config *types.Libp2pConfig, appScore func(p peer.ID) float64, scoreInspect pubsub.ExtendedPeerScoreInspectFn) (host.Host, *dht.IpfsDHT, *pubsub.PubSub, error) {
	newPeer := make(chan peer.AddrInfo)

	var idht *dht.IpfsDHT
	connmgr, err := connmgr.NewConnManager(
		100,
		400,
		connmgr.WithGracePeriod(time.Duration(config.GracePeriodMs)*time.Millisecond),
	)
	if err != nil {
		return nil, nil, nil, err
	}

	filter := ma.NewFilters()
	for _, s := range defaultServerFilters {
		f, err := mafilt.NewMask(s)
		if err != nil {
			log.Errorf("incorrectly formatted address filter in config: %s - %v", s, err)
		}
		filter.AddFilter(*f, ma.ActionDeny)
	}

	ps, err := pstoremem.NewPeerstore()
	if err != nil {
		return nil, nil, nil, err
	}

	var libp2pOpts []libp2p.Option
	baseOpts := []dht.Option{
		dht.ProtocolPrefix(protocol.ID(config.DHTPrefix)),
		dht.NamespacedValidator(strings.ReplaceAll(config.CustomNamespace, "/", ""), dhtValidator{PS: ps}),
		dht.Mode(dht.ModeAutoServer),
	}

	libp2pOpts = append(libp2pOpts, libp2p.ListenAddrStrings(config.ListenAddress...),
		libp2p.Identity(config.PrivateKey),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			idht, err = dht.New(ctx, h, baseOpts...)
			return idht, err
		}),
		libp2p.Peerstore(ps),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Security(noise.ID, noise.New),
		// libp2p.NoListenAddrs,
		libp2p.ChainOptions(
			libp2p.Transport(tcp.NewTCPTransport),
			libp2p.Transport(quic.NewTransport),
			libp2p.Transport(webtransport.New),
			libp2p.Transport(ws.New),
		),
		libp2p.EnableNATService(),
		libp2p.ConnectionManager(connmgr),
		libp2p.EnableRelay(),
		libp2p.EnableHolePunching(),
		libp2p.EnableRelayService(
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

	if config.Server {
		libp2pOpts = append(libp2pOpts, libp2p.AddrsFactory(makeAddrsFactory([]string{}, []string{}, defaultServerFilters)))
		libp2pOpts = append(libp2pOpts, libp2p.ConnectionGater((*filtersConnectionGater)(filter)))
	} else {
		libp2pOpts = append(libp2pOpts, libp2p.NATPortMap())
	}

	host, err := libp2p.New(libp2pOpts...)
	if err != nil {
		return nil, nil, nil, err
	}

	go watchForNewPeers(ctx, host, newPeer)

	optsPS := []pubsub.Option{
		pubsub.WithFloodPublish(true),
		pubsub.WithMessageSigning(true),
		pubsub.WithMaxMessageSize(config.GossipMaxMessageSize),
		pubsub.WithPeerScore(
			&pubsub.PeerScoreParams{
				SkipAtomicValidation: true,
				Topics:               make(map[string]*pubsub.TopicScoreParams),
				TopicScoreCap:        10,
				AppSpecificScore:     appScore,
				AppSpecificWeight:    1,
				DecayInterval:        time.Hour,
				DecayToZero:          0.001,
				RetainScore:          6 * time.Hour,
			},
			&pubsub.PeerScoreThresholds{
				GossipThreshold:             -500,
				PublishThreshold:            -1000,
				GraylistThreshold:           -2500,
				AcceptPXThreshold:           0, // TODO for public mainnet we should limit to botostrappers and set them up without a mesh
				OpportunisticGraftThreshold: 2.5,
			},
		),
		pubsub.WithPeerExchange(true),
		pubsub.WithPeerScoreInspect(scoreInspect, time.Second),
	}
	gossip, err := pubsub.NewGossipSub(ctx, host, optsPS...)
	// gossip, err := pubsub.NewGossipSubWithRouter(ctx, host, pubsub.DefaultGossipSubRouter(host), optsPS...)
	if err != nil {
		return nil, nil, nil, err
	}
	return host, idht, gossip, nil
}

func watchForNewPeers(ctx context.Context, host host.Host, newPeer chan peer.AddrInfo) {
	sub, err := host.EventBus().Subscribe([]interface{}{
		&event.EvtPeerIdentificationCompleted{},
		&event.EvtPeerProtocolsUpdated{},
	})
	if err != nil {
		log.Errorf("failed to subscribe to peer identification events: %v", err)
		return
	}
	defer sub.Close()

	for ctx.Err() == nil {
		var ev any

		select {
		case <-ctx.Done():
			return
		case ev = <-sub.Out():
		}

		if ev, ok := ev.(event.EvtPeerIdentificationCompleted); ok {
			var publicAddrs []ma.Multiaddr
			for _, addr := range ev.ListenAddrs {
				if manet.IsPublicAddr(addr) {
					publicAddrs = append(publicAddrs, addr)
				}
			}
			if len(publicAddrs) > 0 {
				newPeer <- peer.AddrInfo{ID: ev.Peer, Addrs: publicAddrs}
			}
		}
	}
}
