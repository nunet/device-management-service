// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package libp2p

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	dht_pb "github.com/libp2p/go-libp2p-kad-dht/pb"
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"
	"github.com/libp2p/go-libp2p/core/protocol"
	msgio "github.com/libp2p/go-msgio"
	"github.com/libp2p/go-msgio/protoio" //nolint:staticcheck
	multiaddr "github.com/multiformats/go-multiaddr"
	"google.golang.org/protobuf/proto"

	dmscrypto "gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/observability"
	commonproto "gitlab.com/nunet/device-management-service/proto/generated/v1/common"
	"gitlab.com/nunet/device-management-service/types"
)

const kadv1 = "/kad/1.0.0"

var (
	ErrInvalidKeyNamespace     = errors.New("invalid key namespace")
	ErrValidateEnvelopeByPbkey = errors.New("failed to verify public key")
)

// Connect to Bootstrap nodes
func (l *Libp2p) connectToBootstrapNodes(ctx context.Context) error {
	// bootstrap all nodes at the same time.
	if len(l.config.BootstrapPeers) > 0 {
		var wg sync.WaitGroup
		connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		for _, addr := range l.config.BootstrapPeers {
			wg.Add(1)
			go func(peerAddr multiaddr.Multiaddr) {
				defer wg.Done()
				addrInfo, err := peer.AddrInfoFromP2pAddr(peerAddr)
				if err != nil {
					log.Errorf("failed to convert multi addr to addr info %v - %v", peerAddr, err)
					return
				}
				if err := l.Host.Connect(connectCtx, *addrInfo); err != nil {
					log.Errorf("failed to connect to bootstrap node %s - %v", addrInfo.ID.String(), err)
				} else {
					log.Infof("connected to Bootstrap Node %s", addrInfo.ID.String())
				}
			}(addr)
		}
		wg.Wait()
	}
	return nil
}

// Start dht bootstrapper
func (l *Libp2p) bootstrapDHT(ctx context.Context) error {
	endSpan := observability.StartSpan(ctx, "libp2p_bootstrap")
	defer endSpan()

	if err := l.DHT.Bootstrap(ctx); err != nil {
		log.Errorw("libp2p_bootstrap_failure",
			"labels", string(observability.LabelNode),
			"error", err)
		return err
	}

	log.Infow("libp2p_bootstrap_success",
		"labels", string(observability.LabelNode))
	return nil
}

// startRandomWalk starts a background process that crawls the dht by resolving random keys.
func (l *Libp2p) startRandomWalk(ctx context.Context) {
	go func() {
		log.Debug("starting bootstrap process")
		// A simple mechanism to improve our botostrap and peer discovery:
		// 1. initiate a background, never ending, random walk which tries to resolve
		// random keys in the dht and by extension discovers other peers.

		interval := 5 * time.Minute
		delayOnError := 10 * time.Second
		time.Sleep(interval) // wait for dht ready

		dhtProto := protocol.ID(l.config.DHTPrefix + kadv1)
		sender := newDHTMessageSender(l.Host, dhtProto)
		messenger, err := dht_pb.NewProtocolMessenger(sender)
		if err != nil {
			log.Errorf("bootstrap: creating protocol messenger: %s", err)
			return
		}

		var depth int
		var key string
		for {
			select {
			case <-ctx.Done():
				log.Debugf("bootstrap: context done, stopping bootstrap")
				return
			default:
				randomPeerID, err := l.DHT.RoutingTable().GenRandPeerID(0)
				if err != nil {
					log.Debugf("bootstrap: failed to generate random peer ID: %s", err)
					continue
				}
				key = randomPeerID.String()

				log.Debugf("bootstrap: crawling from %s", key)
				peers, err := l.DHT.GetClosestPeers(ctx, key)
				if err != nil {
					log.Debugf("bootstrap: failed to get closest peers with key=%s - error: %s", randomPeerID.String(), err)
					time.Sleep(delayOnError)
					delayOnError = time.Duration(float64(delayOnError) * 1.25)
					if delayOnError > 5*time.Minute {
						delayOnError = 5 * time.Minute
					}
					continue
				}
				delayOnError = 10 * time.Second

				if len(peers) == 0 {
					continue
				}

				peerID := peers[rand.Intn(len(peers))] //nolint:gosec
				if peerID == l.Host.ID() {
					log.Debugf("bootstrap: skipping self")
					continue
				}
				log.Debugf("bootstrap: starting random walk from %s", peerID)

				peerAddrInfo, err := l.resolvePeerAddress(ctx, peerID)
				if err != nil {
					log.Debugf("bootstrap: failed to resolve address for peer %s - %v", peerID, err)
					continue
				}

				var peerInfos []*peer.AddrInfo
				selected := &peerAddrInfo

			crawl:
				log.Debugf("bootstrap: crawling %s", selected.ID)
				if err := l.Host.Connect(ctx, *selected); err != nil {
					log.Debugf("bootstrap: failed to connect to peer %s: %s", peerID, err)
					depth++
					continue
				}

				peerInfos, err = messenger.GetClosestPeers(ctx, selected.ID, randomPeerID)
				if err != nil {
					log.Debugf("bootstrap: failed to get closest peers from %s: %s", selected.ID, err)
					depth++
					continue
				}

				if len(peerInfos) == 0 {
					depth++
					continue
				}

				selected = peerInfos[rand.Intn(len(peerInfos))] //nolint:gosec
				if selected.ID == l.Host.ID() {
					log.Debugf("bootstrap: skipping self")
					depth++
					continue
				}

				if depth < 20 {
					randomPeerID, err = l.DHT.RoutingTable().GenRandPeerID(0)
					if err != nil {
						log.Debugf("bootstrap: failed to generate random peer ID: %s", err)
						goto cooldown
					}

					depth++
					goto crawl
				}

				// cooldown
			cooldown:
				depth = 0
				minDelay := interval / 2
				maxDelay := (3 * interval) / 2
				delay := minDelay + time.Duration(rand.Int63n(int64(maxDelay-minDelay))) //nolint:gosec
				log.Debugf("bootstrap: cooling down for %s", delay)
				select {
				case <-time.After(delay):
				case <-ctx.Done():
					return
				}

				interval = interval * 3 / 2
				if interval > 4*time.Hour {
					interval = 4 * time.Hour
				}
			}
		}
	}()
}

