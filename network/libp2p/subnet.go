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
	"errors"
	"fmt"
	"io/fs"
	"math"
	"math/rand"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	connectip "github.com/quic-go/connect-ip-go"
	"github.com/quic-go/quic-go"
	"github.com/quic-go/quic-go/http3"
	"github.com/yosida95/uritemplate/v3"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	peer "github.com/libp2p/go-libp2p/core/peer"

	"gitlab.com/nunet/device-management-service/utils/sys"
)

const (
	IfaceMTU      = 1420
	MaxPacketSize = 2 * 1420 // Consistent packet size limit
)

type NetInterfaceFactory func(name string) (sys.NetInterface, error)

type subnet struct {
	ctx     context.Context
	network *Libp2p

	info struct {
		id     string
		rtable SubnetRoutingTable
		cidr   *net.IPNet
	}

	mx     sync.Mutex
	ifaces map[string]struct {
		tun    sys.NetInterface
		ctx    context.Context
		cancel context.CancelFunc
	}

	// TODO: add some map to store HTTP/3 tunnel connections

	dnsmx      sync.RWMutex
	dnsRecords map[string]string

	proxiedConns struct {
		mx    sync.Mutex
		conns map[string]*connectip.Conn // key: IP string
	}

	portMapping map[string]*struct {
		destPort string
		destIP   string
		srcIP    string
	}

	locks        map[string]*sync.Mutex
	packetQueues map[string]chan []byte
	ifaceFactory NetInterfaceFactory
}

func newSubnet(ctx context.Context, l *Libp2p, factory NetInterfaceFactory) *subnet {
	return &subnet{
		ctx:     ctx,
		network: l,
		info: struct {
			id     string
			rtable SubnetRoutingTable
			cidr   *net.IPNet
		}{
			rtable: NewRoutingTable(),
			cidr:   &net.IPNet{IP: net.IPv4zero, Mask: net.CIDRMask(0, 32)}, // TODO: replace
		},
		ifaces: make(map[string]struct {
			tun    sys.NetInterface
			ctx    context.Context
			cancel context.CancelFunc
		}),
		proxiedConns: struct {
			mx    sync.Mutex
			conns map[string]*connectip.Conn
		}{
			conns: map[string]*connectip.Conn{},
		},
		dnsRecords: map[string]string{},
		portMapping: map[string]*struct {
			destPort string
			destIP   string
			srcIP    string
		}{},
		locks:        make(map[string]*sync.Mutex),
		packetQueues: make(map[string]chan []byte),
		ifaceFactory: factory,
	}
}

func (l *Libp2p) CreateSubnet(ctx context.Context, subnetID string, cidr string, routingTable map[string]string) error {
	l.subnetsmx.Lock()
	defer l.subnetsmx.Unlock()

	if _, ok := l.subnets[subnetID]; ok {
		return fmt.Errorf("subnet with ID %s already exists", subnetID)
	}

	s := newSubnet(ctx, l, l.NetIfaceFactory)
	s.info.id = subnetID

	_, CIDR, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}

	s.mx.Lock()
	s.info.cidr = CIDR

	for ip, peerctx := range routingTable {
		peerID, err := peer.Decode(peerctx)
		if err != nil {
			s.mx.Unlock()
			return fmt.Errorf("failed to decode peer ID %s: %w", peerctx, err)
		}
		s.info.rtable.Add(peerID, ip)
	}
	s.mx.Unlock()
	if atomic.CompareAndSwapInt32(&l.isHTTPServerRegistered, 0, 1) {
		if err := l.startIPProxy(); err != nil {
			return err
		}
	}
	l.subnets[subnetID] = s

	return nil
}

