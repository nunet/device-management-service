package models

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
)

const (
	Libp2pNetwork NetworkType = "libp2p"
	NATSNetwork   NetworkType = "nats"
)

type MessageType string

type MessageEnvelope struct {
	Type MessageType
	Data []byte
}

type NetworkType string

type NetworkConfig struct {
	Type NetworkType

	// libp2p
	Libp2pConfig

	// nats
	NATSUrl string
}

// Libp2pConfig holds the libp2p configuration
type Libp2pConfig struct {
	DHTPrefix               string
	PrivateKey              crypto.PrivKey
	BootstrapPeers          []multiaddr.Multiaddr
	Rendezvous              string
	Server                  bool
	Scheduler               *bt.Scheduler
	CustomNamespace         string
	ListenAddress           []string
	PeerCountDiscoveryLimit int
	PrivateNetwork          PrivateNetworkConfig
	GracePeriodMs           int
	GossipMaxMessageSize    int
}

type PrivateNetworkConfig struct {
	// WithSwarmKey if true, DMS will try to fetch the key from
	// `<config_path>/swarm.key`.
	WithSwarmKey bool

	// ACL defines the access control list for the private network.
	ACL []multiaddr.Multiaddr
}
