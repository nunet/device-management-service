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
	"errors"
	"fmt"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/types"
)

var ErrAllocationNotFound = errors.New("allocation not found")

// NodeStatus represents the current status of a node
type NodeStatus string

const (
	NodeStatusActive  NodeStatus = "active"  // Node is active and running allocations
	NodeStatusStandby NodeStatus = "standby" // Node is a standby waiting to be activated
	// NodeStatusFailed  NodeStatus = "failed"  // Node has failed
)

type RedundancyRole string

var (
	RolePrimary RedundancyRole = "primary"
	RoleStandby RedundancyRole = "standby"
)

type EnsembleManifest struct {
	ID           string                        `json:"id"`                 // ensemble globally unique id
	Metadata     map[string]string             `json:"metadata,omitempty"` // metadata
	Orchestrator actor.Handle                  `json:"orchestrator"`       // orchestrator actor
	Allocations  map[string]AllocationManifest `json:"allocations"`        // allocation name -> manifest
	Nodes        map[string]NodeManifest       `json:"nodes"`              // node name -> manifest
	Subnet       SubnetConfig                  `json:"subnet"`             // subnet configurations
	Contracts    map[string]ContractManifest   `json:"contracts"`          // contract name -> manifest
}

type ContractManifest struct {
	ID   string `json:"id"`  // contract unique id
	DID  string `json:"did"` // DID of the contract
	Host string `json:"host"`
}

type AllocationManifest struct {
	ID              string                    `json:"id"`                         // allocation unique id
	NodeID          string                    `json:"node_id"`                    // allocation node
	Type            AllocationType            `json:"type"`                       // allocation type
	Handle          actor.Handle              `json:"handle"`                     // handle of the allocation control actor
	DNSName         string                    `json:"dns_name"`                   // (internal) DNS name of the allocation
	PrivAddr        string                    `json:"priv_addr"`                  // (VPN) private IP address of the allocation peer
	Ports           map[int]int               `json:"ports,omitempty"`            // port mapping, public -> private
	Healthcheck     types.HealthCheckManifest `json:"healthcheck"`                // healthcheck configuration
	Status          AllocationStatus          `json:"status"`                     // current status of the allocation
	RedundancyGroup string                    `json:"redundancy_group,omitempty"` // base allocation name this belongs to
	IsStandby       bool                      `json:"is_standby"`                 // whether this is a standby allocation
}

type NodeManifest struct {
	ID             string         `json:"id"`                        // node unique id
	Peer           string         `json:"peer,omitempty"`            // peer where the node is running
	Handle         actor.Handle   `json:"handle"`                    // handle of the control actor for the node
	PubAddress     []string       `json:"pub_address"`               // public IP4/6 address of the node peer
	Location       Location       `json:"location"`                  // location of the peer
	Allocations    []string       `json:"allocations"`               // allocations in the node
	RedundancyRole RedundancyRole `json:"redundancy_role,omitempty"` // "primary" or "standby"
	PrimaryNode    string         `json:"primary_node,omitempty"`    // ID of primary node if this is a standby
	StandbyNodes   []string       `json:"standby_nodes,omitempty"`   // IDs of standby nodes if this is a primary
	StandbyIndex   int            `json:"standby_index,omitempty"`   // Index of this standby (1, 2, etc.)
	Status         NodeStatus     `json:"status"`                    // current status of the node (active, standby, failed)
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

// UpdateAllocation applies the provided updater function to the specified allocation.
func (mf *EnsembleManifest) UpdateAllocation(name string, update func(*AllocationManifest)) error {
	if alloc, ok := mf.Allocations[name]; ok {
		update(&alloc)
		mf.Allocations[name] = alloc
	} else {
		return ErrAllocationNotFound
	}
	return nil
}

func (mf *EnsembleManifest) IsTerminatedTask(name string) bool {
	a, ok := mf.Allocations[name]
	if !ok {
		return false
	}

	if a.Type == AllocationTypeTask &&
		a.Status != AllocationRunning {
		return true
	}
	return false
}

func (mf *EnsembleManifest) Allocation(id string) (AllocationManifest, bool) {
	a, ok := mf.Allocations[id]
	return a, ok
}

func (mf *EnsembleManifest) Node(name string) (NodeManifest, bool) {
	n, ok := mf.Nodes[name]
	return n, ok
}

func (mf *EnsembleManifest) JSON() ([]byte, error) {
	data, err := json.MarshalIndent(mf, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("unable to marshal manifest: %w", err)
	}
	return data, nil
}
