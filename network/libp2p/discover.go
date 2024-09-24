package libp2p

import (
	"context"
	"fmt"
	"math/rand"
	"time"

	"github.com/libp2p/go-libp2p/core/discovery"
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
	maxPeers := 16
	peersToConnect := l.discoveredPeers

	if len(peersToConnect) > maxPeers {
		//nolint:gosec
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(peersToConnect), func(i, j int) {
			peersToConnect[i], peersToConnect[j] = peersToConnect[j],
				peersToConnect[i]
		})

		// Take only the first maxPeers
		peersToConnect = peersToConnect[:maxPeers]
	}

	for _, p := range peersToConnect {
		if p.ID == l.Host.ID() {
			continue
		}

		if !l.PeerConnected(p.ID) {
			go func(p peer.AddrInfo) {
				dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
				defer cancel()

				if err := l.Host.Connect(dialCtx, p); err != nil {
					log.Debugf("couldn't establish connection with: %s - error: %v", p.ID, err)
					return
				}

				log.Debugf("connected with: %s", p.ID)
			}(p)
		}
	}
}
