package libp2p

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"
)

func TestNoAddrIDFilter(t *testing.T) {
	testID := peer.ID("testID")
	filter := NoAddrIDFilter{ID: testID}

	tests := []struct {
		name     string
		p        peer.AddrInfo
		expected bool
	}{
		{
			name:     "No addresses and matching ID",
			p:        peer.AddrInfo{ID: testID},
			expected: false,
		},
		{
			name:     "No addresses and non-matching ID",
			p:        peer.AddrInfo{ID: peer.ID("otherID")},
			expected: false,
		},
		{
			name: "Has addresses and non-matching ID",
			p: peer.AddrInfo{
				ID: peer.ID("otherID"),
				Addrs: []multiaddr.Multiaddr{
					multiaddr.StringCast("/ip4/127.0.0.1/tcp/8080"),
				},
			},
			expected: true,
		},
		{
			name: "Has addresses and matching ID",
			p: peer.AddrInfo{
				ID: testID,
				Addrs: []multiaddr.Multiaddr{
					multiaddr.StringCast("/ip4/127.0.0.1/tcp/8080"),
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, filter.satisfies(tt.p))
		})
	}
}

func TestPeerPassFilter(t *testing.T) {
	testID := peer.ID("testID")
	filter := NoAddrIDFilter{ID: testID}

	peers := []peer.AddrInfo{
		{
			ID:    testID,
			Addrs: nil,
		},
		{
			ID: peer.ID("otherID"),
			Addrs: []multiaddr.Multiaddr{
				multiaddr.StringCast("/ip4/127.0.0.1/tcp/8080"),
			},
		},
	}

	expected := []peer.AddrInfo{
		{
			ID: peer.ID("otherID"),
			Addrs: []multiaddr.Multiaddr{
				multiaddr.StringCast("/ip4/127.0.0.1/tcp/8080"),
			},
		},
	}

	filtered := PeerPassFilter(peers, filter)
	assert.Equal(t, expected, filtered)
}

func TestMakeAddrsFactory(t *testing.T) {
	announce := []string{"/ip4/127.0.0.1/tcp/8080"}
	appendAnnounce := []string{"/ip4/192.168.0.1/tcp/8080"}
	noAnnounce := []string{"/ip4/10.0.0.1/tcp/8080"}

	factory := makeAddrsFactory(announce, appendAnnounce, noAnnounce)
	assert.NotNil(t, factory)

	allAddrs := []multiaddr.Multiaddr{
		multiaddr.StringCast("/ip4/127.0.0.1/tcp/8080"),
		multiaddr.StringCast("/ip4/192.168.0.1/tcp/8080"),
		multiaddr.StringCast("/ip4/10.0.0.1/tcp/8080"),
	}

	expected := []multiaddr.Multiaddr{
		multiaddr.StringCast("/ip4/127.0.0.1/tcp/8080"),
		multiaddr.StringCast("/ip4/192.168.0.1/tcp/8080"),
	}

	result := factory(allAddrs)
	assert.Equal(t, expected, result)
}
