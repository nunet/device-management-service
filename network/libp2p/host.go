// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package libp2p

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/p2p/host/autorelay"
	"github.com/libp2p/go-libp2p/p2p/protocol/holepunch"
	"github.com/quic-go/quic-go"

	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/libp2p/go-libp2p/core/event"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/libp2p/go-libp2p/core/routing"
	"github.com/libp2p/go-libp2p/p2p/host/peerstore/pstoremem"
	rcmgr "github.com/libp2p/go-libp2p/p2p/host/resource-manager"
	"github.com/libp2p/go-libp2p/p2p/net/connmgr"
	"github.com/libp2p/go-libp2p/p2p/protocol/circuitv2/relay"
	"github.com/libp2p/go-libp2p/p2p/security/noise"
	libp2ptls "github.com/libp2p/go-libp2p/p2p/security/tls"
	"github.com/libp2p/go-libp2p/p2p/transport/quicreuse"
	"github.com/libp2p/go-libp2p/p2p/transport/tcp"
	ws "github.com/libp2p/go-libp2p/p2p/transport/websocket"
	webtransport "github.com/libp2p/go-libp2p/p2p/transport/webtransport"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	mafilt "github.com/whyrusleeping/multiaddr-filter"

	libp2pquic "github.com/libp2p/go-libp2p/p2p/transport/quic"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

