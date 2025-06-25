// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/multiformats/go-multiaddr"
	bt "gitlab.com/nunet/device-management-service/internal/background_tasks"
)

const (
	Libp2pNetwork  NetworkType = "libp2p"
	NATSNetwork    NetworkType = "nats"
	VirtualNetwork NetworkType = "memory" // test-only
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
	GracePeriodMs           int
	GossipMaxMessageSize    int
	BootstrapMaxSleep       int // in minutes
	Memory                  int // in MB
	FileDescriptors         int
}
