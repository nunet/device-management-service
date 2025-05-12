// memory_testnet.go — in-memory implementation of network.Network.
// Meant only for tests: no libp2p, no sockets, no goroutine leaks.

package network

import (
	"context"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/libp2p/go-libp2p/core/peer"

	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	commonproto "gitlab.com/nunet/device-management-service/proto/generated/v1/common"
	"gitlab.com/nunet/device-management-service/types"
)

// NewMemoryNetwork returns a fresh, fully-initialised test network.
func NewMemoryNetwork() *MemoryNet {
	id := peer.ID(uuid.New().String())

	m := &MemoryNet{
		hostID:         id,
		msgHandlers:    map[string]func([]byte, peer.ID){},
		subs:           map[string]map[uint64]func([]byte){},
		advertisements: map[string][]byte{},
		peers:          map[PeerID]time.Time{},
		subnets:        map[string]*subnetInfo{},
		notifyCbs:      notifyCallbacks{},
		startTime:      time.Now(),
	}
	// register ourselves so Ping(hostID) succeeds
	m.peers[m.hostID] = time.Now()
	return m
}

type subnetInfo struct {
	routing map[string]string            // host → ip
	peers   map[string]string            // peerID → ip
	dns     map[string]string            // name → ip
	portMap map[string]map[string]string // “srcIP:srcPort” → proto → dst
}

type notifyCallbacks struct {
	pre  func(PeerID, []ProtocolID, int)
	conn func(PeerID)
	disc func(PeerID)
	iden func(PeerID, []ProtocolID)
	upd  func(PeerID, []ProtocolID)
}

// MemoryNet satisfies network.Network and is 100 % in-memory / goroutine-safe.
type MemoryNet struct {
	hostID PeerID

	mx sync.RWMutex

	// messages & pubsub
	msgHandlers map[string]func([]byte, peer.ID)
	subs        map[string]map[uint64]func([]byte)
	subCount    uint64

	// discovery / statistics / ping
	peers     map[PeerID]time.Time // last-seen (touch via Notify if you like)
	startTime time.Time

	// advertisements
	advertisements map[string][]byte

	// subnets
	subnets map[string]*subnetInfo

	// callbacks
	notifyCbs notifyCallbacks
}

// Messenger (point-to-point)
// SendMessage delivers an envelope to the handler registered for its type.
func (m *MemoryNet) SendMessage(
	_ context.Context,
	hostID string,
	env types.MessageEnvelope,
	_ time.Time,
) error {
	m.mx.RLock()
	h, ok := m.msgHandlers[string(env.Type)]
	m.mx.RUnlock()
	if !ok {
		return errors.New("no handler registered for message type")
	}

	// Forward the payload to the handler, converting hostID → peer.ID.
	h(env.Data, peer.ID(hostID))
	return nil
}

// SendMessageSync simply forwards to SendMessage.
func (m *MemoryNet) SendMessageSync(
	ctx context.Context,
	hostID string,
	env types.MessageEnvelope,
	exp time.Time,
) error {
	return m.SendMessage(ctx, hostID, env, exp)
}

// Lifecycle / statistics / ping
func (m *MemoryNet) Init(*config.Config) error { return nil }
func (m *MemoryNet) Start() error              { return nil }
func (m *MemoryNet) Stop() error               { return nil }

func (m *MemoryNet) Stat() types.NetworkStats {
	m.mx.RLock()
	defer m.mx.RUnlock()
	return types.NetworkStats{
		ID:         string(m.hostID),
		ListenAddr: "memory://" + string(m.hostID),
	}
}

func (m *MemoryNet) Ping(_ context.Context, id string, _ time.Duration) (types.PingResult, error) {
	m.mx.RLock()
	last, ok := m.peers[PeerID(id)]
	m.mx.RUnlock()

	if !ok {
		return types.PingResult{Success: false}, nil
	}
	return types.PingResult{Success: true, RTT: time.Since(last)}, nil
}

func (m *MemoryNet) GetHostID() PeerID                  { return m.hostID }
func (m *MemoryNet) GetPeerPubKey(PeerID) crypto.PubKey { return nil }

// Stream handlers
func (m *MemoryNet) HandleMessage(t string, h func([]byte, peer.ID)) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	m.msgHandlers[t] = h
	return nil
}

func (m *MemoryNet) UnregisterMessageHandler(t string) {
	m.mx.Lock()
	defer m.mx.Unlock()
	delete(m.msgHandlers, t)
}

// Rendez-vous (Advertise / Query)
func (m *MemoryNet) Advertise(_ context.Context, k string, data []byte) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	m.advertisements[k] = data
	return nil
}

func (m *MemoryNet) Unadvertise(_ context.Context, k string) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	delete(m.advertisements, k)
	return nil
}

