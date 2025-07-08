// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build integration || !unit

package libp2p

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubnetCreate(t *testing.T) {
	peer1 := createPeer(t, 0)
	require.NotNil(t, peer1)

	err := peer1.CreateSubnet(context.Background(), "subnet1", map[string]string{})
	require.NoError(t, err)

	assert.Equal(t, 1, len(peer1.subnets))
	assert.Equal(t, "subnet1", peer1.subnets["subnet1"].info.id)
	assert.Equal(t, 0, len(peer1.subnets["subnet1"].ifaces))
	assert.Equal(t, 0, len(peer1.subnets["subnet1"].info.rtable.All()))
}

func TestSubnetAddRemovePeer(t *testing.T) {
	peer1 := createPeer(t, 0)
	require.NotNil(t, peer1)

	err := peer1.CreateSubnet(context.Background(), "subnet1", map[string]string{})
	require.NoError(t, err)

	// requires root privileges - skipping if not root
	if os.Getuid() != 0 {
		t.Skip("requires root privileges")
	}
	err = peer1.AddSubnetPeer("subnet1", peer1.Host.ID().String(), "10.0.0.2")
	require.NoError(t, err)

	assert.Equal(t, 1, len(peer1.subnets))
	assert.Equal(t, 1, len(peer1.subnets["subnet1"].ifaces))
	assert.Equal(t, 1, len(peer1.subnets["subnet1"].info.rtable.All()))

	ips, ok := peer1.subnets["subnet1"].info.rtable.Get(peer1.Host.ID())
	require.True(t, ok)

	assert.Equal(t, "10.0.0.2", ips[0])

	peerID, ok := peer1.subnets["subnet1"].info.rtable.GetByIP("10.0.0.2")
	require.True(t, ok)

	assert.Equal(t, peer1.Host.ID(), peerID)

	err = peer1.RemoveSubnetPeers(
		"subnet1",
		map[string]string{
			"10.0.0.2": peer1.Host.ID().String(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, 1, len(peer1.subnets))
	assert.Equal(t, 0, len(peer1.subnets["subnet1"].ifaces))

	assert.Equal(t, 0, len(peer1.subnets["subnet1"].info.rtable.All()))
}

func TestSubnetMapUnmapPorts(t *testing.T) {
	_, err := exec.LookPath("iptables")
	if err != nil {
		t.Skip("iptables not found in path")
	}

	// requires root privileges - skipping if not root
	if os.Getuid() != 0 {
		t.Skip("requires root privileges")
	}

	peer1 := createPeer(t, 0)
	require.NotNil(t, peer1)

	err = peer1.CreateSubnet(context.Background(), "subnet1", map[string]string{})
	require.NoError(t, err)

	err = peer1.MapPort("subnet1", "tcp", "0.0.0.0", "8080", "10.0.0.1", "8888")
	require.NoError(t, err)

	cmd := exec.Command("sh", "-c", "iptables -t nat -L PREROUTING -v -n")
	op, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(op), "tcp dpt:8080 to:10.0.0.1:8888"))

	cmd = exec.Command("sh", "-c", "iptables -L FORWARD -v -n")
	op, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(op), "tcp dpt:8888"))

	// Command to list POSTROUTING rules in the nat table and grep for the port
	cmd = exec.Command("sh", "-c", "iptables -t nat -L POSTROUTING -v -n")
	op, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(op), "0.0.0.0/0"))

	assert.Equal(t, 1, len(peer1.subnets["subnet1"].portMapping))
	assert.Equal(t, "8888", peer1.subnets["subnet1"].portMapping["8080"].destPort)
	assert.Equal(t, "10.0.0.1", peer1.subnets["subnet1"].portMapping["8080"].destIP)
	assert.Equal(t, "0.0.0.0", peer1.subnets["subnet1"].portMapping["8080"].srcIP)

	err = peer1.UnmapPort("subnet1", "tcp", "0.0.0.0", "8080", "10.0.0.1", "9999")
	require.Error(t, err, "is not mapped to")

	err = peer1.UnmapPort("subnet1", "tcp", "0.0.0.0", "8080", "10.0.0.1", "8888")
	require.NoError(t, err)

	// port := "8080"
	cmd = exec.Command("sh", "-c", "iptables -t nat -L PREROUTING -v -n")
	op, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(op), "tcp dpt:8080 to:"))

	cmd = exec.Command("sh", "-c", "iptables -L FORWARD -v -n")
	op, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(op), "tcp dpt:8888"))

	// Command to list POSTROUTING rules in the nat table and grep for the port
	cmd = exec.Command("sh", "-c", "iptables -t nat -L POSTROUTING -v -n")
	op, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(op), "0.0.0.0/0"))

	assert.Equal(t, 0, len(peer1.subnets["subnet1"].portMapping))
}

