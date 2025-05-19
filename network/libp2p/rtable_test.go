// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package libp2p_test

import (
	"testing"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/network/libp2p"
)

const (
	testIP1       = "192.168.1.1"
	testIP2       = "192.168.1.2"
	testPrivateIP = "10.0.0.1"
)

func generatePeerID(t *testing.T) peer.ID {
	t.Helper()
	priv, _, err := crypto.GenerateEd25519Key(nil)
	assert.NoError(t, err)

	id, err := peer.IDFromPrivateKey(priv)
	assert.NoError(t, err)

	return id
}

func TestRouteTable_AddGet(t *testing.T) {
	t.Parallel()
	rt := libp2p.NewRoutingTable()
	aliceID := generatePeerID(t)
	bobID := generatePeerID(t)

	// Test adding a new peer with a single address
	rt.Add(aliceID, testIP1)
	addrs, ok := rt.Get(aliceID)
	assert.True(t, ok)
	assert.Equal(t, 1, len(addrs))
	assert.Equal(t, testIP1, addrs[0])

	// Test adding another address to existing peer
	rt.Add(aliceID, testIP2)
	addrs, ok = rt.Get(aliceID)
	assert.True(t, ok)
	assert.Equal(t, 2, len(addrs))
	assert.Contains(t, addrs, testIP1)
	assert.Contains(t, addrs, testIP2)

	// Test adding a new peer
	rt.Add(bobID, testPrivateIP)
	addrs, ok = rt.Get(bobID)
	assert.True(t, ok)
	assert.Equal(t, 1, len(addrs))
	assert.Equal(t, testPrivateIP, addrs[0])

	// Verify reverse index
	peerID, ok := rt.GetByIP(testIP1)
	assert.True(t, ok)
	assert.Equal(t, aliceID, peerID)

	// Test get with non-existent peer
	unknown := generatePeerID(t)
	_, ok = rt.Get(unknown)
	assert.False(t, ok)
}

func TestRouteTable_Remove(t *testing.T) {
	t.Parallel()
	rt := libp2p.NewRoutingTable()
	aliceID := generatePeerID(t)

	rt.Add(aliceID, testIP1)
	rt.Add(aliceID, testIP2)

	// Test removing one address
	rt.Remove(aliceID, testIP1)
	addrs, ok := rt.Get(aliceID)
	assert.True(t, ok)
	assert.Equal(t, 1, len(addrs))
	assert.Equal(t, testIP2, addrs[0])

	// Verify the IP is no longer in the reverse index
	_, ok = rt.GetByIP(testIP1)
	assert.False(t, ok)

	// Test removing the last address
	rt.Remove(aliceID, testIP2)
	_, ok = rt.Get(aliceID)
	assert.False(t, ok)

	// Test removing a non-existent peer
	bobID := generatePeerID(t)
	rt.Remove(bobID, testPrivateIP) // Should not cause any errors
}

func TestRouteTable_RemoveByIP(t *testing.T) {
	t.Parallel()
	rt := libp2p.NewRoutingTable()
	aliceID := generatePeerID(t)

	rt.Add(aliceID, testIP1)

	// Test removing by IP
	rt.RemoveByIP(testIP1)
	_, ok := rt.Get(aliceID)
	assert.False(t, ok)
	_, ok = rt.GetByIP(testIP1)
	assert.False(t, ok)

	// Test removing a non-existent IP
	rt.RemoveByIP(testPrivateIP) // Should not cause any errors
}

func TestRouteTable_All(t *testing.T) {
	t.Parallel()
	rt := libp2p.NewRoutingTable()

	// Test with empty table
	all := rt.All()
	assert.Equal(t, 0, len(all))

	// Add peers and test
	aliceID := generatePeerID(t)
	bobID := generatePeerID(t)
	rt.Add(aliceID, testIP1)
	rt.Add(bobID, testPrivateIP)

	all = rt.All()
	assert.Equal(t, 2, len(all))
	assert.Contains(t, all, aliceID)
	assert.Contains(t, all, bobID)
	assert.Equal(t, []string{testIP1}, all[aliceID])
	assert.Equal(t, []string{testPrivateIP}, all[bobID])
}

func TestRouteTable_Clear(t *testing.T) {
	t.Parallel()
	rt := libp2p.NewRoutingTable()
	aliceID := generatePeerID(t)
	bobID := generatePeerID(t)

	rt.Add(aliceID, testIP1)
	rt.Add(bobID, testPrivateIP)

	rt.Clear()

	// Verify table is empty
	_, ok := rt.Get(aliceID)
	assert.False(t, ok)
	_, ok = rt.Get(bobID)
	assert.False(t, ok)
	_, ok = rt.GetByIP(testIP1)
	assert.False(t, ok)
	_, ok = rt.GetByIP(testPrivateIP)
	assert.False(t, ok)

	all := rt.All()
	assert.Equal(t, 0, len(all))
}
