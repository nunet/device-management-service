// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobs

import (
	"encoding/json"

	"gitlab.com/nunet/device-management-service/actor"
)

type EnsembleManifest struct {
	ID           string                        // ensemble globally unique id
	Orchestrator actor.Handle                  // orchestrator actor
	Allocations  map[string]AllocationManifest // allocation name -> manifest
	Nodes        map[string]NodeManifest       // node name -> manifest
}

type AllocationManifest struct {
	ID       string       // allocation unique id
	NodeID   string       // allocation node
	Handle   actor.Handle // handle of the allocation control actor
	DNSName  string       // (internal) DNS name of the allocation
	PrivAddr string       // (VPN) private IP address of the allocation peer
	Ports    map[int]int  // port mapping, public -> private
}

type NodeManifest struct {
	ID          string       // node unique id
	Peer        string       // peer where the node is running
	Handle      actor.Handle // handle of the control actor for the node
	PubAddrss   []string     // public IP4/6 address of the node peer
	Location    Location     // location of the peer
	Allocations []string     // allocations in the nod
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