func (m *MemoryNet) Query(_ context.Context, k string) ([]*commonproto.Advertisement, error) {
	m.mx.RLock()
	defer m.mx.RUnlock()
	data, ok := m.advertisements[k]
	if !ok {
		return nil, nil
	}
	return []*commonproto.Advertisement{
		{
			PeerId:    string(m.hostID),
			Timestamp: time.Now().UnixNano(),
			Data:      data,
		},
	}, nil
}

// PubSub
func (m *MemoryNet) Publish(_ context.Context, topic string, data []byte) error {
	m.mx.RLock()
	defer m.mx.RUnlock()
	for _, cb := range m.subs[topic] {
		cb := cb // copy for goroutine capture
		go cb(data)
	}
	return nil
}

func (m *MemoryNet) Subscribe(_ context.Context, topic string, cb func([]byte),
	_ Validator,
) (uint64, error) {
	m.mx.Lock()
	defer m.mx.Unlock()
	m.subCount++
	if m.subs[topic] == nil {
		m.subs[topic] = map[uint64]func([]byte){}
	}
	m.subs[topic][m.subCount] = cb
	return m.subCount, nil
}

func (m *MemoryNet) Unsubscribe(topic string, id uint64) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	if subs, ok := m.subs[topic]; ok {
		delete(subs, id)
		if len(subs) == 0 {
			delete(m.subs, topic)
		}
	}
	return nil
}

// Broadcast scoring (unused in tests – stubbed)
func (m *MemoryNet) SetupBroadcastTopic(string, func(*Topic) error) error { return nil }
func (m *MemoryNet) SetBroadcastAppScore(func(PeerID) float64)            {}
func (m *MemoryNet) GetBroadcastScore() map[PeerID]*PeerScoreSnapshot     { return nil }

// Notify

func (m *MemoryNet) Notify(_ context.Context,
	pre func(PeerID, []ProtocolID, int),
	conn func(PeerID),
	disc func(PeerID),
	iden func(PeerID, []ProtocolID),
	upd func(PeerID, []ProtocolID),
) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	m.notifyCbs = notifyCallbacks{pre, conn, disc, iden, upd}
	return nil
}

func (m *MemoryNet) PeerConnected(p PeerID) bool {
	m.mx.RLock()
	_, ok := m.peers[p]
	m.mx.RUnlock()
	return ok
}

// Subnets
func (m *MemoryNet) CreateSubnet(_ context.Context, id string, rt map[string]string) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	if _, ok := m.subnets[id]; ok {
		return errors.New("subnet exists")
	}
	m.subnets[id] = &subnetInfo{
		routing: rt,
		peers:   map[string]string{},
		dns:     map[string]string{},
		portMap: map[string]map[string]string{},
	}
	return nil
}

func (m *MemoryNet) DestroySubnet(id string) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	delete(m.subnets, id)
	return nil
}

func (m *MemoryNet) AddSubnetPeer(id, peerID, ip string) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	if sn, ok := m.subnets[id]; ok {
		sn.peers[peerID] = ip
	}
	return nil
}

func (m *MemoryNet) RemoveSubnetPeer(id, peerID, _ string) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	if sn, ok := m.subnets[id]; ok {
		delete(sn.peers, peerID)
	}
	return nil
}

func (m *MemoryNet) AcceptSubnetPeer(id, peerID, ip string) error {
	return m.AddSubnetPeer(id, peerID, ip)
}

func (m *MemoryNet) MapPort(id, proto, srcIP, srcPort, dstIP, dstPort string) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	if sn, ok := m.subnets[id]; ok {
		key := net.JoinHostPort(srcIP, srcPort)
		if sn.portMap[key] == nil {
			sn.portMap[key] = map[string]string{}
		}
		sn.portMap[key][proto] = net.JoinHostPort(dstIP, dstPort)
	}
	return nil
}

func (m *MemoryNet) UnmapPort(id, proto, srcIP, srcPort, _, _ string) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	if sn, ok := m.subnets[id]; ok {
		key := net.JoinHostPort(srcIP, srcPort)
		if pm, ok := sn.portMap[key]; ok {
			delete(pm, proto)
			if len(pm) == 0 {
				delete(sn.portMap, key)
			}
		}
	}
	return nil
}

func (m *MemoryNet) AddSubnetDNSRecords(id string, rec map[string]string) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	if sn, ok := m.subnets[id]; ok {
		for k, v := range rec {
			sn.dns[k] = v
		}
	}
	return nil
}

func (m *MemoryNet) RemoveSubnetDNSRecord(id, name string) error {
	m.mx.Lock()
	defer m.mx.Unlock()
	if sn, ok := m.subnets[id]; ok {
		delete(sn.dns, name)
	}
	return nil
}

// Misc
func (m *MemoryNet) ResolveAddress(context.Context, string) ([]string, error) { return nil, nil }
func (m *MemoryNet) GetPeerIP(PeerID) string                                  { return "" }
func (m *MemoryNet) HostPublicIP() (net.IP, error)                            { return nil, nil }