// NewHost returns a new libp2p host with dht and other related settings.
func NewHost(ctx context.Context, config *types.Libp2pConfig, appScore func(p peer.ID) float64, scoreInspect pubsub.ExtendedPeerScoreInspectFn) (host.Host, *dht.IpfsDHT, *pubsub.PubSub, *net.UDPConn, *RawQUICTransport, error) {
	newPeer := make(chan peer.AddrInfo)

	var idht *dht.IpfsDHT
	connmgr, err := connmgr.NewConnManager(
		100,
		400,
		connmgr.WithGracePeriod(time.Duration(config.GracePeriodMs)*time.Millisecond),
	)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	filter := ma.NewFilters()
	for _, s := range defaultServerFilters {
		f, err := mafilt.NewMask(s)
		if err != nil {
			log.Errorw("incorrectly formatted address filter in config",
				"labels", string(observability.LabelNode),
				"filter", s,
				"error", err,
			)
		}
		filter.AddFilter(*f, ma.ActionDeny)
	}

	ps, err := pstoremem.NewPeerstore()
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	var libp2pOpts []libp2p.Option
	dhtOpts := []dht.Option{
		dht.ProtocolPrefix(protocol.ID(config.DHTPrefix)),
		dht.NamespacedValidator(strings.ReplaceAll(config.CustomNamespace, "/", ""), dhtValidator{PS: ps, customNamespace: config.CustomNamespace}),
		dht.Mode(dht.ModeAutoServer),
	}

	// set up the resource manager
	mem := int64(config.Memory)
	if mem > 0 {
		mem = 1024 * 1024 * mem
	} else {
		mem = 1024 * 1024 * 1024 // 1GB
	}

	fds := config.FileDescriptors
	if fds == 0 {
		fds = 512
	}

	limits := rcmgr.DefaultLimits
	limits.SystemBaseLimit.ConnsInbound = 512
	limits.SystemBaseLimit.ConnsOutbound = 512
	limits.SystemBaseLimit.Conns = 1024
	limits.SystemBaseLimit.StreamsInbound = 8192
	limits.SystemBaseLimit.StreamsOutbound = 8192
	limits.SystemBaseLimit.Streams = 16384
	scaled := limits.Scale(mem, fds)

	log.Infow("libp2p_limits",
		"labels", string(observability.LabelNode),
		"limits", scaled,
	)

	mgr, err := rcmgr.NewResourceManager(rcmgr.NewFixedLimiter(scaled))
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	// get quic port from listen address
	quicPort := 0
	hasQUICPort := false
	for _, addr := range config.ListenAddress {
		maddr := ma.StringCast(addr)
		maddrComps := ma.Split(maddr)
		for _, comp := range maddrComps {
			if comp.Protocol().Code == ma.P_UDP {
				quicPort, err = strconv.Atoi(comp.Value())
				if err != nil {
					log.Errorf("failed to parse QUIC port from address %s: %v", addr, err)
					return nil, nil, nil, nil, nil, fmt.Errorf("failed to parse QUIC port from address %s: %v", addr, err)
				}
				log.Infof("QUIC port found in address %s: %d", addr, quicPort)
				hasQUICPort = true
				break
			}
		}
	}

	// quic port must be set
	if !hasQUICPort {
		log.Errorf("QUIC port not found in listen addresses")
		return nil, nil, nil, nil, nil, fmt.Errorf("QUIC port not found in listen addresses")
	}

	udpConn, err := net.ListenUDP("udp4", &net.UDPAddr{IP: net.IPv4zero, Port: quicPort})
	if err != nil {
		return nil, nil, nil, nil, nil, fmt.Errorf("failed to create UDP listener: %w", err)
	}
	rqtr := NewRawQUICTransport(udpConn)
	newReuse := func(statelessResetKey quic.StatelessResetKey, tokenGeneratorKey quic.TokenGeneratorKey) (*quicreuse.ConnManager, error) {
		reuseConnM, err := quicreuse.NewConnManager(
			statelessResetKey,
			tokenGeneratorKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create reuse: %w", err)
		}

		trDone, err := reuseConnM.LendTransport("udp4", rqtr, udpConn)
		if err != nil {
			return nil, fmt.Errorf("failed to add transport to reuse: %w", err)
		}

		go func() {
			// wait for the connection manager to be done to close the raw quic transport
			<-trDone
			log.Info("closing raw quic transport")
			rqtr.Close()
		}()

		return reuseConnM, nil
	}

	libp2pOpts = append(libp2pOpts,
		libp2p.ListenAddrStrings(config.ListenAddress...),
		libp2p.ResourceManager(mgr),
		libp2p.Identity(config.PrivateKey),
		libp2p.Routing(func(h host.Host) (routing.PeerRouting, error) {
			idht, err = dht.New(ctx, h, dhtOpts...)
			return idht, err
		}),
		libp2p.Peerstore(ps),
		libp2p.Security(libp2ptls.ID, libp2ptls.New),
		libp2p.Security(noise.ID, noise.New),
		libp2p.ChainOptions(
			libp2p.Transport(tcp.NewTCPTransport),
			libp2p.Transport(libp2pquic.NewTransport),
			libp2p.Transport(webtransport.New),
			libp2p.Transport(ws.New),
		),
		libp2p.ConnectionManager(connmgr),
		libp2p.EnableNATService(),
		libp2p.EnableAutoNATv2(),
		libp2p.EnableRelay(),
		libp2p.EnableRelayService(
			relay.WithLimit(&relay.RelayLimit{
				Duration: 5 * time.Minute,
				Data:     1 << 21, // 2 MiB
			}),
		),
		// TODO debug: disable relay
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
		libp2p.EnableHolePunching(holepunch.WithAddrFilter(&quicAddrFilter{})),
		libp2p.QUICReuse(newReuse),
	)

	if config.Server {
		libp2pOpts = append(libp2pOpts, libp2p.AddrsFactory(makeAddrsFactory([]string{}, []string{}, defaultServerFilters)))
		libp2pOpts = append(libp2pOpts, libp2p.ConnectionGater((*filtersConnectionGater)(filter)))
	}

	host, err := libp2p.New(libp2pOpts...)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}

	go watchForNewPeers(ctx, host, newPeer)

	optsPS := []pubsub.Option{
		pubsub.WithFloodPublish(true),
		pubsub.WithMessageSigning(true),
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
		pubsub.WithStrictSignatureVerification(true),
	}
	if config.GossipMaxMessageSize > 0 {
		optsPS = append(optsPS, pubsub.WithMaxMessageSize(config.GossipMaxMessageSize))
	}
	gossip, err := pubsub.NewGossipSub(ctx, host, optsPS...)
	// gossip, err := pubsub.NewGossipSubWithRouter(ctx, host, pubsub.DefaultGossipSubRouter(host), optsPS...)
	if err != nil {
		return nil, nil, nil, nil, nil, err
	}
	return host, idht, gossip, udpConn, rqtr, nil
}

func watchForNewPeers(ctx context.Context, host host.Host, newPeer chan peer.AddrInfo) {
	sub, err := host.EventBus().Subscribe([]interface{}{
		&event.EvtPeerIdentificationCompleted{},
	})
	if err != nil {
		log.Errorw("failed to subscribe to peer identification events",
			"labels", string(observability.LabelNode),
			"error", err,
		)
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
			var identPeer peer.AddrInfo
			identPeer.ID = ev.Peer
			copy(identPeer.Addrs, ev.ListenAddrs)
			go handleNewPeers(ctx, identPeer, newPeer)
		}
	}
}

func handleNewPeers(ctx context.Context, identifiedPeer peer.AddrInfo, newPeer chan peer.AddrInfo) {
	select {
	case <-ctx.Done():
		return
	default:
		var publicAddrs []ma.Multiaddr
		for _, addr := range identifiedPeer.Addrs {
			if manet.IsPublicAddr(addr) {
				publicAddrs = append(publicAddrs, addr)
			}
		}
		if len(publicAddrs) > 0 {
			newPeer <- peer.AddrInfo{ID: identifiedPeer.ID, Addrs: publicAddrs}
		}
	}
}
