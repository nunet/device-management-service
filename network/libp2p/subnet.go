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
	"encoding/binary"
	"errors"
	"fmt"
	"io/fs"
	"math/rand"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	network "github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"

	"gitlab.com/nunet/device-management-service/lib/sys"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	IfaceMTU = 1420

	PacketExchangeProtocolID = "/dms/subnet/packet-exchange/0.0.1"
)

type subnet struct {
	ctx     context.Context
	network *Libp2p

	info struct {
		id     string
		rtable SubnetRoutingTable
	}

	mx     sync.Mutex
	ifaces map[string]struct {
		tun    *sys.NetInterface
		ctx    context.Context
		cancel context.CancelFunc
	}

	io struct {
		mx      sync.RWMutex
		streams map[string]*struct {
			mx     sync.Mutex
			stream network.Stream
		}
	}

	dnsmx      sync.RWMutex
	dnsRecords map[string]string

	portMapping map[string]*struct {
		destPort string
		destIP   string
		srcIP    string
	}
}

func (l *Libp2p) CreateSubnet(ctx context.Context, subnetID string, routingTable map[string]string) error {
	if _, ok := l.subnets[subnetID]; ok {
		return fmt.Errorf("subnet with ID %s already exists", subnetID)
	}

	s := newSubnet(ctx, l)
	s.info.id = subnetID

	for ip, peerctx := range routingTable {
		peerID, err := peer.Decode(peerctx)
		if err != nil {
			return fmt.Errorf("failed to decode peer ID %s: %w", peerctx, err)
		}

		s.info.rtable.Add(peerID, ip)
	}

	if atomic.CompareAndSwapInt32(&l.isSubnetWriteProtocolRegistered, 0, 1) {
		err := s.network.RegisterStreamMessageHandler(
			types.MessageType(PacketExchangeProtocolID),
			func(stream network.Stream) {
				l.writePackets(stream)
			})
		if err != nil {
			return err
		}
	}
	l.subnets[subnetID] = s
	return nil
}

func (l *Libp2p) DestroySubnet(subnetID string) error {
	s, ok := l.subnets[subnetID]
	if !ok {
		return fmt.Errorf("subnet with ID %s does not exist", subnetID)
	}

	for ip := range s.ifaces {
		s.ifaces[ip].cancel()
		_ = s.ifaces[ip].tun.Down()
		_ = s.ifaces[ip].tun.Delete()
	}

	s.mx.Lock()
	s.ifaces = make(map[string]struct {
		tun    *sys.NetInterface
		ctx    context.Context
		cancel context.CancelFunc
	})
	s.mx.Unlock()

	s.io.mx.Lock()
	for _, ms := range s.io.streams {
		ms.mx.Lock()
		_ = ms.stream.Reset()
		ms.mx.Unlock()
	}
	s.io.streams = make(map[string]*struct {
		mx     sync.Mutex
		stream network.Stream
	})
	s.io.mx.Unlock()

	s.dnsmx.Lock()
	s.dnsRecords = make(map[string]string)
	s.dnsmx.Unlock()

	s.info.rtable.Clear()

	for sourcePort, mapping := range s.portMapping {
		_ = l.UnmapPort(subnetID, "tcp", mapping.srcIP, sourcePort, mapping.destIP, mapping.destPort)
	}

	l.UnregisterMessageHandler(PacketExchangeProtocolID)
	delete(l.subnets, subnetID)
	return nil
}