func (l *Libp2p) DestroySubnet(subnetID string) error {
	l.subnetsmx.Lock()
	defer l.subnetsmx.Unlock()

	s, ok := l.subnets[subnetID]
	if !ok {
		return fmt.Errorf("subnet with ID %s does not exist", subnetID)
	}

	for ip := range s.ifaces {
		s.ifaces[ip].cancel()
		s.ifaces[ip].tun.Close()
		_ = s.ifaces[ip].tun.Down()
		_ = s.ifaces[ip].tun.Delete()
	}

	s.mx.Lock()
	s.ifaces = make(map[string]struct {
		tun    sys.NetInterface
		ctx    context.Context
		cancel context.CancelFunc
	})
	s.mx.Unlock()

	// Clean up proxied connections
	s.proxiedConns.mx.Lock()
	for ip, conn := range s.proxiedConns.conns {
		log.Debugf("closing proxied connection for %s during subnet destruction", ip)
		conn.Close()
	}
	s.proxiedConns.conns = make(map[string]*connectip.Conn)
	s.proxiedConns.mx.Unlock()

	s.dnsmx.Lock()
	s.dnsRecords = make(map[string]string)
	s.dnsmx.Unlock()

	s.info.rtable.Clear()

	for sourcePort, mapping := range s.portMapping {
		err := l.UnmapPort(subnetID, "tcp", mapping.srcIP, sourcePort, mapping.destIP, mapping.destPort)
		if err != nil {
			log.Errorf("failed to unmap port %s: %v", sourcePort, err)
		}
	}

	if len(l.subnets) == 1 {
		if atomic.CompareAndSwapInt32(&l.isHTTPServerRegistered, 1, 0) {
			l.stopIPProxy()
		}
	}

	delete(l.subnets, subnetID)
	log.Debugf("subnet %s destroyed", subnetID)
	return nil
}

