package libp2p

import (
	"context"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p/core/discovery"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
)

// DiscoverDialPeers discovers peers using randevouz point
func (l *Libp2p) DiscoverDialPeers(ctx context.Context) error {
	foundPeers, err := l.findPeersFromRendezvousDiscovery(ctx)
	if err != nil {
		return err
	}

	if len(foundPeers) > 0 {
		l.discoveredPeers = foundPeers
	}

	// filter out peers with no listening addresses and self host
	filterSpec := NoAddrIDFilter{ID: l.Host.ID()}
	l.discoveredPeers = PeerPassFilter(l.discoveredPeers, filterSpec)

	l.dialPeers(ctx)

	return nil
}

// advertiseForRendezvousDiscovery is used to advertise node using the dht by giving it the randevouz point.
func (l *Libp2p) advertiseForRendezvousDiscovery(context context.Context) error {
	_, err := l.discovery.Advertise(context, l.config.Rendezvous)
	return err
}

// findPeersFromRendezvousDiscovery uses the randevouz point to discover other peers.
func (l *Libp2p) findPeersFromRendezvousDiscovery(ctx context.Context) ([]peer.AddrInfo, error) {
	peers, err := dutil.FindPeers(
		ctx,
		l.discovery,
		l.config.Rendezvous,
		discovery.Limit(l.config.PeerCountDiscoveryLimit),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to discover peers: %w", err)
	}
	return peers, nil
}

func (l *Libp2p) dialPeers(ctx context.Context) {
	for _, p := range l.discoveredPeers {
		if p.ID == l.Host.ID() {
			continue
		}
		if l.Host.Network().Connectedness(p.ID) != network.Connected {
			_, err := l.Host.Network().DialPeer(ctx, p.ID)
			if err != nil {
				if _, debugMode := os.LookupEnv("NUNET_DEBUG_VERBOSE"); debugMode {
					log.Debugf("couldn't establish connection with: %s - error: %v", p.ID.String(), err)
				}
				continue
			}
			if _, debugMode := os.LookupEnv("NUNET_DEBUG_VERBOSE"); debugMode {
				log.Debugf("connected with: %s", p.ID.String())
			}
		}
	}
}
