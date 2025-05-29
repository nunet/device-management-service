// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

// vnet.go — virtual network of in-memory hosts
// Meant only for tests: no libp2p, no sockets, no goroutine leaks.

package network

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"

	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	common "gitlab.com/nunet/device-management-service/proto/generated/v1/common"
	"gitlab.com/nunet/device-management-service/types"
)

// Substrate (shared “world”) where each host has its own view of
// the network. Actions are usually done as: hostAlice gets
// hostBob handler for message XXX, and call the handler.
//
// The only global state is the dht for simplicity purposes.
// And the globalPeers which is used only for connecting new hosts.
//
// Improvements:
// - Make use of real pub-priv key pair and real peerIDs
// - Some methods should be more realistic: HostPublicIP(),
// ResolvePeerAddress(), GetPeerIP()...
// - Notify() implementation may be necessary
type Substrate struct {
	mx sync.RWMutex

	dht         map[string]map[string][]byte // key -> peerID -> value
	globalPeers map[string]*MemoryHost       // used only for connecting new hosts
}

// MemoryHost — implements Network, delegates to Substrate
type MemoryHost struct {
	pid       peer.ID
	substrate *Substrate

	// local state
	mx          sync.RWMutex
	peers       map[string]*MemoryHost
	msgHandlers map[string]func([]byte, peer.ID)
	subs        map[string]map[uint64]func([]byte)
	score       map[string]*PeerScoreSnapshot
	nextSubID   uint64
}

var _ Network = (*MemoryHost)(nil)

func NewSubstrate() *Substrate {
	return &Substrate{
		dht:         map[string]map[string][]byte{},
		globalPeers: map[string]*MemoryHost{},
	}
}

// AddWiredPeer adds and returns a host to the substrate connected
// to all existent peers
func (substrate *Substrate) AddWiredPeer(id peer.ID) Network {
	return substrate.AddPeer(id, true)
}

// AddPeer returns a Network implementation bound to this substrate.
// If forceConnection is true, the new peer connects to all existing peers.
func (substrate *Substrate) AddPeer(id peer.ID, forceConnection bool) Network {
	host := &MemoryHost{
		pid:         id,
		substrate:   substrate,
		peers:       map[string]*MemoryHost{},
		msgHandlers: map[string]func([]byte, peer.ID){},
		subs:        map[string]map[uint64]func([]byte){},
		score:       map[string]*PeerScoreSnapshot{},
		nextSubID:   0,
	}

	substrate.mx.Lock()
	defer substrate.mx.Unlock()
	if forceConnection {
		// Connect to all existing peers
		for peerID, existingHost := range substrate.globalPeers {
			if existingHost != nil &&
				peerID != id.String() {
				connectPeers(host, existingHost)
			}
		}
	} else { //nolint
		// TODO: implement connecting to x random peers when forceConnection is false
	}

	substrate.globalPeers[id.String()] = host

	return host
}

// connectPeers establishes a bidirectional connection between two peers
func connectPeers(bob, alice *MemoryHost) {
	bob.mx.Lock()
	bob.peers[alice.pid.String()] = alice
	bob.mx.Unlock()

	alice.mx.Lock()
	alice.peers[bob.pid.String()] = bob
	alice.mx.Unlock()
}

// NewMemoryNetHost is a substrate wrapper that simply
// returns a host who is the only participant of the network.
//
// Useful for tests that don't need conn between peers but
// only a single instance of Network
func NewMemoryNetHost() (Network, error) {
	substrate := NewSubstrate()
	_, pubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	if err != nil {
		return nil, err
	}
	peerID, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return nil, err
	}
	return substrate.AddPeer(peerID, true), nil
}

// Messenger
func (h *MemoryHost) SendMessage(
	_ context.Context,
	hostID string,
	env types.MessageEnvelope,
	_ time.Time,
) error {
	// TODO: instead of checking self explicitly
	// solve the mutex locks
	if h.pid.String() == hostID {
		handler, ok := h.msgHandlers[string(env.Type)]
		if !ok {
			return errors.New("virtual: no handler for msgType")
		}
		handler(env.Data, h.pid)
		return nil
	}

	h.mx.RLock()
	targetHost, ok := h.peers[hostID]
	h.mx.RUnlock()
	if !ok {
		return errors.New("virtual: peer not connected")
	}

	targetHost.mx.RLock()
	handler, ok := targetHost.msgHandlers[string(env.Type)]
	targetHost.mx.RUnlock()
	if !ok {
		return errors.New("virtual: no handler for msgType")
	}

	handler(env.Data, h.pid)
	return nil
}

func (h *MemoryHost) SendMessageSync(
	ctx context.Context,
	host string,
	env types.MessageEnvelope,
	exp time.Time,
) error {
	return h.SendMessage(ctx, host, env, exp)
}

// Handlers
func (h *MemoryHost) HandleMessage(msgType string, handler func([]byte, peer.ID)) error {
	h.mx.Lock()
	defer h.mx.Unlock()

	h.msgHandlers[msgType] = handler
	return nil
}

func (h *MemoryHost) UnregisterMessageHandler(msgType string) {
	h.mx.Lock()
	defer h.mx.Unlock()
	delete(h.msgHandlers, msgType)
}

// PubSub
func (h *MemoryHost) Publish(_ context.Context, topic string, data []byte) error {
	for _, destPeer := range h.peers {
		destPeer.mx.RLock()
		subs := destPeer.subs[topic]
		destPeer.mx.RUnlock()

		for _, cb := range subs {
			go cb(data)
		}
	}
	return nil
}

