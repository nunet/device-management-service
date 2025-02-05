// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobtypes

import (
	"encoding/json"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/types"
)

type EnsembleManifest struct {
	ID           string                        `json:"id"`           // ensemble globally unique id
	Orchestrator actor.Handle                  `json:"orchestrator"` // orchestrator actor
	Allocations  map[string]AllocationManifest `json:"allocations"`  // allocation name -> manifest
	Nodes        map[string]NodeManifest       `json:"nodes"`        // node name -> manifest
}

type AllocationManifest struct {
	ID          string                    `json:"id"`              // allocation unique id
	NodeID      string                    `json:"node_id"`         // allocation node
	Handle      actor.Handle              `json:"handle"`          // handle of the allocation control actor
	DNSName     string                    `json:"dns_name"`        // (internal) DNS name of the allocation
	PrivAddr    string                    `json:"priv_addr"`       // (VPN) private IP address of the allocation peer
	Ports       map[int]int               `json:"ports,omitempty"` // port mapping, public -> private
	Healthcheck types.HealthCheckManifest `json:"healthcheck"`     // healthcheck configuration
	Status      AllocationStatus          `json:"status"`          // current status of the allocation
}

type NodeManifest struct {
	ID          string       `json:"id"`             // node unique id
	Peer        string       `json:"peer,omitempty"` // peer where the node is running
	Handle      actor.Handle `json:"handle"`         // handle of the control actor for the node
	PubAddrss   []string     `json:"pub_addrss"`     // public IP4/6 address of the node peer
	Location    Location     `json:"location"`       // location of the peer
	Allocations []string     `json:"allocations"`    // allocations in the nod
}

func (mf *EnsembleManifest) Clone() EnsembleManifest {
	var clone EnsembleManifest

	bytes, err := json.Marshal(mf)
	if err != nil {
		log.Errorf("error marshaling ensemble manifest: %s", err)
		return clone
	}

	if err := json.Unmarshal(bytes, &clone); err != nil {
		log.Errorf("error unmarshaling ensemble manifest: %s", err)
	}

	return clone
}
