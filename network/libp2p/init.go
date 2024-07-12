package libp2p

import (
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
)

const (
	// Custom namespace for DHT protocol with version number
	customNamespace = "/nunet-dht-1/"
)

// TODO: pass the logger to the constructor and remove from here
var (
	zlog    otelzap.Logger
	newPeer = make(chan peer.AddrInfo)
)

func init() {
	zlog = logger.OtelZapLogger("network.libp2p")
}
