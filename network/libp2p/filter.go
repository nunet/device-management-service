package libp2p

import (
	"github.com/libp2p/go-libp2p/core/connmgr"
	"github.com/libp2p/go-libp2p/core/control"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	mafilt "github.com/whyrusleeping/multiaddr-filter"
)

var defaultServerFilters = []string{
	"/ip4/10.0.0.0/ipcidr/8",
	"/ip4/100.64.0.0/ipcidr/10",
	"/ip4/169.254.0.0/ipcidr/16",
	"/ip4/172.16.0.0/ipcidr/12",
	"/ip4/192.0.0.0/ipcidr/24",
	"/ip4/192.0.2.0/ipcidr/24",
	"/ip4/192.168.0.0/ipcidr/16",
	"/ip4/198.18.0.0/ipcidr/15",
	"/ip4/198.51.100.0/ipcidr/24",
	"/ip4/203.0.113.0/ipcidr/24",
	"/ip4/240.0.0.0/ipcidr/4",
	"/ip6/100::/ipcidr/64",
	"/ip6/2001:2::/ipcidr/48",
	"/ip6/2001:db8::/ipcidr/32",
	"/ip6/fc00::/ipcidr/7",
	"/ip6/fe80::/ipcidr/10",
}

// PeerFilter is an interface for filtering peers
// satisfaction of filter criteria allows the peer to pass
type PeerFilter interface {
	satisfies(p peer.AddrInfo) bool
}

// NoAddrIDFilter filters out peers with no listening addresses
// and a peer with a specific ID
type NoAddrIDFilter struct {
	ID peer.ID
}

func (f NoAddrIDFilter) satisfies(p peer.AddrInfo) bool {
	return len(p.Addrs) > 0 && p.ID != f.ID
}

func PeerPassFilter(peers []peer.AddrInfo, pf PeerFilter) []peer.AddrInfo {
	var filtered []peer.AddrInfo
	for _, p := range peers {
		if pf.satisfies(p) {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

type filtersConnectionGater multiaddr.Filters

var _ connmgr.ConnectionGater = (*filtersConnectionGater)(nil)

func (f *filtersConnectionGater) InterceptAddrDial(_ peer.ID, addr multiaddr.Multiaddr) (allow bool) {
	return !(*multiaddr.Filters)(f).AddrBlocked(addr)
}

func (f *filtersConnectionGater) InterceptPeerDial(_ peer.ID) (allow bool) {
	return true
}

func (f *filtersConnectionGater) InterceptAccept(connAddr network.ConnMultiaddrs) (allow bool) {
	return !(*multiaddr.Filters)(f).AddrBlocked(connAddr.RemoteMultiaddr())
}

func (f *filtersConnectionGater) InterceptSecured(_ network.Direction, _ peer.ID, connAddr network.ConnMultiaddrs) (allow bool) {
	return !(*multiaddr.Filters)(f).AddrBlocked(connAddr.RemoteMultiaddr())
}

func (f *filtersConnectionGater) InterceptUpgraded(_ network.Conn) (allow bool, reason control.DisconnectReason) {
	return true, 0
}

func makeAddrsFactory(announce []string, appendAnnouce []string, noAnnounce []string) func([]multiaddr.Multiaddr) []multiaddr.Multiaddr {
	var err error                     // To assign to the slice in the for loop
	existing := make(map[string]bool) // To avoid duplicates

	annAddrs := make([]multiaddr.Multiaddr, len(announce))
	for i, addr := range announce {
		annAddrs[i], err = multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil
		}
		existing[addr] = true
	}

	appendAnnAddrs := make([]multiaddr.Multiaddr, 0)
	for _, addr := range appendAnnouce {
		if existing[addr] {
			// skip AppendAnnounce that is on the Announce list already
			continue
		}
		appendAddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil
		}
		appendAnnAddrs = append(appendAnnAddrs, appendAddr)
	}

	filters := multiaddr.NewFilters()
	noAnnAddrs := map[string]bool{}
	for _, addr := range noAnnounce {
		f, err := mafilt.NewMask(addr)
		if err == nil {
			filters.AddFilter(*f, multiaddr.ActionDeny)
			continue
		}
		maddr, err := multiaddr.NewMultiaddr(addr)
		if err != nil {
			return nil
		}
		noAnnAddrs[string(maddr.Bytes())] = true
	}

	return func(allAddrs []multiaddr.Multiaddr) []multiaddr.Multiaddr {
		var addrs []multiaddr.Multiaddr
		if len(annAddrs) > 0 {
			addrs = annAddrs
		} else {
			addrs = allAddrs
		}
		addrs = append(addrs, appendAnnAddrs...)

		var out []multiaddr.Multiaddr
		for _, maddr := range addrs {
			// check for exact matches
			ok := noAnnAddrs[string(maddr.Bytes())]
			// check for /ipcidr matches
			if !ok && !filters.AddrBlocked(maddr) {
				out = append(out, maddr)
			}
		}
		return out
	}
}
