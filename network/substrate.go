// substrate.go — virtual multi-peer network inside the *network* package.
// No libp2p / sockets: ideal for tests with multiple logical peers.

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

// Latency model (pluggable)
type LatencyModel interface {
	RTT(src, dst PeerID) time.Duration
}
type zeroLatency struct{}

func (zeroLatency) RTT(_, _ PeerID) time.Duration { return 0 }

// Substrate (shared “world”)
type Substrate struct {
	mx   sync.RWMutex
	next uint64

	lat         LatencyModel
	msgHandlers map[string]func([]byte, PeerID)
	subs        map[string]map[uint64]func([]byte)
	advert      map[string][]byte
	peers       map[PeerID]time.Time
	score       map[PeerID]*PeerScoreSnapshot
}

func NewSubstrate() *Substrate {
	return &Substrate{
		lat:         zeroLatency{},
		msgHandlers: map[string]func([]byte, PeerID){},
		subs:        map[string]map[uint64]func([]byte){},
		advert:      map[string][]byte{},
		peers:       map[PeerID]time.Time{},
		score:       map[PeerID]*PeerScoreSnapshot{},
	}
}

// MakeNetwork returns a Network implementation bound to this substrate.
func (s *Substrate) MakeNetwork(id PeerID) Network {
	s.mx.Lock()
	s.peers[id] = time.Now()
	s.mx.Unlock()
	return &virtualNet{pid: id, s: s}
}

// virtualNet — implements Network, delegates to Substrate
type virtualNet struct {
	pid PeerID
	s   *Substrate
}

// Messenger
func (v *virtualNet) SendMessage(
	_ context.Context,
	host string,
	env types.MessageEnvelope,
	_ time.Time,
) error {
	v.s.mx.RLock()
	h, ok := v.s.msgHandlers[string(env.Type)]
	v.s.mx.RUnlock()
	if !ok {
		return errors.New("virtual: no handler for msgType")
	}
	h(env.Data, PeerID(host))
	return nil
}

func (v *virtualNet) SendMessageSync(
	ctx context.Context,
	host string,
	env types.MessageEnvelope,
	exp time.Time,
) error {
	return v.SendMessage(ctx, host, env, exp)
}

// Handlers
func (v *virtualNet) HandleMessage(t string, h func([]byte, peer.ID)) error {
	v.s.mx.Lock()
	defer v.s.mx.Unlock()

	// v.pid is already a PeerID alias of peer.ID, so no conversion required.
	v.s.msgHandlers[t] = func(b []byte, _ PeerID) { h(b, v.pid) }
	return nil
}

func (v *virtualNet) UnregisterMessageHandler(t string) {
	v.s.mx.Lock()
	defer v.s.mx.Unlock()
	delete(v.s.msgHandlers, t)
}

// PubSub
func (v *virtualNet) Publish(_ context.Context, topic string, data []byte) error {
	v.s.mx.RLock()
	sinks := v.s.subs[topic]
	v.s.mx.RUnlock()
	for _, cb := range sinks {
		go cb(data)
	}
	return nil
}

func (v *virtualNet) Subscribe(
	_ context.Context,
	topic string,
	cb func([]byte),
	_ Validator,
) (uint64, error) {
	v.s.mx.Lock()
	defer v.s.mx.Unlock()

	v.s.next++
	if v.s.subs[topic] == nil {
		v.s.subs[topic] = map[uint64]func([]byte){}
	}
	v.s.subs[topic][v.s.next] = cb
	return v.s.next, nil
}

func (v *virtualNet) Unsubscribe(topic string, id uint64) error {
	v.s.mx.Lock()
	defer v.s.mx.Unlock()
	if m := v.s.subs[topic]; m != nil {
		delete(m, id)
		if len(m) == 0 {
			delete(v.s.subs, topic)
		}
	}
	return nil
}

// Rendez-vous (Advertise / Query)
func (v *virtualNet) Advertise(_ context.Context, k string, d []byte) error {
	v.s.mx.Lock()
	defer v.s.mx.Unlock()
	v.s.advert[k] = d
	return nil
}

func (v *virtualNet) Unadvertise(_ context.Context, k string) error {
	v.s.mx.Lock()
	defer v.s.mx.Unlock()
	delete(v.s.advert, k)
	return nil
}

func (v *virtualNet) Query(_ context.Context, k string) ([]*common.Advertisement, error) {
	v.s.mx.RLock()
	defer v.s.mx.RUnlock()
	d, ok := v.s.advert[k]
	if !ok {
		return nil, nil
	}
	return []*common.Advertisement{{
		PeerId:    string(v.pid),
		Timestamp: time.Now().UnixNano(),
		Data:      d,
	}}, nil
}

// Misc stubs (same behaviour as MemoryNet)
func (*virtualNet) Init(*config.Config) error          { return nil }
func (*virtualNet) Start() error                       { return nil }
func (*virtualNet) Stop() error                        { return nil }
func (v *virtualNet) GetHostID() PeerID                { return v.pid }
func (*virtualNet) GetPeerPubKey(PeerID) crypto.PubKey { return nil }
func (v *virtualNet) Stat() types.NetworkStats {
	return types.NetworkStats{ID: string(v.pid), ListenAddr: "virtual://" + string(v.pid)}
}
func (*virtualNet) ResolveAddress(context.Context, string) ([]string, error) { return nil, nil }
func (v *virtualNet) Ping(context.Context, string, time.Duration) (types.PingResult, error) {
	return types.PingResult{Success: true}, nil
}
func (*virtualNet) SetupBroadcastTopic(string, func(*Topic) error) error { return nil }
func (*virtualNet) SetBroadcastAppScore(func(PeerID) float64)            {}
func (v *virtualNet) GetBroadcastScore() map[PeerID]*PeerScoreSnapshot   { return v.s.score }
func (*virtualNet) Notify(context.Context,
	func(PeerID, []ProtocolID, int),
	func(PeerID),
	func(PeerID),
	func(PeerID, []ProtocolID),
	func(PeerID, []ProtocolID),
) error {
	return nil
}
func (*virtualNet) PeerConnected(PeerID) bool     { return true }
func (*virtualNet) GetPeerIP(PeerID) string       { return "" }
func (*virtualNet) HostPublicIP() (net.IP, error) { return nil, nil }
func (*virtualNet) CreateSubnet(context.Context, string, map[string]string) error {
	return nil
}
func (*virtualNet) DestroySubnet(string) error                    { return nil }
func (*virtualNet) AddSubnetPeer(string, string, string) error    { return nil }
func (*virtualNet) RemoveSubnetPeer(string, string, string) error { return nil }
func (*virtualNet) AcceptSubnetPeer(string, string, string) error { return nil }
func (*virtualNet) MapPort(string, string, string, string, string, string) error {
	return nil
}

func (*virtualNet) UnmapPort(string, string, string, string, string, string) error {
	return nil
}
func (*virtualNet) AddSubnetDNSRecords(string, map[string]string) error { return nil }
func (*virtualNet) RemoveSubnetDNSRecord(string, string) error          { return nil }
