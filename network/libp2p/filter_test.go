// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