func (l *Libp2p) AddSubnetPeer(subnetID, peerID, ip string) error {
	s, ok := l.subnets[subnetID]
	if !ok {
		return fmt.Errorf("subnet with ID %s does not exist", subnetID)
	}

	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return fmt.Errorf("invalid IP address %s", ip)
	}

	peerIDObj, err := peer.Decode(peerID)
	if err != nil {
		return fmt.Errorf("failed to decode peer ID %s: %w", peerID, err)
	}

	s.info.rtable.Add(peerIDObj, ip)

	ifaces, err := sys.GetNetInterfaces()
	if err != nil {
		return err
	}

	takenNames := make([]string, 0)
	for _, iface := range ifaces {
		takenNames = append(takenNames, iface.Name)
	}

	log.Debug("finding proper iface name for TUN interface", "taken_names", takenNames)
	name, err := generateUniqueName(takenNames)
	if err != nil {
		return fmt.Errorf("failed to generate unique name for TUN interface: %w", err)
	}

	log.Debug("Creating TUN interface", "name", name)
	address := fmt.Sprintf("%s/24", ipAddr.String())

	iface, err := sys.NewTunTapInterface(name, sys.NetTunMode, false)
	if err != nil {
		return fmt.Errorf("failed to create tun interface: %w", err)
	}

	err = iface.SetAddress(address)
	if err != nil {
		return fmt.Errorf("failed to set address on tun interface: %w", err)
	}

	err = iface.SetMTU(IfaceMTU)
	if err != nil {
		return fmt.Errorf("failed to set MTU on tun interface: %w", err)
	}

	if err := iface.Up(); err != nil {
		return fmt.Errorf("failed to bring up tun interface: %w", err)
	}

	ctx, cancel := context.WithCancel(s.ctx)
	s.mx.Lock()
	s.ifaces[ipAddr.String()] = struct {
		tun    *sys.NetInterface
		ctx    context.Context
		cancel context.CancelFunc
	}{
		tun:    iface,
		ctx:    ctx,
		cancel: cancel,
	}
	s.mx.Unlock()

	go s.readPackets(ctx, iface)

	return nil
}

func (l *Libp2p) RemoveSubnetPeer(subnetID, peerID, ip string) error {
	s, ok := l.subnets[subnetID]
	if !ok {
		return fmt.Errorf("subnet with ID %s does not exist", subnetID)
	}

	peerIDObj, err := peer.Decode(peerID)
	if err != nil {
		return fmt.Errorf("failed to decode peer ID %s: %w", peerID, err)
	}

	ips, ok := s.info.rtable.Get(peerIDObj)
	if !ok {
		return fmt.Errorf("peer with ID %s is not in the subnet", peerID)
	}

	for _, i := range ips {
		if i == ip {
			goto delete_iface
		}
	}

	return nil

delete_iface:
	s.mx.Lock()
	iface, ok := s.ifaces[ip]
	if ok {
		iface.cancel()
		_ = iface.tun.Down()
		_ = iface.tun.Delete()
		delete(s.ifaces, ip)
	}
	s.mx.Unlock()

	s.info.rtable.Remove(peerIDObj, ip)
	return nil
}

func (l *Libp2p) AcceptSubnetPeer(subnetID, peerID, ip string) error {
	s, ok := l.subnets[subnetID]
	if !ok {
		return fmt.Errorf("subnet with ID %s does not exist", subnetID)
	}

	ipAddr := net.ParseIP(ip)
	if ipAddr == nil {
		return fmt.Errorf("invalid IP address %s", ip)
	}

	peerIDObj, err := peer.Decode(peerID)
	if err != nil {
		return fmt.Errorf("failed to decode peer ID %s: %w", peerID, err)
	}

	s.info.rtable.Add(peerIDObj, ip)

	return nil
}

// AddDNSRecord adds a dns record to our local resolver
func (l *Libp2p) AddSubnetDNSRecord(subnetID, name, ip string) error {
	s, ok := l.subnets[subnetID]
	if !ok {
		return fmt.Errorf("subnet with ID %s does not exist", subnetID)
	}

	s.dnsmx.Lock()
	s.dnsRecords[name] = ip
	s.dnsmx.Unlock()

	return nil
}

func (l *Libp2p) RemoveSubnetDNSRecord(subnetID, name string) error {
	s, ok := l.subnets[subnetID]
	if !ok {
		return fmt.Errorf("subnet with ID %s does not exist", subnetID)
	}

	s.dnsmx.Lock()
	delete(s.dnsRecords, name)
	s.dnsmx.Unlock()

	return nil
}