type dhtValidator struct {
	PS              peerstore.Peerstore
	customNamespace string
}

// Validate validates an item placed into the dht.
func (d dhtValidator) Validate(key string, value []byte) error {
	endSpan := observability.StartSpan("libp2p_dht_validate")
	defer endSpan()

	// empty value is considered deleting an item from the dht
	if len(value) == 0 {
		log.Infow("libp2p_dht_validate_success",
			"labels", string(observability.LabelNode),
			"key", key)
		return nil
	}

	if !strings.HasPrefix(key, d.customNamespace) {
		log.Errorw("libp2p_dht_validate_failure",
			"labels", string(observability.LabelNode),
			"key", key,
			"error", "invalid key namespace")
		return ErrInvalidKeyNamespace
	}

	// verify signature
	var envelope commonproto.Advertisement
	err := proto.Unmarshal(value, &envelope)
	if err != nil {
		log.Errorw("libp2p_dht_validate_failure",
			"labels", string(observability.LabelNode),
			"key", key,
			"error", fmt.Sprintf("failed to unmarshal envelope: %v", err))
		return fmt.Errorf("%w envelope: %w", types.ErrUnmarshal, err)
	}

	pubKey, err := crypto.UnmarshalEd25519PublicKey(envelope.PublicKey)
	if err != nil {
		log.Errorw("libp2p_dht_validate_failure",
			"labels", string(observability.LabelNode),
			"key", key,
			"error", fmt.Sprintf("failed to unmarshal public key: %v", err))
		return fmt.Errorf("%w: %w", dmscrypto.ErrUnmarshalPublicKey, err)
	}

	concatenatedBytes := bytes.Join([][]byte{
		[]byte(envelope.PeerId),
		{byte(envelope.Timestamp)},
		envelope.Data,
		envelope.PublicKey,
	}, nil)
	ok, err := pubKey.Verify(concatenatedBytes, envelope.Signature)
	if err != nil {
		log.Errorw("libp2p_dht_validate_failure",
			"labels", string(observability.LabelNode),
			"key", key,
			"error", fmt.Sprintf("failed to verify envelope: %v", err))
		return fmt.Errorf("%w envelope: %w", ErrValidateEnvelopeByPbkey, err)
	}

	if !ok {
		log.Errorw("libp2p_dht_validate_failure",
			"labels", string(observability.LabelNode),
			"key", key,
			"error", "public key didn't sign the payload")
		return fmt.Errorf("%w, public key didn't sign payload", ErrValidateEnvelopeByPbkey)
	}

	log.Infow("libp2p_dht_validate_success",
		"labels", string(observability.LabelNode),
		"key", key)
	return nil
}

func (dhtValidator) Select(_ string, _ [][]byte) (int, error) { return 0, nil }

type dhtMessenger struct {
	host  host.Host
	proto protocol.ID
}

func newDHTMessageSender(h host.Host, proto protocol.ID) dht_pb.MessageSender {
	return &dhtMessenger{host: h, proto: proto}
}

func (m *dhtMessenger) SendRequest(ctx context.Context, p peer.ID, msg *dht_pb.Message) (*dht_pb.Message, error) {
	s, err := m.host.NewStream(ctx, p, m.proto)
	if err != nil {
		return nil, fmt.Errorf("open stream: %w", err)
	}
	defer s.Close()

	wr := protoio.NewDelimitedWriter(s)
	if err := wr.WriteMsg(msg); err != nil {
		_ = s.Reset()
		return nil, fmt.Errorf("write message: %w", err)
	}

	r := msgio.NewVarintReaderSize(s, network.MessageSizeMax)
	bytes, err := r.ReadMsg()
	if err != nil {
		_ = s.Reset()
		return nil, fmt.Errorf("read message: %w", err)
	}
	defer r.ReleaseMsg(bytes)

	reply := new(dht_pb.Message)
	if err := proto.Unmarshal(bytes, reply); err != nil {
		_ = s.Reset()
		return nil, fmt.Errorf("unmarshal message: %w", err)
	}

	return reply, nil
}

func (m *dhtMessenger) SendMessage(ctx context.Context, p peer.ID, msg *dht_pb.Message) error {
	s, err := m.host.NewStream(ctx, p, m.proto)
	if err != nil {
		return fmt.Errorf("open stream: %w", err)
	}
	defer s.Close()

	wr := protoio.NewDelimitedWriter(s)
	if err := wr.WriteMsg(msg); err != nil {
		_ = s.Reset()
		return fmt.Errorf("write message: %w", err)
	}

	return s.CloseWrite()
}