// TODO: This method isn't doing what its name implies.
// This is basically Creating the subnet, not adding adding a peer to the subnet.
// Move all this business logic to the CreateSubnet.
func (l *Libp2p) AddSubnetPeer(subnetID, peerID, ip string) error {
	l.subnetsmx.Lock()
	defer l.subnetsmx.Unlock()

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

	log.Debugf("finding proper iface name for TUN interface (taken_names=%s)", takenNames)
	name, err := generateUniqueName(takenNames)
	if err != nil {
		return fmt.Errorf("failed to generate unique name for TUN interface: %w", err)
	}

	log.Debugf("Creating TUN interface with name: %s", name)
	address := fmt.Sprintf("%s/24", ipAddr.String())

	iface, err := s.ifaceFactory(name)
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
		tun    sys.NetInterface
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

func (l *Libp2p) RemoveSubnetPeers(subnetID string, partialRoutinTable map[string]string) error {
	for ip, peerID := range partialRoutinTable {
		err := l.removeSubnetPeer(subnetID, peerID, ip)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Libp2p) removeSubnetPeer(subnetID, peerID, ip string) error {
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

	found := false
	for _, i := range ips {
		if i == ip {
			found = true
			break
		}
	}
	if !found {
		return nil
	}

	s.mx.Lock()
	iface, ok := s.ifaces[ip]
	if ok {
		iface.cancel()
		if err := iface.tun.Down(); err != nil {
			log.Errorf("failed to bring down tun device: %v (subnet=%s, ip=%s)", err, s.info.id, ip)
		}
		if err := iface.tun.Delete(); err != nil {
			log.Errorf("failed to delete tun device: %v (subnet=%s, ip=%s)", err, s.info.id, ip)
		}
		delete(s.ifaces, ip)
	}
	s.mx.Unlock()

	s.info.rtable.Remove(peerIDObj, ip)
	return nil
}

func (l *Libp2p) AcceptSubnetPeers(subnetID string, partialRoutingTable map[string]string) error {
	for ip, peerID := range partialRoutingTable {
		err := l.acceptSubnetPeer(subnetID, peerID, ip)
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Libp2p) acceptSubnetPeer(subnetID, peerID, ip string) error {
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

	// TODO: remove this or check if the IP exists.
	// if _, ok := s.info.rtable.Get(peerIDObj); ok {
	// 	return nil
	// }

	s.info.rtable.Add(peerIDObj, ip)
	return nil
}

func (l *Libp2p) AddSubnetDNSRecords(subnetID string, records map[string]string) error {
	l.subnetsmx.Lock()
	defer l.subnetsmx.Unlock()

	s, ok := l.subnets[subnetID]
	if !ok {
		return fmt.Errorf("subnet with ID %s does not exist", subnetID)
	}

	s.dnsmx.Lock()
	for name, ip := range records {
		s.dnsRecords[name] = ip
	}
	s.dnsmx.Unlock()

	return nil
}

func (l *Libp2p) RemoveSubnetDNSRecord(subnetID, name string) error {
	l.subnetsmx.Lock()
	defer l.subnetsmx.Unlock()

	s, ok := l.subnets[subnetID]
	if !ok {
		return fmt.Errorf("subnet with ID %s does not exist", subnetID)
	}

	s.dnsmx.Lock()
	delete(s.dnsRecords, name)
	s.dnsmx.Unlock()

	return nil
}

func (l *Libp2p) startIPProxy() error {
	p := connectip.Proxy{}
	hostIP, err := l.HostPublicIP()
	if err != nil {
		return fmt.Errorf("failed to get host public IP: %w", err)
	}

	if l.rawqtr.listener == nil {
		<-l.rawqtr.listenerReady
	}

	actualPort := l.rawqtr.listener.Addr().(*net.UDPAddr).Port
	template := uritemplate.MustNew(fmt.Sprintf("http://%s:%d/vpn", hostIP, actualPort))

	mux := http.NewServeMux()
	mux.HandleFunc("/vpn", func(w http.ResponseWriter, r *http.Request) {
		// get subnet id from the query
		subnetID := r.URL.Query().Get("subnetID")
		if subnetID == "" {
			log.Debug("received bad http proxy request, no subnetID was provided")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// get src ip from query params
		srcIP := r.URL.Query().Get("srcIP")
		if srcIP == "" || !IsIPv4(srcIP) {
			log.Debug("received bad http proxy request, no srcIP was provided")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		log.Debugf("received http proxy request for subnet %s from %s", subnetID, srcIP)

		l.subnetsmx.Lock()
		// retrieve subnet
		subnet, ok := l.subnets[subnetID]
		l.subnetsmx.Unlock()
		if !ok {
			log.Debugf("subnet with ID %s does not exist", subnetID)
			w.WriteHeader(http.StatusNotFound)
			return
		}

		addr := netip.MustParseAddr(srcIP)
		route := netip.MustParsePrefix(subnet.info.cidr.String())
		req, err := connectip.ParseRequest(r, template)
		if err != nil {
			log.Errorf("failed to parse request: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		conn, err := p.Proxy(w, req)
		if err != nil {
			log.Errorf("failed to proxy connection: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// ATOMIC check and store for incoming connections
		subnet.proxiedConns.mx.Lock()
		if old, ok := subnet.proxiedConns.conns[srcIP]; ok {
			old.Close()
		}
		subnet.proxiedConns.conns[srcIP] = conn
		subnet.proxiedConns.mx.Unlock()

		// Double-check: is this still the connection in the map?
		subnet.proxiedConns.mx.Lock()
		if subnet.proxiedConns.conns[srcIP] != conn {
			subnet.proxiedConns.mx.Unlock()
			log.Debugf("connection from %s lost race, closing", srcIP)
			log.Errorf("closing connection for %s", srcIP)
			conn.Close()
			return
		}
		subnet.proxiedConns.mx.Unlock()

		log.Debugf("connection from %s stored in subnet", srcIP)

		if err := l.handleIPProxyConn(subnet, conn, addr, route, 0); err != nil {
			log.Error("failed to handle connection: %v", err)
		}
	})

	ctx, cancel := context.WithCancel(context.Background())
	s := &http3.Server{
		Handler:         mux,
		EnableDatagrams: true,
	}
	go func() {
		if l.rawqtr.listener == nil {
			<-l.rawqtr.listenerReady
		}

		for {
			select {
			case <-ctx.Done():
				log.Debug("ip proxy context done, shutting down")
				return
			case <-l.ctx.Done():
				log.Debug("libp2p context done, shutting down")
				return
			case rawQUICConn := <-l.rawqtr.listener.acceptQueue:
				l.ipproxyConnsMx.Lock()
				l.ipproxyConns[rawQUICConn.RemoteAddr().String()] = rawQUICConn
				l.ipproxyConnsMx.Unlock()

				go func() {
					log.Debug("serve http3 connection on raw quic connection")
					err = s.ServeQUICConn(rawQUICConn)
					if err != nil {
						log.Errorf("failed to serve http3 connection on raw quic connection: %v", err)
						err := rawQUICConn.CloseWithError(quic.ApplicationErrorCode(quic.NoError), "server is down")
						if err != nil {
							log.Errorf("failed to close raw quic connection: %v", err)
						}
					}
				}()
			}
		}
	}()

	l.ipproxyCtx = ctx
	l.ipproxyCtxCancel = cancel
	l.ipproxy = s
	l.ipproxyConnsMx.Lock()
	l.ipproxyConns = make(map[string]*quic.Conn)
	l.ipproxyConnsMx.Unlock()

	log.Info("started ip proxy for all subnets")

	return nil
}

func (l *Libp2p) handleIPProxyConn(
	snet *subnet,
	conn *connectip.Conn,
	addr netip.Addr,
	route netip.Prefix,
	ipProtocol uint8,
) error {
	log.Debugf(
		"handling ip proxy conn (subnet=%s, addr=%s, route=%s, ipProtocol=%d)",
		snet.info.id, addr.String(), route.String(), ipProtocol,
	)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := conn.AssignAddresses(ctx, []netip.Prefix{netip.PrefixFrom(addr, addr.BitLen())}); err != nil {
		return fmt.Errorf("failed to assign addresses: %w", err)
	}
	if err := conn.AdvertiseRoute(ctx, []connectip.IPRoute{
		{StartIP: route.Addr(), EndIP: LastIP(route), IPProtocol: ipProtocol},
	}); err != nil {
		return fmt.Errorf("failed to advertise route: %w", err)
	}

	errChan := make(chan error, 2)
	go func() {
		for {
			select {
			case <-snet.ctx.Done():
				snet.cleanupConn(addr.String(), conn)
				return
			default:
				b := make([]byte, 2000)
				n, err := conn.ReadPacket(b)
				if err != nil {
					errChan <- fmt.Errorf("failed to read from connection: %w", err)
					snet.cleanupConn(addr.String(), conn)
					return
				}

				log.Debugf("read %d bytes from connection", n)

				// 1. retrieve dest ip
				destIP := net.IPv4(b[16], b[17], b[18], b[19]).String()

				_, ok := snet.info.rtable.GetByIP(destIP)
				if !ok {
					log.Debugf("unrecognized destination ip %s, no peerID found for ip, not a subnet member", destIP)
					continue
				}

				// 2. fetch the respective tun dev
				snet.mx.Lock()
				if iface, ok := snet.ifaces[destIP]; ok {
					log.Debugf("writing packet to tun device %s", iface.tun.Name())
					// 3. write to tun dev
					if _, err := iface.tun.Write(b[:n]); err != nil {
						log.Errorf("failed to write to tun device: %v (subnet=%s, destIP=%s)", err, snet.info.id, destIP)
					}
				} else {
					log.Debugf("unrecognized destination ip %s, no tun device found for ip", destIP)
				}
				snet.mx.Unlock()
			}
		}
	}()

	go func() {
		select {
		case err := <-errChan:
			log.Errorf("failed to handle connection: %v", err)
		case <-snet.ctx.Done():
			snet.cleanupConn(addr.String(), conn)
		}
	}()

	return nil
}

func (l *Libp2p) stopIPProxy() {
	log.Infof("stopping ip proxy")

	l.ipproxyCtxCancel()
	// ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	// defer cancel()
	// l.ipproxy.Shutdown(ctx)
	l.ipproxy.Close()
	for _, conn := range l.ipproxyConns {
		err := conn.CloseWithError(quic.ApplicationErrorCode(quic.NoError), "shutting down")
		if err != nil {
			log.Errorf("failed to close ipproxy connection: %v", err)
		}
	}
	l.ipproxyConns = make(map[string]*quic.Conn)
	l.ipproxy = nil
}

func (s *subnet) handleDNSQueries(iface sys.NetInterface, packet []byte, packetlen int) error {
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

	_, _ = iface.Write(buffer.Bytes())
	return nil
}

func (s *subnet) Route(iface sys.NetInterface, srcIP, destIP string, packet []byte, plen int) {
	log.Debugf("routing packet (subnet=%s, dstIP=%s, packet_len=%d)", s.info.id, destIP, plen)

	s.mx.Lock()
	peerID, ok := s.info.rtable.GetByIP(destIP)
	if !ok {
		log.Debugf("unrecognized destination ip on subnetID: %s, dstIP: %s, not a subnet member", s.info.id, destIP)
		s.mx.Unlock()
		return
	}

	_, ok = s.info.rtable.GetByIP(srcIP)
	if !ok {
		log.Debugf("unrecognized source ip on subnetID: %s, srcIP: %s, not a subnet member", s.info.id, srcIP)
		s.mx.Unlock()
		return
	}

	if _, ok := s.ifaces[destIP]; ok {
		log.Debugf("found destination ip in tuns table (local) on subnetID: %s, dstIP: %s, writing packet of length %d", s.info.id, destIP, plen)
		// if so, write to the tun
		n, err := s.ifaces[destIP].tun.Write(packet[:plen])
		if err != nil {
			log.Errorf("failed to write to tun device: %v (subnet=%s, destIP=%s, bytes_written=%d)", err, s.info.id, destIP, n)
		} else {
			log.Debugf("successfully wrote %d bytes to tun device (subnet=%s, destIP=%s)", n, s.info.id, destIP)
		}
		s.mx.Unlock()
		return
	}
	s.mx.Unlock()

	log.Debugf(
		"found destination ip in routing table (subnet=%s, destIP=%s, peerID=%s)",
		s.info.id, destIP, peerID.String(),
	)

	if err := s.proxyPacket(s.ctx, iface, peerID, srcIP, destIP, packet, plen); err != nil {
		log.Errorf("failed to proxy packet: %v", err)
	}
}

func (s *subnet) proxyPacket(
	ctx context.Context,
	iface sys.NetInterface,
	dst peer.ID,
	srcIP,
	destIP string,
	packet []byte,
	plen int,
) error {
	s.proxiedConns.mx.Lock()
	conn, ok := s.proxiedConns.conns[destIP]
	if !ok {
		log.Debugf("no connection for %s, establishing one", destIP)
		// No connection: establish one synchronously
		newConn, quicConn, err := s.dialIPProxy(ctx, dst, srcIP)
		if err != nil {
			s.proxiedConns.mx.Unlock()
			return fmt.Errorf("failed to establish connection to %s: %w", destIP, err)
		}

		if old, ok := s.proxiedConns.conns[destIP]; ok {
			if old != newConn { // Only close if it's a different connection
				log.Debugf("closing old connection for %s", destIP)
				old.Close()
			}
		}
		s.proxiedConns.conns[destIP] = newConn
		s.proxiedConns.mx.Unlock()
		conn = newConn

		// Start a goroutine to read from the new connection and write to the TUN device
		go func(ipconn *connectip.Conn, destIP, srcIP string, quicConn *quic.Conn) {
			for {
				select {
				case <-ctx.Done():
					log.Debugf("context done, abandoning read loop... (subnet=%s)", s.info.id)
					return
				case <-s.ctx.Done():
					log.Debugf("context done, abandoning read loop... (subnet=%s)", s.info.id)
					return
				case <-quicConn.Context().Done():
					log.Debugf("quic connection for %s closed, closing connection", destIP)
					s.cleanupConn(destIP, ipconn)
					return
				default:
					b := make([]byte, 2000)
					n, err := ipconn.ReadPacket(b)
					if err != nil {
						log.Errorf("failed to read from outgoing connection: %v (subnet=%s, dst=%s)", err, s.info.id, dst.String())
						s.cleanupConn(destIP, ipconn)
						return
					}
					// Write to the appropriate TUN device
					s.mx.Lock()
					iface, ok := s.ifaces[srcIP]
					s.mx.Unlock()
					if ok {
						if _, err := iface.tun.Write(b[:n]); err != nil {
							log.Errorf("failed to write to TUN from outgoing connection: %v (subnet=%s, dst=%s)", err, s.info.id, dst.String())
						}
					} else {
						log.Debugf("no TUN device for destIP %s (subnet=%s)", destIP, s.info.id)
					}
				}
			}
		}(newConn, destIP, srcIP, quicConn)
	} else {
		log.Debugf("found connection for %s, writing packet", destIP)
		s.proxiedConns.mx.Unlock()
	}
	// Now write to the connection
	icmp, err := conn.WritePacket(packet[:plen])
	if err != nil {
		log.Errorf("failed to write packet to connection: %s (subnet=%s, dst=%s)", err, s.info.id, dst.String())
		s.cleanupConn(destIP, conn)
		return fmt.Errorf("failed to write packet to connection: %w", err)
	}
	if len(icmp) > 0 {
		_, err := iface.Write(icmp)
		if err != nil {
			log.Errorf("failed to write ICMP packet to tun device: %s (subnet=%s, dst=%s)", err, s.info.id, dst.String())
			return fmt.Errorf("failed to write ICMP packet to tun device: %w", err)
		}
	}
	return nil
}

func (s *subnet) readPackets(ctx context.Context, iface sys.NetInterface) {
	for {
		select {
		case <-ctx.Done():
			log.Debugf("context done, abandoning read loop... (subnet=%s)", s.info.id)
			return
		case <-s.ctx.Done():
			log.Debugf("context done, abandoning read loop... (subnet=%s)", s.info.id)
			return
		default:
			{
				packet := make([]byte, MaxPacketSize)
				// Read in a packet from the tun device.
				plen, err := iface.Read(packet)
				if errors.Is(err, fs.ErrClosed) {
					time.Sleep(1 * time.Second)
					log.Debugf("tun device closed, abandoning read loop... (err=%s, subnet=%s)", err, s.info.id)
					return
				} else if err != nil {
					log.Debugf("(error): failed to read packet from tun device: %s (subnet=%s)", err, s.info.id)
					continue
				}

				if plen == 0 {
					log.Debugf("(error): received zero-length packet from tun device (subnet=%s, iface=%s)", s.info.id, iface.Name())
					continue
				}

				if plen > MaxPacketSize {
					log.Warnf("received packet with length %d, truncating to %d", plen, MaxPacketSize)
					plen = MaxPacketSize
				}

				srcPort, destPort, srcIP, destIP, err := s.parseIPPacket(packet)
				if err != nil {
					log.Warnf("(error): failed to parse IP packet: %s", err)
					continue
				}

				log.Debugf(
					"read packet from tun device (tun=%s, subnet=%s, destIP=%s, destPort=%s, srcIP=%s, srcPort=%s)",
					iface.Name(),
					s.info.id,
					destIP,
					destPort,
					srcIP,
					srcPort,
				)

				// Fix DNS filtering logic - only handle DNS queries, route everything else
				if destPort == 53 {
					log.Debugf(
						"handling DNS query (tun=%s, subnet=%s, destIP=%s, destPort=%s, srcIP=%s, srcPort=%s)",
						iface.Name(),
						s.info.id,
						destIP,
						destPort,
						srcIP,
						srcPort,
					)

					if err := s.handleDNSQueries(iface, packet, plen); err != nil {
						log.Errorf("failed to handle DNS query: %s", err)
					}
				} else {
					s.Route(iface, srcIP, destIP, packet, plen)
				}
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

	udpLayer := packet.Layer(layers.LayerTypeUDP)
	if udpLayer != nil {
		udp, _ := udpLayer.(*layers.UDP)
		srcPort = int(udp.SrcPort)
		destPort = int(udp.DstPort)
	}

	// Add TCP parsing
	tcpLayer := packet.Layer(layers.LayerTypeTCP)
	if tcpLayer != nil {
		tcp, _ := tcpLayer.(*layers.TCP)
		srcPort = int(tcp.SrcPort)
		destPort = int(tcp.DstPort)
	}

	return srcPort, destPort, srcIP, destIP, err
}

func (s *subnet) dialIPProxy(
	ctx context.Context,
	target peer.ID,
	srcIP string,
) (*connectip.Conn, *quic.Conn, error) {
	conn, proxyAddr, err := s.network.RawQUICConnect(target, s.info.id)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial raw QUIC connection: %w", err)
	}

	tr := &http3.Transport{EnableDatagrams: true}
	hconn := tr.NewClientConn(conn)
	template := uritemplate.MustNew(fmt.Sprintf(
		"https://%s:%d/vpn?subnetID=%s&srcIP=%s",
		proxyAddr.Addr().Unmap().String(),
		proxyAddr.Port(),
		s.info.id,
		srcIP,
	))

	ipconn, rsp, err := connectip.Dial(ctx, hconn, template)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to dial connect-ip connection: %w", err)
	}
	if rsp.StatusCode != http.StatusOK {
		log.Errorf("unexpected status code: %d (err=%s, body=%s)", rsp.StatusCode, err, rsp.Body)
		return nil, nil, fmt.Errorf("unexpected status code: %d", rsp.StatusCode)
	}

	log.Debugf("connected to IP Proxy for target %s on %s", target, proxyAddr)

	return ipconn, conn, nil
}

func (s *subnet) PeersAddresses() map[string]bool {
	addresses := make(map[string]bool)
	s.mx.Lock()
	rtable := s.info.rtable
	s.mx.Unlock()

	for peerID := range rtable.All() {
		addrs := s.network.Host.Peerstore().Addrs(peerID)
		if len(addrs) == 0 {
			pinfo, err := s.network.resolvePeerAddress(context.TODO(), peerID)
			if err != nil {
				log.Errorf("failed to resolve peer address: %s", err)
				continue
			}
			addrs = pinfo.Addrs
		}
		for _, addr := range addrs {
			parts := strings.Split(addr.String(), "/")
			ip := parts[2]
			port := parts[4]
			addresses[fmt.Sprintf("%s:%s", ip, port)] = true
		}
	}
	return addresses
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

func LastIP(prefix netip.Prefix) netip.Addr {
	addr := prefix.Addr()
	bytes := addr.AsSlice()

	hostBits := len(bytes)*8 - prefix.Bits()
	for i := len(bytes) - 1; i >= 0; i-- {
		setBits := math.Min(8, float64(hostBits))
		if setBits <= 0 {
			break
		}
		bytes[i] |= byte(0xff >> (8 - int(setBits)))
		hostBits -= 8
	}

	if addr.Is4() {
		return netip.AddrFrom4([4]byte(bytes[:4]))
	}
	return netip.AddrFrom16([16]byte(bytes))
}

func IsIPv4(ip string) bool {
	return net.ParseIP(ip).To4() != nil
}

func (s *subnet) cleanupConn(ip string, conn *connectip.Conn) {
	defer s.proxiedConns.mx.Unlock()
	s.proxiedConns.mx.Lock()
	current, ok := s.proxiedConns.conns[ip]
	if ok && current == conn {
		log.Debugf("cleanupConn: closing and removing connection for %s", ip)
		delete(s.proxiedConns.conns, ip)
		conn.Close()
	}
}

func NetTunFactory(name string) (sys.NetInterface, error) {
	return sys.NewTunTapInterface(name, sys.NetTunMode, false)
}