func (l *Libp2p) writePackets(stream network.Stream) {
	IDSize := make([]byte, 2)
	// read_subnet_id
	// Read the incoming packet's size as a binary value.
	_, err := stream.Read(IDSize)
	if err != nil {
		log.Error("failed to read subnet id size from stream", err)
		_ = stream.Reset()
		return
	}

	// Decode the incoming packet's size from binary.
	size := binary.LittleEndian.Uint16(IDSize)
	subnetID := make([]byte, size)

	// Read in the packet until completion.
	var IDLen uint16
	for IDLen < size {
		tmp, err := stream.Read(subnetID[IDLen:size])
		IDLen += uint16(tmp)
		if err != nil {
			log.Error("failed to read subnet id from stream", err)
			_ = stream.Reset()
			return
		}
	}

	// retrieve subnet object
	subnet, ok := l.subnets[string(subnetID)]
	if !ok {
		log.Errorf("unrecognized subnet id %s, subnet does not exist on this host", string(subnetID))
		_ = stream.Reset()
		return
	}

	subnet.writePackets(stream)
}

func newSubnet(ctx context.Context, l *Libp2p) *subnet {
	return &subnet{
		ctx:     ctx,
		network: l,
		info: struct {
			id     string
			rtable SubnetRoutingTable
		}{
			rtable: NewRoutingTable(),
		},
		ifaces: make(map[string]struct {
			tun    *sys.NetInterface
			ctx    context.Context
			cancel context.CancelFunc
		}),
		io: struct {
			mx      sync.RWMutex
			streams map[string]*struct {
				mx     sync.Mutex
				stream network.Stream
			}
		}{
			streams: make(map[string]*struct {
				mx     sync.Mutex
				stream network.Stream
			}),
		},
		dnsRecords: map[string]string{},
		portMapping: map[string]*struct {
			destPort string
			destIP   string
			srcIP    string
		}{},
	}
}

func (s *subnet) readPackets(ctx context.Context, iface *sys.NetInterface) {
	for {
		select {
		case <-ctx.Done():
			log.Debug("context done, abandoning read loop...", "subnet", s.info.id)
			return
		default:
			{
				packet := make([]byte, 1420)
				// Read in a packet from the tun device.
				plen, err := iface.Iface.Read(packet)
				if errors.Is(err, fs.ErrClosed) {
					time.Sleep(1 * time.Second)
					log.Debug("tun device closed, abandoning read loop...", err, "subnet", s.info.id)
					return
				} else if err != nil {
					log.Error("failed to read packet from tun device ", err, "subnet", s.info.id)
					continue
				}

				if plen == 0 {
					continue
				}

				srcPort, destPort, srcIP, destIP, err := s.parseIPPacket(packet)
				if err != nil {
					log.Error("failed to parse IP packet", err)
					continue
				}

				log.Debug(
					"read packet from tun device",
					"tun", iface.Iface.Name(),
					"subnet", s.info.id,
					"destIP", destIP,
					"destPort", destPort,
					"srcIP", srcIP,
					"srcPort", srcPort,
				)

				if destIP != "10.0.0.1" && destPort != 53 {
					s.Route(destIP, packet, plen)
					continue
				}

				log.Debug(
					"handling DNS query",
					"subnet", s.info.id,
					"destIP", destIP,
					"destPort", destPort,
					"srcIP", srcIP,
					"srcPort", srcPort,
				)

				if err := s.handleDNSQueries(iface, packet, plen); err != nil {
					log.Error("failed to handle DNS query", err)
				}
			}
		}
	}
}

func (s *subnet) handleDNSQueries(iface *sys.NetInterface, packet []byte, packetlen int) error {
	s.dnsmx.RLock()
	payload, err := handleDNSQuery(packet[28:packetlen], s.dnsRecords)
	s.dnsmx.RUnlock()
	if err != nil {
		return err
	}

	srcPort, destPort, srcIP, destIP, err := s.parseIPPacket(packet)
	if err != nil {
		return err
	}

	ipLayer := &layers.IPv4{
		Version:  4,
		TTL:      64,
		SrcIP:    net.ParseIP(destIP),
		DstIP:    net.ParseIP(srcIP),
		Protocol: layers.IPProtocolUDP,
	}

	udpLayer := &layers.UDP{
		SrcPort: layers.UDPPort(destPort),
		DstPort: layers.UDPPort(srcPort),
	}

	// Set the UDP checksum
	err = udpLayer.SetNetworkLayerForChecksum(ipLayer)
	if err != nil {
		return err
	}

	// Create the packet
	buffer := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{
		ComputeChecksums: true,
		FixLengths:       true,
	}

	err = gopacket.SerializeLayers(buffer, opts,
		ipLayer,
		udpLayer,
		gopacket.Payload(payload),
	)
	if err != nil {
		return err
	}

	_, _ = iface.Iface.Write(buffer.Bytes())
	return nil
}