func (h *MemoryHost) Subscribe(
	_ context.Context,
	topic string,
	cb func([]byte),
	_ Validator,
) (uint64, error) {
	h.mx.Lock()
	defer h.mx.Unlock()

	h.nextSubID++
	if h.subs[topic] == nil {
		h.subs[topic] = map[uint64]func([]byte){}
	}
	h.subs[topic][h.nextSubID] = cb
	return h.nextSubID, nil
}

func (h *MemoryHost) Unsubscribe(topic string, id uint64) error {
	h.mx.Lock()
	defer h.mx.Unlock()
	if m := h.subs[topic]; m != nil {
		delete(m, id)
		if len(m) == 0 {
			delete(h.subs, topic)
		}
	}
	return nil
}

// DHT lookups
func (h *MemoryHost) Advertise(_ context.Context, k string, d []byte) error {
	h.substrate.mx.Lock()
	defer h.substrate.mx.Unlock()

	if h.substrate.dht[k] == nil {
		h.substrate.dht[k] = map[string][]byte{}
	}
	h.substrate.dht[k][h.pid.String()] = d
	return nil
}

func (h *MemoryHost) Unadvertise(_ context.Context, k string) error {
	h.substrate.mx.Lock()
	defer h.substrate.mx.Unlock()

	if h.substrate.dht[k] != nil {
		delete(h.substrate.dht[k], h.pid.String())
		if len(h.substrate.dht[k]) == 0 {
			delete(h.substrate.dht, k)
		}
	}
	return nil
}

func (h *MemoryHost) Query(_ context.Context, k string) ([]*common.Advertisement, error) {
	h.substrate.mx.RLock()
	defer h.substrate.mx.RUnlock()

	peers, ok := h.substrate.dht[k]
	if !ok {
		return nil, nil
	}

	ads := make([]*common.Advertisement, 0, len(peers))
	for peerID, data := range peers {
		ads = append(ads, &common.Advertisement{
			PeerId:    peerID,
			Timestamp: time.Now().UnixNano(),
			Data:      data,
		})
	}
	return ads, nil
}

func (h *MemoryHost) Ping(_ context.Context, id string, _ time.Duration) (types.PingResult, error) {
	h.mx.RLock()
	_, ok := h.peers[id]
	h.mx.RUnlock()

	if !ok {
		return types.PingResult{Success: false}, nil
	}
	return types.PingResult{Success: true, RTT: time.Millisecond}, nil
}

func (h *MemoryHost) GetHostID() peer.ID { return h.pid }

func (h *MemoryHost) GetPeerPubKey(_ peer.ID) crypto.PubKey {
	// TODO: I think memoryHost should make use of real pub-pvkey-peerIDs
	_, pubKey, _ := crypto.GenerateKeyPair(crypto.Ed25519)
	return pubKey
}

func (h *MemoryHost) Stop() error {
	h.substrate.mx.Lock()
	delete(h.substrate.globalPeers, h.pid.String())
	h.substrate.mx.Unlock()

	// Remove this host from all other hosts' peer lists
	h.mx.Lock()
	for _, anotherHost := range h.peers {
		anotherHost.mx.Lock()
		delete(anotherHost.peers, h.pid.String())
		anotherHost.mx.Unlock()
	}
	h.mx.Unlock()

	return nil
}

func (h *MemoryHost) Stat() types.NetworkStats {
	return types.NetworkStats{ID: h.pid.String(), ListenAddr: "virtual://" + string(h.pid)}
}

func (h *MemoryHost) PeerConnected(p peer.ID) bool {
	h.mx.RLock()
	defer h.mx.RUnlock()
	_, ok := h.peers[p.String()]
	return ok
}

// Misc stubs
func (*MemoryHost) Init(*config.Config) error { return nil }

func (*MemoryHost) Start() error { return nil }

func (*MemoryHost) ResolveAddress(context.Context, string) ([]string, error) { return nil, nil }

func (*MemoryHost) SetupBroadcastTopic(string, func(*Topic) error) error { return nil }

func (*MemoryHost) SetBroadcastAppScore(func(peer.ID) float64) {}

func (h *MemoryHost) GetBroadcastScore() map[peer.ID]*PeerScoreSnapshot {
	h.mx.RLock()
	defer h.mx.RUnlock()
	// Return a copy to avoid race conditions
	scores := make(map[peer.ID]*PeerScoreSnapshot)
	for k, v := range h.score {
		id, err := peer.Decode(k)
		if err != nil {
			continue
		}
		scores[id] = v
	}
	return scores
}

func (*MemoryHost) Notify(context.Context,
	func(peer.ID, []ProtocolID, int),
	func(peer.ID),
	func(peer.ID),
	func(peer.ID, []ProtocolID),
	func(peer.ID, []ProtocolID),
) error {
	return nil
}

func (*MemoryHost) GetPeerIP(peer.ID) string { return "" }

func (*MemoryHost) HostPublicIP() (net.IP, error) { return nil, nil }

func (*MemoryHost) CreateSubnet(context.Context, string, map[string]string) error { return nil }

func (*MemoryHost) DestroySubnet(string) error { return nil }

func (*MemoryHost) AddSubnetPeer(string, string, string) error { return nil }

func (*MemoryHost) RemoveSubnetPeer(string, string, string) error { return nil }

func (*MemoryHost) AcceptSubnetPeer(string, string, string) error { return nil }

func (*MemoryHost) MapPort(string, string, string, string, string, string) error { return nil }

func (*MemoryHost) UnmapPort(string, string, string, string, string, string) error { return nil }

func (*MemoryHost) AddSubnetDNSRecords(string, map[string]string) error { return nil }

func (*MemoryHost) RemoveSubnetDNSRecord(string, string) error { return nil }
