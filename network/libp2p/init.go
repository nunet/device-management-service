package libp2p

import (
	dht "github.com/libp2p/go-libp2p-kad-dht"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
)

const (
	// Custom namespace for DHT protocol with version number
	customNamespace = "/nunet-dht-1/"
)

var (
	zlog                 otelzap.Logger
	kadPrefix            = dht.ProtocolPrefix("/nunet")
	gettingDHTUpdate     = false
	doneGettingDHTUpdate = make(chan bool) // XXX dirty hack to wait for DHT update to finish - should be removed

	// bootstrap peers provided by NuNet
	NuNetBootstrapPeers []multiaddr.Multiaddr

	// new peer channel for relay discovery
	// XXX doesn't look good - should be reconsidered
	newPeer = make(chan peer.AddrInfo)
)

func init() {
	zlog = logger.OtelZapLogger("network.libp2p")

	for _, s := range config.GetConfig().P2P.BootstrapPeers {
		ma, err := multiaddr.NewMultiaddr(s)
		if err != nil {
			panic(err)
		}
		NuNetBootstrapPeers = append(NuNetBootstrapPeers, ma)
	}

}