func (s *subnet) Route(destIP string, packet []byte, plen int) {
	log.Debug("routing packet", "subnet", s.info.id, "dstIP", destIP)
	// check if present in our tuns table first
	defer s.mx.Unlock()
	s.mx.Lock()
	if _, ok := s.ifaces[destIP]; ok {
		log.Debug("found destination ip in tuns table", "subnet", s.info.id, "dstIP", destIP)
		// if so, write to the tun
		_, _ = s.ifaces[destIP].tun.Iface.Write(packet[:plen])
		return
	}

	// if else check if present in our routing table
	peerID, ok := s.info.rtable.GetByIP(destIP)
	if !ok {
		log.Debug("unrecognized destination ip", "subnet", s.info.id, "dstIP", destIP)
		return
	}

	log.Debugf("found destination ip in routing table", "subnet", s.info.id, "dstIP", destIP, "peerID", peerID.String())

	go s.redirectPacketToStream(s.ctx, peerID, packet, plen)
}

func (s *subnet) redirectPacketToStream(ctx context.Context, dst peer.ID, packet []byte, plen int) {
	// Check if we already have an open connection to the destination peer.
	defer s.io.mx.Unlock()
	s.io.mx.Lock()
	ms, ok := s.io.streams[dst.String()]
	if ok {
		log.Debug("found existing stream to destination peer", "subnet", s.info.id, "dst", dst.String())
		if func() bool {
			ms.mx.Lock()
			defer ms.mx.Unlock()
			_ = ms.stream.SetWriteDeadline(time.Now().Add(time.Second))
			// Write out the packet's length to the libp2p stream to ensure
			// we know the full size of the packet at the other end.
			err := binary.Write(ms.stream, binary.LittleEndian, uint16(len(s.info.id)))
			if err == nil {
				// Write the packet out to the libp2p stream.
				// If everything succeeds continue on to the next packet.
				_, _ = (ms.stream).Write([]byte(s.info.id))
			} else {
				// If we encounter an error when writing to a stream we should
				// close that stream and delete it from the active stream map.
				_ = ms.stream.Reset()
				delete(s.io.streams, dst.String())
				return false
			}

			// Write out the packet's length to the libp2p stream to ensure
			// we know the full size of the packet at the other end.
			err = binary.Write(ms.stream, binary.LittleEndian, uint16(plen))
			if err == nil {
				// Write the packet out to the libp2p stream.
				// If everything succeeds continue on to the next packet.
				_, err = (ms.stream).Write(packet[:plen])
				if err == nil {
					return true
				}
			}
			// If we encounter an error when writing to a stream we should
			// close that stream and delete it from the active stream map.
			ms.stream.Close()
			delete(s.io.streams, dst.String())
			return false
		}() {
			return
		}
	}

	log.Debug("no existing stream to destination peer", "subnet", s.info.id, "dst", dst.String())

	addrs, err := s.network.ResolveAddress(ctx, dst.String())
	if err != nil {
		log.Error("failed to resolve peer address", err, "subnet", s.info.id, "dst", dst.String())
		return
	}

	protocolID := types.MessageType(PacketExchangeProtocolID)
	stream, err := s.network.OpenStream(ctx, addrs[0], protocolID)
	if err != nil {
		log.Error("failed to open stream", err, "subnet", s.info.id, "dst", dst.String())
		return
	}

	_ = stream.SetWriteDeadline(time.Now().Add(time.Second))

	// Write packet length
	err = binary.Write(stream, binary.LittleEndian, uint16(len([]byte(s.info.id))))
	if err != nil {
		log.Error("failed to write subnet id length", err, "subnet", s.info.id, "dst", dst.String())
		stream.Close()
		return
	}

	// Write the packet
	_, err = stream.Write([]byte(s.info.id))
	if err != nil {
		log.Error("failed to write subnet id", err, "subnet", s.info.id, "dst", dst.String())
		stream.Close()
		return
	}

	// Write packet length
	err = binary.Write(stream, binary.LittleEndian, uint16(plen))
	if err != nil {
		log.Error("failed to write packet length", err, "subnet", s.info.id, "dst", dst.String())
		stream.Close()
		return
	}

	// Write the packet
	_, err = stream.Write(packet[:plen])
	if err != nil {
		log.Error("failed to write packet", err, "subnet", s.info.id, "dst", dst.String())
		stream.Close()
		return
	}

	// If all succeeds when writing the packet to the stream
	// we should reuse this stream by adding it active streams map.
	s.io.streams[dst.String()] = &struct {
		mx     sync.Mutex
		stream network.Stream
	}{
		mx:     sync.Mutex{},
		stream: stream,
	}
}

