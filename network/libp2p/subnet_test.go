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
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/google/gopacket"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/peerstore"

	multiaddr "github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/assert"

	rand "math/rand/v2"

	layers "github.com/google/gopacket/layers"
	"github.com/miekg/dns"
	"github.com/stretchr/testify/require"
	sys "gitlab.com/nunet/device-management-service/lib/sys"
	"gitlab.com/nunet/device-management-service/observability"
)

func TestSubnetCreate(t *testing.T) {
	// Use a specific port for this test
	peer1 := createPeer(t, 0, 3000, []multiaddr.Multiaddr{})
	require.NotNil(t, peer1)

	// Wait a bit to ensure resources are available
	time.Sleep(500 * time.Millisecond)

	err := peer1.CreateSubnet(context.Background(), "subnet1", "10.20.30.0/24", map[string]string{})
	require.NoError(t, err)

	assert.Equal(t, 1, len(peer1.subnets))
	assert.Equal(t, "subnet1", peer1.subnets["subnet1"].info.id)
	assert.Equal(t, 0, len(peer1.subnets["subnet1"].ifaces))
	assert.Equal(t, 0, len(peer1.subnets["subnet1"].info.rtable.All()))
}

func TestSubnetAddRemovePeer(t *testing.T) {
	// Use a specific port for this test
	peer1 := createPeer(t, 0, 3000, []multiaddr.Multiaddr{})
	require.NotNil(t, peer1)

	// Wait a bit to ensure resources are available
	time.Sleep(500 * time.Millisecond)

	err := peer1.CreateSubnet(context.Background(), "subnet1", "10.0.0.0/24", map[string]string{})
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

	// Use a specific port for this test
	peer1 := createPeer(t, 0, 3000, []multiaddr.Multiaddr{})
	require.NotNil(t, peer1)

	// Wait a bit to ensure resources are available
	time.Sleep(500 * time.Millisecond)

	err = peer1.CreateSubnet(context.Background(), "subnet1", "10.0.0.0/24", map[string]string{})
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

	// Use a specific port for this test
	peer1 := createPeer(t, 0, 3000, []multiaddr.Multiaddr{})
	require.NotNil(t, peer1)

	// Wait a bit to ensure resources are available
	time.Sleep(500 * time.Millisecond)

	err = peer1.CreateSubnet(context.Background(), "subnet1", "10.0.0.0/24", map[string]string{})
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

	var cmd *exec.Cmd
	var op []byte

	cmd = exec.Command("sh", "-c", "dig +short @10.0.0.1 example.com")
	op, err = cmd.CombinedOutput()
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

	// Use a specific port for this test
	peer1 := createPeer(t, 0, 3000, []multiaddr.Multiaddr{})
	require.NotNil(t, peer1)

	// Wait a bit to ensure resources are available
	time.Sleep(500 * time.Millisecond)

	err = peer1.CreateSubnet(context.Background(), "subnet1", "10.0.0.0/24", map[string]string{})
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

	var cmd *exec.Cmd
	var op []byte

	cmd = exec.Command("sh", "-c", "dig +short @10.0.0.1")
	op, err = cmd.CombinedOutput()
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

func craftICMPEchoPacket(srcIP, dstIP string, icmpType uint8, payload []byte) ([]byte, error) {
	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		SrcIP:    net.ParseIP(srcIP),
		DstIP:    net.ParseIP(dstIP),
		Protocol: layers.IPProtocolICMPv4,
	}
	icmp := &layers.ICMPv4{
		TypeCode: layers.CreateICMPv4TypeCode(icmpType, 0),
		Id:       uint16(rand.IntN(65535)),
		Seq:      1,
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	if err := gopacket.SerializeLayers(buf, opts, ip, icmp, gopacket.Payload(payload)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func TestSubnetIOLoop(t *testing.T) {
	require.NoError(t, observability.SetLogLevel("DEBUG"))

	peer1 := createPeer(t, 0, 3011, []multiaddr.Multiaddr{}, mockNetInterfaceFactory)
	require.NoError(t, peer1.Start())

	peer2 := createPeer(t, 0, 3013, []multiaddr.Multiaddr{
		multiaddr.StringCast(peer1.Host.Addrs()[2].String() + "/p2p/" + peer1.Host.ID().String()),
	}, mockNetInterfaceFactory)
	require.NoError(t, peer2.Start())

	t.Log("Adding peer addresses to peerstore")
	peer2.
		Host.
		Peerstore().
		AddAddrs(
			peer1.Host.ID(),
			[]multiaddr.Multiaddr{multiaddr.StringCast(peer1.Host.Addrs()[2].String())},
			peerstore.PermanentAddrTTL,
		)
	peer1.
		Host.
		Peerstore().
		AddAddrs(
			peer2.Host.ID(),
			[]multiaddr.Multiaddr{multiaddr.StringCast(peer2.Host.Addrs()[2].String())},
			peerstore.PermanentAddrTTL,
		)
	peer2AddrInfo := peer.AddrInfo{
		ID:    peer2.Host.ID(),
		Addrs: peer2.Host.Addrs(),
	}
	require.NoError(t, peer1.Host.Connect(context.Background(), peer2AddrInfo))

	t.Log("Connecting peers")
	t.Log("Starting peers")
	subnetID := "testsubnet"
	ip1 := "10.0.0.2"
	ip2 := "10.0.0.3"
	ip3 := "10.0.0.4"
	peer1ID := peer1.Host.ID().String()
	peer2ID := peer2.Host.ID().String()

	routingTable := map[string]string{
		ip1: peer1ID,
		ip2: peer2ID,
		ip3: peer1ID,
	}

	// No need to set NetIfaceFactory after construction

	t.Log("Creating subnets")
	require.NoError(t, peer1.CreateSubnet(context.Background(), subnetID, "10.0.0.0/24", routingTable))
	require.NoError(t, peer2.CreateSubnet(context.Background(), subnetID, "10.0.0.0/24", routingTable))

	t.Log("Adding subnet peers")
	// Only add the local peer as a subnet peer on each
	require.NoError(t, peer1.AddSubnetPeer(subnetID, peer1ID, ip1))
	require.NoError(t, peer1.AddSubnetPeer(subnetID, peer1ID, ip3))
	require.NoError(t, peer2.AddSubnetPeer(subnetID, peer2ID, ip2))

	// Get the mocks from the subnet's ifaces
	mock1 := peer1.subnets[subnetID].ifaces[ip1].tun.(*MockNetInterface)
	mock2 := peer2.subnets[subnetID].ifaces[ip2].tun.(*MockNetInterface)
	mock3 := peer1.subnets[subnetID].ifaces[ip3].tun.(*MockNetInterface)

	t.Run("test subnet io loop - happy path", func(t *testing.T) {
		// ICMP Echo (ping/pong) packets
		payload := []byte("test payload")

		pingPacket, err := craftICMPEchoPacket(ip1, ip2, layers.ICMPv4TypeEchoRequest, payload)
		require.NoError(t, err)
		pongPacket, err := craftICMPEchoPacket(ip2, ip1, layers.ICMPv4TypeEchoReply, payload)
		require.NoError(t, err)

		rounds := 5
		done := make(chan struct{})

		t.Log("Starting ping-pong sequence")

		errCh := make(chan error)
		go func() {
			for i := 0; i < rounds; i++ {
				// Peer1 sends ICMP Echo Request
				_, err := mock1.Write(pingPacket)
				require.NoError(t, err)
				// Peer2 receives ping
				var got []byte
				select {
				case got = <-mock2.writeCh:
					t.Log("Received ping from peer1")
					packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
					icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
					require.NotNil(t, icmpLayer)
					icmp, _ := icmpLayer.(*layers.ICMPv4)
					assert.Equal(t, uint8(layers.ICMPv4TypeEchoRequest), icmp.TypeCode.Type())
				case <-time.After(60 * time.Second):
					errCh <- fmt.Errorf("timeout waiting for ping at peer2")
				}
				// Peer2 responds with ICMP Echo Reply
				_, err = mock2.Write(pongPacket)
				require.NoError(t, err)
				t.Log("Sent pong to peer1")
				// Peer1 receives pong
				select {
				case got = <-mock1.writeCh:
					t.Log("Received pong from peer2")
					packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
					icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
					require.NotNil(t, icmpLayer)
					icmp, _ := icmpLayer.(*layers.ICMPv4)
					assert.Equal(t, uint8(layers.ICMPv4TypeEchoReply), icmp.TypeCode.Type())
				case <-time.After(60 * time.Second):
					errCh <- fmt.Errorf("timeout waiting for pong at peer1")
				}
			}
			close(done)
		}()

		select {
		case <-done:
			assert.NotNil(t, peer1.subnets[subnetID].proxiedConns.conns[ip2])
			assert.NotNil(t, peer2.subnets[subnetID].proxiedConns.conns[ip1])
		case err := <-errCh:
			t.Fatal(err)
		case <-time.After(time.Duration(rounds) * 200 * time.Second):
			t.Fatal("ping-pong test did not complete in time")
		}
	})
	t.Run("test subnet io loop - inject non-member ip in packet", func(t *testing.T) {
		// ICMP Echo (ping/pong) packets
		payload := []byte("test payload")

		wrongSrcPacket, err := craftICMPEchoPacket("90.90.90.90", ip2, layers.ICMPv4TypeEchoRequest, payload)
		require.NoError(t, err)
		wrongDestPacket, err := craftICMPEchoPacket(ip2, "90.90.90.90", layers.ICMPv4TypeEchoReply, payload)
		require.NoError(t, err)

		t.Log("Sending faulty packet")

		// Peer1 sends ICMP Echo Request
		_, err = mock1.Write(wrongSrcPacket)
		require.NoError(t, err)
		// Peer2 receives ping
		select {
		case <-mock2.writeCh:
			t.Fail()
		case <-time.After(5 * time.Second):
			t.Log("did not receive packet, continuing")
		}

		// Peer1 sends ICMP Echo Request
		_, err = mock1.Write(wrongDestPacket)
		require.NoError(t, err)
		// Peer2 receives ping
		select {
		case <-mock2.writeCh:
			t.Fail()
		case <-time.After(5 * time.Second):
			t.Log("did not receive packet, continuing")
		}
	})

	t.Run("test subnet io loop - shortcuts network hops when member is local", func(t *testing.T) {
		// ICMP Echo (ping/pong) packets
		payload := []byte("test payload")

		pingPacketToLocalMember, err := craftICMPEchoPacket(ip1, ip3, layers.ICMPv4TypeEchoRequest, payload)
		require.NoError(t, err)
		pongPacketToLocalMember, err := craftICMPEchoPacket(ip3, ip1, layers.ICMPv4TypeEchoReply, payload)
		require.NoError(t, err)

		t.Log("Sending packet to local member")

		// Peer1 sends ICMP Echo Request
		_, err = mock1.Write(pingPacketToLocalMember)
		require.NoError(t, err)
		// Peer2 receives ping
		select {
		case <-mock3.writeCh:
			t.Log("received ping packet from local member")
		case <-time.After(10 * time.Second):
			t.Fatal("did not receive ping packet from local member")
		}

		// Peer1 sends ICMP Echo Request
		_, err = mock3.Write(pongPacketToLocalMember)
		require.NoError(t, err)
		// Peer2 receives ping
		select {
		case <-mock1.writeCh:
			t.Log("received pong packet from local member")
		case <-time.After(10 * time.Second):
			t.Fatal("did not receive pong packet from local member")
		}
	})

	t.Run("test subnet io loop - maintains one connection per peer", func(t *testing.T) {
		t.Skip()
		if peer1.subnets[subnetID].proxiedConns.conns[ip2] != nil {
			peer1.subnets[subnetID].proxiedConns.conns[ip2].Close()
		}
		time.Sleep(1 * time.Second) // allow peer2 to detect closed conn and clean in

		// assert that existing connection was closed
		assert.Nil(t, peer1.subnets[subnetID].proxiedConns.conns[ip2])
		assert.Nil(t, peer2.subnets[subnetID].proxiedConns.conns[ip1])

		// ICMP Echo (ping/pong) packets
		payload := []byte("test payload")

		pingPacket, err := craftICMPEchoPacket(ip2, ip1, layers.ICMPv4TypeEchoRequest, payload)
		require.NoError(t, err)
		pongPacket, err := craftICMPEchoPacket(ip1, ip2, layers.ICMPv4TypeEchoReply, payload)
		require.NoError(t, err)

		rounds := 5
		done := make(chan struct{})

		t.Log("Starting ping-pong sequence")

		errCh := make(chan error)
		go func() {
			for i := 0; i < rounds; i++ {
				// Peer1 sends ICMP Echo Request
				_, err := mock2.Write(pingPacket)
				require.NoError(t, err)
				// Peer2 receives ping
				var got []byte
				select {
				case got = <-mock1.writeCh:
					t.Log("Received ping from peer1")
					packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
					icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
					require.NotNil(t, icmpLayer)
					icmp, _ := icmpLayer.(*layers.ICMPv4)
					assert.Equal(t, uint8(layers.ICMPv4TypeEchoRequest), icmp.TypeCode.Type())
				case <-time.After(60 * time.Second):
					errCh <- fmt.Errorf("timeout waiting for ping at peer2")
				}
				// Peer2 responds with ICMP Echo Reply
				_, err = mock1.Write(pongPacket)
				require.NoError(t, err)
				t.Log("Sent pong to peer1")
				// Peer1 receives pong
				select {
				case got = <-mock2.writeCh:
					t.Log("Received pong from peer2")
					packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
					icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
					require.NotNil(t, icmpLayer)
					icmp, _ := icmpLayer.(*layers.ICMPv4)
					assert.Equal(t, uint8(layers.ICMPv4TypeEchoReply), icmp.TypeCode.Type())
				case <-time.After(60 * time.Second):
					errCh <- fmt.Errorf("timeout waiting for pong at peer1")
				}
			}
			close(done)
		}()

		select {
		case <-done:
			assert.NotNil(t, peer1.subnets[subnetID].proxiedConns.conns[ip2])
			assert.NotNil(t, peer2.subnets[subnetID].proxiedConns.conns[ip1])
		case err := <-errCh:
			t.Fatal(err)
		case <-time.After(time.Duration(rounds) * 200 * time.Second):
			t.Fatal("ping-pong test did not complete in time")
		}
	})

	t.Run("test subnet io loop - large TCP packet echo", func(t *testing.T) {
		payload2 := make([]byte, 200)
		payload := make([]byte, 1400)
		for i := range payload {
			payload[i] = byte(i % 256)
		}
		for i := range payload2 {
			payload2[i] = byte(i % 256)
		}

		tcpPacket, err := craftTCPPacket(ip1, ip2, 12345, 54321, payload)
		require.NoError(t, err)

		t.Log("sending large TCP packet", len(tcpPacket))

		done := make(chan struct{})
		errCh := make(chan error)
		go func() {
			// Peer1 sends large TCP packet
			_, err := mock1.Write(tcpPacket)
			require.NoError(t, err)
			// Peer2 receives
			var got []byte
			select {
			case got = <-mock1.writeCh:
				packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
				if packet.Layer(layers.LayerTypeTCP) == nil {
					t.Log("received non-TCP packet back on peer 1")
					ipLayer := packet.Layer(layers.LayerTypeIPv4)
					require.NotNil(t, ipLayer)
					icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
					require.NotNil(t, icmpLayer)
					icmp, _ := icmpLayer.(*layers.ICMPv4)
					// Assert ICMP Type 3 (Destination Unreachable), Code 4 (Fragmentation Needed)
					assert.Equal(t, uint8(3), icmp.TypeCode.Type(), "ICMP type should be 3 (Destination Unreachable)")
					assert.Equal(t, uint8(4), icmp.TypeCode.Code(), "ICMP code should be 4 (Fragmentation Needed)")
					t.Log("received ICMP Fragmentation Needed packet")

					// Fragment the packet
					for i := range payload2 {
						payload2[i] = byte(i % 256)
					}

					// Craft TCP packet from ip1 to ip2
					tcpPacket, err := craftTCPPacket(ip1, ip2, 12345, 54321, payload2)
					require.NoError(t, err)
					_, err = mock1.Write(tcpPacket)
					require.NoError(t, err)
				}
			case <-time.After(5 * time.Second):
				errCh <- fmt.Errorf("timeout waiting for large TCP packet at peer2")
			}

			select {
			case got = <-mock2.writeCh:
				t.Log("received TCP packet on peer 2")
				packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
				ipLayer := packet.Layer(layers.LayerTypeIPv4)
				require.NotNil(t, ipLayer)
				tcpLayer := packet.Layer(layers.LayerTypeTCP)
				require.NotNil(t, tcpLayer)
				payloadLayer := packet.ApplicationLayer()
				require.NotNil(t, payloadLayer)
				assert.Equal(t, payload2, payloadLayer.Payload())

				// Echo backgg
				tcpPacket, err = craftTCPPacket(ip2, ip1, 54321, 12345, payload2)
				require.NoError(t, err)
				t.Log("sending TCP packet back to peer 1")
				_, err = mock2.Write(tcpPacket)
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				errCh <- fmt.Errorf("timeout waiting for TCP echo at peer2")
			}

			// Peer1 receives echo
			select {
			case got = <-mock1.writeCh:
				packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
				if packet.Layer(layers.LayerTypeTCP) == nil {
					t.Log("received ICMP Fragmentation Needed on peer 1")
					ipLayer := packet.Layer(layers.LayerTypeIPv4)
					require.NotNil(t, ipLayer)
					icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
					require.NotNil(t, icmpLayer)
					icmp, _ := icmpLayer.(*layers.ICMPv4)
					assert.Equal(t, uint8(3), icmp.TypeCode.Type(), "ICMP type should be 3 (Destination Unreachable)")
					assert.Equal(t, uint8(4), icmp.TypeCode.Code(), "ICMP code should be 4 (Fragmentation Needed)")
				} else {
					t.Log("received TCP packet on peer 1")
					ipLayer := packet.Layer(layers.LayerTypeIPv4)
					require.NotNil(t, ipLayer)
					tcpLayer := packet.Layer(layers.LayerTypeTCP)
					require.NotNil(t, tcpLayer)
					payloadLayer := packet.ApplicationLayer()
					require.NotNil(t, payloadLayer)
					assert.Equal(t, payload2, payloadLayer.Payload())
				}
				fmt.Println("received TCP packet - 2")
			case <-time.After(35 * time.Second):
				errCh <- fmt.Errorf("timeout waiting for TCP echo at peer1")
			}
			close(done)
		}()
		select {
		case <-done:
			t.Log("large TCP packet echo test completed")
		case err := <-errCh:
			t.Fatal(err)
		case <-time.After(60 * time.Second):
			t.Fatal("large TCP packet echo test did not complete in time")
		}
	})

	t.Run("test subnet io loop - packets larger than MTU and buffer", func(t *testing.T) {
		// Create a very large payload (5000 bytes)
		smallPayload := make([]byte, 1000)
		for i := range smallPayload {
			smallPayload[i] = byte(i % 256)
		}
		payload := make([]byte, 5000)
		for i := range payload {
			payload[i] = byte(i % 256)
		}

		// Craft TCP packet with the large payload
		tcpPacket, err := craftTCPPacket(ip1, ip2, 12345, 54321, payload)
		require.NoError(t, err)

		t.Log("sending very large TCP packet", len(tcpPacket))

		done := make(chan struct{})
		errCh := make(chan error)
		go func() {
			// Peer1 sends very large TCP packet
			_, err := mock1.Write(tcpPacket)
			require.NoError(t, err)

			// We should receive an ICMP Fragmentation Needed message
			var got []byte
			select {
			case got = <-mock1.writeCh:
				packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
				ipLayer := packet.Layer(layers.LayerTypeIPv4)
				require.NotNil(t, ipLayer)
				icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
				require.NotNil(t, icmpLayer)
				icmp, _ := icmpLayer.(*layers.ICMPv4)

				// Verify we got an ICMP Fragmentation Needed message
				assert.Equal(t, uint8(3), icmp.TypeCode.Type(), "ICMP type should be 3 (Destination Unreachable)")
				assert.Equal(t, uint8(4), icmp.TypeCode.Code(), "ICMP code should be 4 (Fragmentation Needed)")
				t.Log("received ICMP Fragmentation Needed packet as expected")

				// Now send a smaller packet that should fit within MTU
				smallPacket, err := craftTCPPacket(ip1, ip2, 12345, 54321, smallPayload)
				require.NoError(t, err)
				_, err = mock1.Write(smallPacket)
				require.NoError(t, err)
			case <-time.After(5 * time.Second):
				errCh <- fmt.Errorf("timeout waiting for ICMP Fragmentation Needed message")
				return
			}

			select {
			case got = <-mock2.writeCh:
				t.Log("received small packet on peer 2")
				packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
				ipLayer := packet.Layer(layers.LayerTypeIPv4)
				require.NotNil(t, ipLayer)
				tcpLayer := packet.Layer(layers.LayerTypeTCP)
				require.NotNil(t, tcpLayer)
				payloadLayer := packet.ApplicationLayer()
				require.NotNil(t, payloadLayer)
				assert.Equal(t, smallPayload, payloadLayer.Payload())
			case <-time.After(5 * time.Second):
				errCh <- fmt.Errorf("timeout waiting for small packet response")
				return
			}
			close(done)
		}()

		select {
		case <-done:
			t.Log("large packet test completed")
			// empty the writeCh
			for len(mock1.writeCh) > 0 {
				<-mock1.writeCh
			}
			for len(mock2.writeCh) > 0 {
				<-mock2.writeCh
			}
		case err := <-errCh:
			t.Fatal(err)
		case <-time.After(60 * time.Second):
			t.Fatal("large packet test did not complete in time")
		}
	})
	t.Run("test subnet io loop - large UDP packet echo", func(t *testing.T) {
		oversizePayload := make([]byte, 1700)
		acceptablePayload := make([]byte, 1200)
		for i := range oversizePayload {
			oversizePayload[i] = byte(255 - (i % 256))
		}
		for i := range acceptablePayload {
			acceptablePayload[i] = byte(255 - (i % 256))
		}

		largeUDPPacket, err := craftUDPPacket(ip1, ip2, 23456, 65432, oversizePayload)
		require.NoError(t, err)
		done := make(chan struct{})
		errCh := make(chan error)
		go func() {
			t.Helper()
			t.Log("sending large UDP packet to peer 2")
			// Peer1 sends large UDP packet
			_, err := mock1.Write(largeUDPPacket)
			require.NoError(t, err)

			// Peer 1 receives ICMP Fragmentation Needed message
			var got []byte
			select {
			case got = <-mock1.writeCh:
				t.Log("received packet on peer 1")
				packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
				if packet.Layer(layers.LayerTypeUDP) == nil {
					t.Log("received non-UDP packet back on peer 1")
					ipLayer := packet.Layer(layers.LayerTypeIPv4)
					require.NotNil(t, ipLayer)
					icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
					require.NotNil(t, icmpLayer)
					icmp, _ := icmpLayer.(*layers.ICMPv4)
					assert.Equal(t, uint8(3), icmp.TypeCode.Type(), "ICMP type should be 3 (Destination Unreachable)")
					assert.Equal(t, uint8(4), icmp.TypeCode.Code(), "ICMP code should be 4 (Fragmentation Needed)")
					t.Log("received ICMP Fragmentation Needed packet")

					// Fragment the packet
					for i := range acceptablePayload {
						acceptablePayload[i] = byte(255 - (i % 256))
					}
					smallUDPPacket, err := craftUDPPacket(ip1, ip2, 23456, 65432, acceptablePayload)
					require.NoError(t, err)
					t.Log("sending small UDP packet to peer 2")
					_, err = mock1.Write(smallUDPPacket)
					require.NoError(t, err)
				} else {
					errCh <- fmt.Errorf("received UDP packet on peer 1 instead of ICMP Fragmentation Needed")
				}
			case <-time.After(30 * time.Second):
				errCh <- fmt.Errorf("timeout waiting for ICMP Fragmentation Needed packet at peer1")
			}

			select {
			case got = <-mock2.writeCh:
				t.Log("received packet on peer 2")
				packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
				ipLayer := packet.Layer(layers.LayerTypeIPv4)
				require.NotNil(t, ipLayer)
				udpLayer := packet.Layer(layers.LayerTypeUDP)
				if udpLayer == nil {
					t.Log("received non-UDP packet on peer 2")
					ipLayer := packet.Layer(layers.LayerTypeIPv4)
					require.NotNil(t, ipLayer)
					icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
					require.NotNil(t, icmpLayer)
				} else {
					require.NotNil(t, udpLayer)
					payloadLayer := packet.ApplicationLayer()
					require.NotNil(t, payloadLayer)
					assert.Equal(t, acceptablePayload, payloadLayer.Payload())
				}

				// Echo back
				largeUDPPacket, err = craftUDPPacket(ip2, ip1, 65432, 23456, acceptablePayload)
				require.NoError(t, err)
				t.Log("sending UDP packet back to peer 1")
				_, err = mock2.Write(largeUDPPacket)
				require.NoError(t, err)
			case <-time.After(10 * time.Second):
				errCh <- fmt.Errorf("timeout waiting for UDP echo at peer2")
			}

			// Peer1 receives echo
			select {
			case got = <-mock1.writeCh:
				packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
				t.Log("received UDP packet on peer 1")
				ipLayer := packet.Layer(layers.LayerTypeIPv4)
				require.NotNil(t, ipLayer)
				udpLayer := packet.Layer(layers.LayerTypeUDP)
				require.NotNil(t, udpLayer)
				payloadLayer := packet.ApplicationLayer()
				require.NotNil(t, payloadLayer)
				assert.Equal(t, acceptablePayload, payloadLayer.Payload())
			case <-time.After(35 * time.Second):
				errCh <- fmt.Errorf("timeout waiting for UDP echo at peer1")
			}
			close(done)
		}()
		select {
		case <-done:
			t.Log("large UDP packet echo test completed")
		case err := <-errCh:
			t.Fatal(err)
		case <-time.After(60 * time.Second):
			t.Fatal("large UDP packet echo test did not complete in time")
		}
	})

	t.Run("test subnet io loop - concurrent packets", func(t *testing.T) {
		// Number of concurrent packets to send
		numPackets := 2
		// Start concurrent writes for peer1
		for i := 0; i < numPackets; i++ {
			// Send ping packet
			pingPacket, err := craftICMPEchoPacket(ip1, ip2, layers.ICMPv4TypeEchoRequest, []byte(fmt.Sprintf("ping-%d", i)))
			require.NoError(t, err)
			t.Logf("sending ping packet %d", i)
			go func() {
				_, err = mock1.Write(pingPacket)
				require.NoError(t, err)
			}()
		}

		for i := 0; i < numPackets; i++ {
			select {
			case got := <-mock2.writeCh:
				packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
				icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
				require.NotNil(t, icmpLayer)
				icmp, _ := icmpLayer.(*layers.ICMPv4)
				assert.Equal(t, uint8(layers.ICMPv4TypeEchoRequest), icmp.TypeCode.Type())
				// Send pong packet
			case <-time.After(5 * time.Second):
				t.Fatal("timeout waiting for ping")
			}
		}
		for i := 0; i < numPackets; i++ {
			go func(i int) {
				pongPacket, err := craftICMPEchoPacket(ip2, ip1, layers.ICMPv4TypeEchoReply, []byte(fmt.Sprintf("pong-%d", i)))
				require.NoError(t, err)
				_, err = mock2.Write(pongPacket)
				require.NoError(t, err)
			}(i)
		}

		// Wait for pong
		for i := 0; i < numPackets; i++ {
			select {
			case got := <-mock1.writeCh:
				t.Logf("received pong packet %d", i)
				packet := gopacket.NewPacket(got, layers.LayerTypeIPv4, gopacket.Default)
				icmpLayer := packet.Layer(layers.LayerTypeICMPv4)
				require.NotNil(t, icmpLayer)
				icmp, _ := icmpLayer.(*layers.ICMPv4)
				assert.Equal(t, uint8(layers.ICMPv4TypeEchoReply), icmp.TypeCode.Type())
			case <-time.After(10 * time.Second):
				t.Fatal("timeout waiting for pong")
			}
		}
	})

	require.NoError(t, peer1.Stop())
	require.NoError(t, peer2.Stop())
}

// --- MOCK NET INTERFACE ---
type MockNetInterface struct {
	name    string
	writeCh chan []byte
	readCh  chan []byte
	written [][]byte
}

func NewMockNetInterface(name string) *MockNetInterface {
	return &MockNetInterface{
		name:    name,
		writeCh: make(chan []byte, 10),
		readCh:  make(chan []byte, 10),
		written: make([][]byte, 0),
	}
}

func (m *MockNetInterface) Name() string { return m.name }
func (m *MockNetInterface) Write(b []byte) (int, error) {
	copyB := make([]byte, len(b))
	copy(copyB, b)
	m.written = append(m.written, copyB)
	m.writeCh <- copyB
	return len(b), nil
}

func (m *MockNetInterface) Read(b []byte) (int, error) {
	pkt := <-m.writeCh
	copy(b, pkt)
	return len(pkt), nil
}
func (m *MockNetInterface) Close() error            { return nil }
func (m *MockNetInterface) Up() error               { return nil }
func (m *MockNetInterface) Down() error             { return nil }
func (m *MockNetInterface) Delete() error           { return nil }
func (m *MockNetInterface) SetAddress(string) error { return nil }
func (m *MockNetInterface) SetMTU(int) error        { return nil }

// --- MOCK FACTORY ---
func mockNetInterfaceFactory(name string) (sys.NetInterface, error) {
	return NewMockNetInterface(name), nil
}

func craftTCPPacket(srcIP, dstIP string, srcPort, dstPort int, payload []byte) ([]byte, error) {
	// Craft TCP packet from ip1 to ip2
	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(srcPort),
		DstPort: layers.TCPPort(dstPort),
		Seq:     11050,
		SYN:     true,
	}
	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		SrcIP:    net.ParseIP(srcIP),
		DstIP:    net.ParseIP(dstIP),
		Protocol: layers.IPProtocolTCP,
	}
	err := tcp.SetNetworkLayerForChecksum(ip)
	if err != nil {
		return nil, err
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	err = gopacket.SerializeLayers(buf, opts, ip, tcp, gopacket.Payload(payload))
	if err != nil {
		return nil, err
	}
	tcpPacket := buf.Bytes()
	return tcpPacket, nil
}

func craftUDPPacket(srcIP, dstIP string, srcPort, dstPort int, payload []byte) ([]byte, error) {
	udp := &layers.UDP{
		SrcPort: layers.UDPPort(srcPort),
		DstPort: layers.UDPPort(dstPort),
	}
	ip := &layers.IPv4{
		Version:  4,
		IHL:      5,
		TTL:      64,
		SrcIP:    net.ParseIP(srcIP),
		DstIP:    net.ParseIP(dstIP),
		Protocol: layers.IPProtocolUDP,
	}
	err := udp.SetNetworkLayerForChecksum(ip)
	if err != nil {
		return nil, err
	}
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	err = gopacket.SerializeLayers(buf, opts, ip, udp, gopacket.Payload(payload))
	if err != nil {
		return nil, err
	}
	udpPacket := buf.Bytes()
	return udpPacket, nil
}

func TestSubnetCreationDestructionRecreation(t *testing.T) {
	// Create two peers with mock network interfaces
	peer1 := createPeer(t, 0, 3002, []multiaddr.Multiaddr{}, mockNetInterfaceFactory)
	peer2 := createPeer(t, 0, 3001, []multiaddr.Multiaddr{}, mockNetInterfaceFactory)
	require.NotNil(t, peer1)
	require.NotNil(t, peer2)

	// Start the peers
	require.NoError(t, peer1.Start())
	require.NoError(t, peer2.Start())

	// Wait a bit to ensure resources are available
	time.Sleep(500 * time.Millisecond)

	// Connect the peers to each other
	peer1Addrs := peer1.Host.Addrs()[2]
	require.NotEmpty(t, peer1Addrs)

	// Add peer1's addresses to peer2's peerstore
	peer2.Host.Peerstore().AddAddrs(
		peer1.Host.ID(),
		[]multiaddr.Multiaddr{peer1Addrs},
		peerstore.PermanentAddrTTL,
	)

	// Add peer2's addresses to peer1's peerstore
	peer2Addrs := peer2.Host.Addrs()[2]
	require.NotEmpty(t, peer2Addrs)

	peer1.Host.Peerstore().AddAddrs(
		peer2.Host.ID(),
		[]multiaddr.Multiaddr{peer2Addrs},
		peerstore.PermanentAddrTTL,
	)

	// Explicitly connect the peers
	peer2AddrInfo := peer.AddrInfo{
		ID:    peer2.Host.ID(),
		Addrs: []multiaddr.Multiaddr{peer2Addrs},
	}
	require.NoError(t, peer1.Host.Connect(context.Background(), peer2AddrInfo))

	// Wait for connection to be established
	time.Sleep(500 * time.Millisecond)

	// Verify the peers are connected
	require.Eventually(t, func() bool {
		return peer1.Host.Network().Connectedness(peer2.Host.ID()) == network.Connected
	}, 5*time.Second, 100*time.Millisecond)

	require.Eventually(t, func() bool {
		return peer2.Host.Network().Connectedness(peer1.Host.ID()) == network.Connected
	}, 5*time.Second, 100*time.Millisecond)

	createSubnetTestCommunication := func(t *testing.T) {
		t.Helper()
		// Create subnet on peer1
		err := peer1.CreateSubnet(context.TODO(), "test_subnet", "10.20.20.0/24", map[string]string{
			"10.20.20.2": peer1.Host.ID().String(),
			"10.20.20.3": peer2.Host.ID().String(),
		})
		require.NoError(t, err)

		// Create subnet on peer2
		err = peer2.CreateSubnet(context.TODO(), "test_subnet", "10.20.20.0/24", map[string]string{
			"10.20.20.2": peer1.Host.ID().String(),
			"10.20.20.3": peer2.Host.ID().String(),
		})
		require.NoError(t, err)

		// Verify IP proxy is started (peer1 should have IP proxy running)
		require.NotNil(t, peer1.ipproxy)
		require.NotNil(t, peer2.ipproxy)

		// Add peer1 to the subnet
		err = peer1.AddSubnetPeer("test_subnet", peer1.Host.ID().String(), "10.20.20.2")
		require.NoError(t, err)

		// Add peer2 to the subnet
		err = peer2.AddSubnetPeer("test_subnet", peer2.Host.ID().String(), "10.20.20.3")
		require.NoError(t, err)

		// Verify TUN interfaces are created
		peer1.subnetsmx.Lock()
		peer1Subnet, exists := peer1.subnets["test_subnet"]
		peer1.subnetsmx.Unlock()
		require.True(t, exists)
		require.Len(t, peer1Subnet.ifaces, 1)

		peer2.subnetsmx.Lock()
		peer2Subnet, exists := peer2.subnets["test_subnet"]
		peer2.subnetsmx.Unlock()
		require.True(t, exists)
		require.Len(t, peer2Subnet.ifaces, 1)

		// Create a simple ICMP ping packet using the existing function
		pingPacket, err := craftICMPEchoPacket("10.20.20.2", "10.20.20.3", layers.ICMPv4TypeEchoRequest, []byte("ping-test"))
		require.NoError(t, err)

		// Write packet to peer1's TUN interface
		peer1.subnetsmx.Lock()
		peer1Subnet = peer1.subnets["test_subnet"]
		peer1.subnetsmx.Unlock()

		peer1Subnet.mx.Lock()
		peer1Iface, exists := peer1Subnet.ifaces["10.20.20.2"]
		peer1Subnet.mx.Unlock()
		require.True(t, exists)

		// Write ping packet to TUN interface
		n, err := peer1Iface.tun.Write(pingPacket)
		require.NoError(t, err)
		require.Greater(t, n, 0)

		// Wait for packet to be processed and check if peer2's mock received it
		time.Sleep(200 * time.Millisecond)

		// Get peer2's mock interface
		peer2.subnetsmx.Lock()
		peer2Subnet = peer2.subnets["test_subnet"]
		peer2.subnetsmx.Unlock()

		peer2Subnet.mx.Lock()
		peer2Iface, exists := peer2Subnet.ifaces["10.20.20.3"]
		peer2Subnet.mx.Unlock()
		require.True(t, exists)

		// Check if peer2's mock interface received the packet
		select {
		case receivedPacket := <-peer2Iface.tun.(*MockNetInterface).writeCh:
			require.NotNil(t, receivedPacket)
			require.Greater(t, len(receivedPacket), 0)
			t.Logf("Peer2's mock interface received packet of length %d", len(receivedPacket))
		case <-time.After(30 * time.Second):
			t.Log("Peer2's mock interface did not receive packet within timeout")
			t.Fail()
		}

		err = peer1.DestroySubnet("test_subnet")
		require.NoError(t, err)

		// Destroy subnet on peer2
		err = peer2.DestroySubnet("test_subnet")
		require.NoError(t, err)

		// Verify IP proxy is stopped (should be nil after last subnet is destroyed)
		peer1.subnetsmx.Lock()
		peer1SubnetCount := len(peer1.subnets)
		peer1.subnetsmx.Unlock()
		require.Equal(t, 0, peer1SubnetCount)

		peer2.subnetsmx.Lock()
		peer2SubnetCount := len(peer2.subnets)
		peer2.subnetsmx.Unlock()
		require.Equal(t, 0, peer2SubnetCount)
		t.Log("create subnet and test communication: success")
	}

	// Test 1: Create subnets and verify IP proxy startup
	t.Run("create subnet and test communication", createSubnetTestCommunication)

	// Test 4: Destroy subnets and verify IP proxy shutdown
	t.Run("destroy and recreate subnet and test communication", func(t *testing.T) {
		createSubnetTestCommunication(t)

		t.Log("recreating subnet and test communication")

		// Recreate subnet on peer1
		err := peer1.CreateSubnet(context.TODO(), "test_subnet_2", "10.30.30.0/24", map[string]string{
			"10.30.30.2": peer1.Host.ID().String(),
			"10.30.30.3": peer2.Host.ID().String(),
		})
		require.NoError(t, err)

		// Recreate subnet on peer2
		err = peer2.CreateSubnet(context.TODO(), "test_subnet_2", "10.30.30.0/24", map[string]string{
			"10.30.30.2": peer1.Host.ID().String(),
			"10.30.30.3": peer2.Host.ID().String(),
		})
		require.NoError(t, err)

		// Verify IP proxy is restarted
		require.NotNil(t, peer1.ipproxy)
		require.NotNil(t, peer2.ipproxy)

		// Add peers to the new subnet
		err = peer1.AddSubnetPeer("test_subnet_2", peer1.Host.ID().String(), "10.30.30.2")
		require.NoError(t, err)

		err = peer2.AddSubnetPeer("test_subnet_2", peer2.Host.ID().String(), "10.30.30.3")
		require.NoError(t, err)

		// Verify TUN interfaces are created for the new subnet
		peer1.subnetsmx.Lock()
		peer1Subnet2, exists := peer1.subnets["test_subnet_2"]
		peer1.subnetsmx.Unlock()
		require.True(t, exists)
		require.Len(t, peer1Subnet2.ifaces, 1)

		peer2.subnetsmx.Lock()
		peer2Subnet2, exists := peer2.subnets["test_subnet_2"]
		peer2.subnetsmx.Unlock()
		require.True(t, exists)
		require.Len(t, peer2Subnet2.ifaces, 1)

		// Create a ping packet for the new subnet
		pingPacket, err := craftICMPEchoPacket("10.30.30.2", "10.30.30.3", layers.ICMPv4TypeEchoRequest, []byte("ping-test-2"))
		require.NoError(t, err)

		peer1.subnetsmx.Lock()
		peer1Subnet2 = peer1.subnets["test_subnet_2"]
		peer1.subnetsmx.Unlock()

		peer1Subnet2.mx.Lock()
		peer1Iface2, exists := peer1Subnet2.ifaces["10.30.30.2"]
		peer1Subnet2.mx.Unlock()
		require.True(t, exists)

		// Write ping packet to the new TUN interface
		n, err := peer1Iface2.tun.Write(pingPacket)
		require.NoError(t, err)
		require.Greater(t, n, 0)

		// Wait for packet to be processed and check if peer2's new mock received it
		time.Sleep(200 * time.Millisecond)

		// Get peer2's new mock interface
		peer2.subnetsmx.Lock()
		peer2Subnet2 = peer2.subnets["test_subnet_2"]
		peer2.subnetsmx.Unlock()

		peer2Subnet2.mx.Lock()
		peer2Iface2, exists := peer2Subnet2.ifaces["10.30.30.3"]
		peer2Subnet2.mx.Unlock()
		require.True(t, exists)

		// Check if peer2's new mock interface received the packet
		select {
		case receivedPacket := <-peer2Iface2.tun.(*MockNetInterface).writeCh:
			require.NotNil(t, receivedPacket)
			require.Greater(t, len(receivedPacket), 0)
			t.Logf("Peer2's new mock interface received packet of length %d", len(receivedPacket))
		case <-time.After(30 * time.Second):
			t.Log("Peer2's new mock interface did not receive packet within timeout")
			t.Fail()
		}

		// Destroy the recreated subnets
		err = peer1.DestroySubnet("test_subnet_2")
		require.NoError(t, err)

		err = peer2.DestroySubnet("test_subnet_2")
		require.NoError(t, err)

		// Verify final state
		peer1.subnetsmx.Lock()
		peer1SubnetCount := len(peer1.subnets)
		peer1.subnetsmx.Unlock()
		require.Equal(t, 0, peer1SubnetCount)

		peer2.subnetsmx.Lock()
		peer2SubnetCount := len(peer2.subnets)
		peer2.subnetsmx.Unlock()
		require.Equal(t, 0, peer2SubnetCount)
	})

	// Cleanup
	require.NoError(t, peer1.Stop())
	require.NoError(t, peer2.Stop())
}

func TestSubnetDNSCreationAndResolution(t *testing.T) {
	// Create a peer with mock network interfaces
	peer1 := createPeer(t, 0, 3004, []multiaddr.Multiaddr{}, mockNetInterfaceFactory)
	require.NotNil(t, peer1)

	// Start the peer
	require.NoError(t, peer1.Start())

	// Wait a bit to ensure resources are available
	time.Sleep(500 * time.Millisecond)

	// Create subnet with DNS records
	dnsRecords := map[string]string{
		"test.local": "10.0.0.100",
		"api.local":  "10.0.0.101",
		"db.local":   "10.0.0.102",
	}

	err := peer1.CreateSubnet(context.Background(), "test_subnet", "10.0.0.0/24", map[string]string{})
	require.NoError(t, err)

	// Add DNS records to the subnet
	err = peer1.AddSubnetDNSRecords("test_subnet", dnsRecords)
	require.NoError(t, err)

	// Add a peer to the subnet to create a TUN interface
	err = peer1.AddSubnetPeer("test_subnet", peer1.Host.ID().String(), "10.0.0.2")
	require.NoError(t, err)

	// Verify subnet was created with DNS records
	peer1.subnetsmx.Lock()
	subnet, exists := peer1.subnets["test_subnet"]
	peer1.subnetsmx.Unlock()
	require.True(t, exists)

	// Verify DNS records are stored in the subnet
	subnet.dnsmx.RLock()
	require.Equal(t, len(dnsRecords), len(subnet.dnsRecords))
	for domain, ip := range dnsRecords {
		require.Equal(t, ip, subnet.dnsRecords[domain])
	}
	subnet.dnsmx.RUnlock()

	// Test DNS resolution by crafting a DNS query packet
	t.Run("test DNS resolution for existing domain", func(t *testing.T) {
		// Create a DNS query for "test.local"
		query := new(dns.Msg)
		query.SetQuestion(dns.Fqdn("test.local"), dns.TypeA)
		queryPacket, err := query.Pack()
		require.NoError(t, err)

		// Create a UDP packet containing the DNS query
		udpLayer := &layers.UDP{
			SrcPort: layers.UDPPort(12345),
			DstPort: layers.UDPPort(53), // DNS port
		}

		ipLayer := &layers.IPv4{
			Version:  4,
			TTL:      64,
			SrcIP:    net.ParseIP("10.0.0.2"),
			DstIP:    net.ParseIP("10.0.0.1"),
			Protocol: layers.IPProtocolUDP,
		}

		err = udpLayer.SetNetworkLayerForChecksum(ipLayer)
		require.NoError(t, err)

		buffer := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{
			ComputeChecksums: true,
			FixLengths:       true,
		}

		err = gopacket.SerializeLayers(buffer, opts,
			ipLayer,
			udpLayer,
			gopacket.Payload(queryPacket),
		)
		require.NoError(t, err)

		dnsPacket := buffer.Bytes()

		// Get the real TUN interface
		subnet.mx.Lock()
		iface, exists := subnet.ifaces["10.0.0.2"]
		subnet.mx.Unlock()
		require.True(t, exists)

		// Write the DNS query packet to the TUN interface
		n, err := iface.tun.Write(dnsPacket)
		require.NoError(t, err)
		require.Greater(t, n, 0)

		// Read the DNS response from the TUN interface with timeout
		responseBuf := make([]byte, 1500)
		respLenCh := make(chan int, 1)
		errCh := make(chan error, 1)
		go func() {
			n, err := iface.tun.Read(responseBuf)
			if err != nil {
				errCh <- err
				return
			}
			respLenCh <- n
		}()
		var respLen int
		select {
		case respLen = <-respLenCh:
			// ok
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(30 * time.Second):
			t.Fatal("timeout waiting for DNS response")
		}
		require.Greater(t, respLen, 0)
		responsePacket := responseBuf[:respLen]

		// Parse the response packet
		packet := gopacket.NewPacket(responsePacket, layers.LayerTypeIPv4, gopacket.Default)
		if err := packet.ErrorLayer(); err != nil {
			require.NoError(t, err.Error())
		}

		// Extract the UDP payload (DNS response)
		udpLayerFromPacket := packet.Layer(layers.LayerTypeUDP)
		require.NotNil(t, udpLayerFromPacket)
		udp, ok := udpLayerFromPacket.(*layers.UDP)
		require.True(t, ok)
		require.Equal(t, uint16(53), uint16(udp.SrcPort)) // DNS response from port 53

		// Parse the DNS response
		dnsResponse := new(dns.Msg)
		err = dnsResponse.Unpack(udp.Payload)
		require.NoError(t, err)

		// Verify the DNS response
		require.Equal(t, 1, len(dnsResponse.Answer))
		aRecord, ok := dnsResponse.Answer[0].(*dns.A)
		require.True(t, ok)
		require.Equal(t, "test.local.", aRecord.Header().Name)
		require.Equal(t, "10.0.0.100", aRecord.A.String())
	})

	t.Run("test DNS resolution for non-existent domain", func(t *testing.T) {
		// Create a DNS query for a non-existent domain
		query := new(dns.Msg)
		query.SetQuestion(dns.Fqdn("nonexistent.local"), dns.TypeA)
		queryPacket, err := query.Pack()
		require.NoError(t, err)

		// Create a UDP packet containing the DNS query
		udpLayer := &layers.UDP{
			SrcPort: layers.UDPPort(12346),
			DstPort: layers.UDPPort(53),
		}

		ipLayer := &layers.IPv4{
			Version:  4,
			TTL:      64,
			SrcIP:    net.ParseIP("10.0.0.2"),
			DstIP:    net.ParseIP("10.0.0.1"),
			Protocol: layers.IPProtocolUDP,
		}

		err = udpLayer.SetNetworkLayerForChecksum(ipLayer)
		require.NoError(t, err)

		buffer := gopacket.NewSerializeBuffer()
		opts := gopacket.SerializeOptions{
			ComputeChecksums: true,
			FixLengths:       true,
		}

		err = gopacket.SerializeLayers(buffer, opts,
			ipLayer,
			udpLayer,
			gopacket.Payload(queryPacket),
		)
		require.NoError(t, err)

		dnsPacket := buffer.Bytes()

		// Get the real TUN interface
		subnet.mx.Lock()
		iface, exists := subnet.ifaces["10.0.0.2"]
		subnet.mx.Unlock()
		require.True(t, exists)

		// Write the DNS query packet to the TUN interface
		n, err := iface.tun.Write(dnsPacket)
		require.NoError(t, err)
		require.Greater(t, n, 0)

		// Wait for the DNS response to be processed
		responseBuf := make([]byte, 1500)
		respLenCh := make(chan int, 1)
		errCh := make(chan error, 1)
		go func() {
			n, err := iface.tun.Read(responseBuf)
			if err != nil {
				errCh <- err
				return
			}
			respLenCh <- n
		}()
		var respLen int
		select {
		case respLen = <-respLenCh:
			// ok
		case err := <-errCh:
			require.NoError(t, err)
		case <-time.After(30 * time.Second):
			t.Fatal("timeout waiting for DNS response")
		}
		require.Greater(t, respLen, 0)
		responsePacket := responseBuf[:respLen]

		// Parse the response packet
		packet := gopacket.NewPacket(responsePacket, layers.LayerTypeIPv4, gopacket.Default)
		if err := packet.ErrorLayer(); err != nil {
			require.NoError(t, err.Error())
		}

		// Extract the UDP payload (DNS response)
		udpLayerFromPacket := packet.Layer(layers.LayerTypeUDP)
		require.NotNil(t, udpLayerFromPacket)
		udp, ok := udpLayerFromPacket.(*layers.UDP)
		require.True(t, ok)
		require.Equal(t, uint16(53), uint16(udp.SrcPort))

		// Parse the DNS response
		dnsResponse := new(dns.Msg)
		err = dnsResponse.Unpack(udp.Payload)
		require.NoError(t, err)

		// Verify the DNS response indicates NXDOMAIN
		require.Equal(t, 0, len(dnsResponse.Answer))
		require.Equal(t, dns.RcodeNameError, dnsResponse.Rcode)
	})

	// Cleanup
	require.NoError(t, peer1.Stop())
}

func TestSubnetDNSResolutionWithDig(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("requires root privileges")
	}

	var peer1 *Libp2p
	dnsRecords := map[string]string{
		"test.local": "10.0.0.100",
		"api.local":  "10.0.0.101",
		"db.local":   "10.0.0.102",
	}

	t.Run("create peer with real interface", func(t *testing.T) {
		peer1 = createPeer(t, 0, 3001, []multiaddr.Multiaddr{})
		require.NotNil(t, peer1)
		require.NoError(t, peer1.Start())
		<-time.After(5 * time.Second)
	})

	t.Run("create subnet", func(t *testing.T) {
		err := peer1.CreateSubnet(context.Background(), "test_subnet_dig", "10.0.0.0/24", map[string]string{})
		require.NoError(t, err)
	})

	t.Run("add DNS records", func(t *testing.T) {
		err := peer1.AddSubnetDNSRecords("test_subnet_dig", dnsRecords)
		require.NoError(t, err)
	})

	t.Run("add peer to subnet", func(t *testing.T) {
		err := peer1.AddSubnetPeer("test_subnet_dig", peer1.Host.ID().String(), "10.0.0.2")
		require.NoError(t, err)
	})

	t.Run("test DNS resolution using dig", func(t *testing.T) {
		time.Sleep(1 * time.Second)
		cmd := exec.Command("dig", "+short", "@10.0.0.1", "test.local")
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "dig command failed: %s", string(output))
		result := strings.TrimSpace(string(output))
		require.Equal(t, "10.0.0.100", result)

		cmd = exec.Command("dig", "+short", "@10.0.0.1", "api.local")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "dig command failed: %s", string(output))
		result = strings.TrimSpace(string(output))
		require.Equal(t, "10.0.0.101", result)

		cmd = exec.Command("dig", "+short", "@10.0.0.1", "db.local")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "dig command failed: %s", string(output))
		result = strings.TrimSpace(string(output))
		require.Equal(t, "10.0.0.102", result)

		cmd = exec.Command("dig", "+short", "@10.0.0.1", "nonexistent.local")
		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "dig command failed: %s", string(output))
		result = strings.TrimSpace(string(output))
		require.Equal(t, "", result)
	})

	t.Run("cleanup", func(t *testing.T) {
		require.NoError(t, peer1.Stop())
	})
}
