package libp2p

import (
	"context"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
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
	"github.com/multiformats/go-multiaddr"
	mafilt "github.com/whyrusleeping/multiaddr-filter"
	"gitlab.com/nunet/device-management-service/types"
)

// NewHost returns a new libp2p host with dht and other related settings.
func NewHost(ctx context.Context, config *types.Libp2pConfig) (host.Host, *dht.IpfsDHT, *pubsub.PubSub, error) {
	var idht *dht.IpfsDHT
	connmgr, err := connmgr.NewConnManager(
		100,
		400,
		connmgr.WithGracePeriod(time.Duration(config.GracePeriodMs)*time.Millisecond),
	)
	if err != nil {
		return nil, nil, nil, err
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
		return nil, nil, nil, err
	}

	var libp2pOpts []libp2p.Option
	baseOpts := []dht.Option{
		dht.ProtocolPrefix(protocol.ID(config.DHTPrefix)),
		dht.NamespacedValidator(strings.ReplaceAll(config.CustomNamespace, "/", ""), dhtValidator{PS: ps}),
		dht.Mode(dht.ModeServer),
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

	optsPS := []pubsub.Option{pubsub.WithMessageSigning(true), pubsub.WithMaxMessageSize(config.GossipMaxMessageSize)}
	gossip, err := pubsub.NewGossipSub(ctx, host, optsPS...)
	// gossip, err := pubsub.NewGossipSubWithRouter(ctx, host, pubsub.DefaultGossipSubRouter(host), optsPS...)
	if err != nil {
		return nil, nil, nil, err
	}
	return host, idht, gossip, nil
}