func (s *subnet) writePackets(stream network.Stream) {
	defer stream.Close()

	if _, ok := s.info.rtable.Get(stream.Conn().RemotePeer()); !ok {
		log.Debug("unrecognized source peer", "subnet", s.info.id, "src", stream.Conn().RemotePeer().String())
		_ = stream.Reset()
		return
	}

	packet := make([]byte, 1420)
	packetSize := make([]byte, 2)
	for {
		select {
		case <-s.ctx.Done():
			log.Debug("context done", "subnet", s.info.id)
			_ = stream.Reset()
			return

		default:
			{
				// read_packet
				// Read the incoming packet's size as a binary value.
				_, err := stream.Read(packetSize)
				if err != nil {
					log.Error("failed to read packet size from stream", err, "subnet", s.info.id)
					_ = stream.Reset()
					return
				}

				// Decode the incoming packet's size from binary.
				size := binary.LittleEndian.Uint16(packetSize)

				// Read in the packet until completion.
				var plen uint16
				for plen < size {
					tmp, err := stream.Read(packet[plen:size])
					plen += uint16(tmp)
					if err != nil {
						log.Error("failed to read packet from stream", err, "subnet", s.info.id)
						_ = stream.Reset()
						return
					}
				}
				_ = stream.SetWriteDeadline(time.Now().Add(time.Second))

				log.Debug("read packet from stream", "subnet", s.info.id, "src", stream.Conn().RemotePeer().String())

				log.Debug("read packet from stream", "subnet", s.info.id, "src", stream.Conn().RemotePeer().String())

				// write_packet
				destIP := net.IPv4(packet[16], packet[17], packet[18], packet[19]).String()

				// retrieve proper tun and write to it
				// if no tun is found, drop the packet
				s.mx.Lock()
				if iface, ok := s.ifaces[destIP]; ok {
					log.Debug("writing packet to tun device", "tun", iface.tun.Iface.Name(), "subnet", s.info.id, "dstIP", destIP)
					_, _ = iface.tun.Iface.Write(packet[:plen])
				} else {
					// drop the packet
					log.Debug("unrecognized destination ip, no tun device found for ip", "subnet", s.info.id, "dstIP", destIP)
				}
				s.mx.Unlock()
			}
		}
	}
}

func (s *subnet) parseIPPacket(rawPacket []byte) (srcPort int, destPort int, srcIP string, destIP string, err error) {
	// Create a packet object from the raw data
	packet := gopacket.NewPacket(rawPacket, layers.LayerTypeIPv4, gopacket.Default)
	if err := packet.ErrorLayer(); err != nil {
		return 0, 0, "", "", fmt.Errorf("failed to decode packet: %s", err)
	}

	// Get IP layer
	ipLayer := packet.Layer(layers.LayerTypeIPv4)
	if ipLayer != nil {
		ip, _ := ipLayer.(*layers.IPv4)
		srcIP = ip.SrcIP.String()
		destIP = ip.DstIP.String()
	}

	// Get TCP layer
	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		srcPort = int(udp.SrcPort)
		destPort = int(udp.DstPort)
	}

	return
}

// stringSliceContains checks if a string is in a slice of strings
func stringSliceContains(slice []string, word string) bool {
	for _, s := range slice {
		if s == word {
			return true
		}
	}
	return false
}

func generateUniqueName(takenList []string) (string, error) {
	var retries int
	var candidate string
	i := 30
	for {
		candidate = fmt.Sprintf("dms%d", rand.Intn(i)) //nolint:gosec
		if !stringSliceContains(takenList, candidate) {
			break
		}

		retries++
		if retries > 30 {
			i += 30
		}
		if retries > 100 {
			return "", fmt.Errorf("failed to generate unique name")
		}
	}
	return candidate, nil
}