func TestSubnetAddRemoveDNSRecord(t *testing.T) {
	// requires iptables and dig
	_, err := exec.LookPath("iptables")
	if err != nil {
		t.Skip("iptables not found in path")
	}

	// requires root privileges - skipping if not root
	if os.Getuid() != 0 {
		t.Skip("requires root privileges")
	}

	peer1 := createPeer(t, 0)
	require.NotNil(t, peer1)

	err = peer1.CreateSubnet(context.Background(), "subnet1", map[string]string{})
	require.NoError(t, err)

	err = peer1.AddSubnetPeer("subnet1", peer1.Host.ID().String(), "10.0.0.2")
	require.NoError(t, err)

	err = peer1.AddSubnetDNSRecords("subnet1", map[string]string{"example.com.": "10.20.30.40"})
	require.NoError(t, err)

	<-time.After(3 * time.Second)

	// requires dig
	_, err = exec.LookPath("dig")
	if err != nil {
		t.Skip("dig not found in path")
	}

	cmd := exec.Command("sh", "-c", "dig +short @10.0.0.1 example.com")
	op, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Equal(t, "10.20.30.40", strings.TrimSpace(string(op)))

	err = peer1.RemoveSubnetDNSRecord("subnet1", "example.com.")
	require.NoError(t, err)

	cmd = exec.Command("sh", "-c", "dig +short @10.0.0.1")
	op, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Equal(t, "", strings.TrimSpace(string(op)))
}

func TestSubnetDestroy(t *testing.T) {
	// requires iptables and dig
	_, err := exec.LookPath("iptables")
	if err != nil {
		t.Skip("iptables not found in path")
	}

	// requires root privileges - skipping if not root
	if os.Getuid() != 0 {
		t.Skip("requires root privileges")
	}

	peer1 := createPeer(t, 0)
	require.NotNil(t, peer1)

	err = peer1.CreateSubnet(context.Background(), "subnet1", map[string]string{})
	require.NoError(t, err)

	err = peer1.AddSubnetPeer("subnet1", peer1.Host.ID().String(), "10.0.0.2")
	require.NoError(t, err)

	err = peer1.AddSubnetDNSRecords("subnet1", map[string]string{"example.com.": "10.20.30.40"})
	require.NoError(t, err)

	err = peer1.MapPort("subnet1", "tcp", "0.0.0.0", "8080", "10.0.0.1", "8888")
	require.NoError(t, err)

	err = peer1.DestroySubnet("subnet1")
	require.NoError(t, err)

	assert.Equal(t, 0, len(peer1.subnets))

	// requires dig
	_, err = exec.LookPath("dig")
	if err != nil {
		t.Skip("dig not found in path")
	}

	cmd := exec.Command("sh", "-c", "dig +short @10.0.0.1")
	op, err := cmd.CombinedOutput()
	require.NoError(t, err)
	assert.Equal(t, "", strings.TrimSpace(string(op)))

	cmd = exec.Command("sh", "-c", "iptables -t nat -L PREROUTING -v -n")
	op, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(op), "tcp dpt:8080 to:"))

	cmd = exec.Command("sh", "-c", "iptables -L FORWARD -v -n")
	op, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(op), "tcp dpt:8888"))

	// Command to list POSTROUTING rules in the nat table and grep for the port
	cmd = exec.Command("sh", "-c", "iptables -t nat -L POSTROUTING -v -n")
	op, err = cmd.CombinedOutput()
	require.NoError(t, err)
	assert.False(t, strings.Contains(string(op), "0.0.0.0/0"))
}
