// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"time"
)

const (
	NetP2P = "p2p"
)

// NetworkSpec defines the network specification
type NetworkSpec interface{}

// NetConfig is a stub. Please expand it or completely change it based on requirements.
type NetConfig struct {
	NetworkSpec SpecConfig `json:"network_spec"` // Network specification
}

func (nc *NetConfig) GetNetworkConfig() *SpecConfig {
	return &nc.NetworkSpec
}

// NetworkStats should contain all network info the user is interested in.
// for now there's only peerID and listening address but reachability, local and remote addr etc...
// can be added when necessary.
type NetworkStats struct {
	ID         string `json:"id"`
	ListenAddr string `json:"listen_addr"`
}

// MessageInfo is a stub. Please expand it or completely change it based on requirements.
type MessageInfo struct {
	Info string `json:"info"` // Message information
}

type PingResult struct {
	RTT     time.Duration
	Success bool
	Error   error
}
