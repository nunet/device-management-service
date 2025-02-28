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

	"gitlab.com/nunet/device-management-service/types"
)

type (
	AllocationExecutor string
	AllocationType     string
)

const (
	// Executor types define the runtime environment for allocations
	ExecutorFirecracker AllocationExecutor = "firecracker" // Firecracker VM-based execution
	ExecutorDocker      AllocationExecutor = "docker"      // Docker container-based execution
	ExecutorNull        AllocationExecutor = "null"        // Null executor for testing

	// AllocationType defines the lifecycle behavior of the allocation
	AllocationTypeService AllocationType = "service" // Long-running process that should restart on failure
	AllocationTypeTask    AllocationType = "task"    // One-off job that runs to completion
)

// EnsembleConfig is the versioned structure that contains the ensemble configuration
type EnsembleConfig struct {
	V1 *EnsembleConfigV1 `json:"v1"`
}

// EnsembleConfigV1 is version 1 of the configuration for an ensemble
type EnsembleConfigV1 struct {
	Allocations map[string]AllocationConfig `json:"allocations"`          // (named) allocations in the ensemble
	Nodes       map[string]NodeConfig       `json:"nodes"`                // (named) nodes in the ensemble
	Edges       []EdgeConstraint            `json:"edges,omitempty"`      // network edge constraints
	Supervisor  SupervisorConfig            `json:"supervisor,omitempty"` // supervision structure
	Keys        map[string]string           `json:"keys,omitempty"`       // (named) ssh public keys relevant to the allocation
	Scripts     map[string][]byte           `json:"scripts,omitempty"`    // (named) provisioning scripts
}

// AllocationConfig is the configuration of an allocation
type AllocationConfig struct {
	Executor    AllocationExecutor        `json:"executor"`              // the executor of the allocation
	Type        AllocationType            `json:"type"`                  // the type of allocation (service vs task)
	Resources   types.Resources           `json:"resources"`             // the HW resources required by the allocation
	Execution   types.SpecConfig          `json:"execution"`             // the allocation execution configuration
	DNSName     string                    `json:"dns_name,omitempty"`    // the internal DNS name of the allocation
	Keys        []string                  `json:"keys,omitempty"`        // names of the authorized ssh keys for the allocation
	Provision   []string                  `json:"provision,omitempty"`   // names of provisioning scripts to run (in order)
	HealthCheck types.HealthCheckManifest `json:"healthcheck,omitempty"` // name of the health check script
}

// NodeConfig is the configuration of a distinct DMS node
type NodeConfig struct {
	Allocations []string            `json:"allocations"`        // list of allocation IDs
	Ports       []PortConfig        `json:"ports,omitempty"`    // list of port mappings
	Location    LocationConstraints `json:"location,omitempty"` // location constraints
	Peer        string              `json:"peer,omitempty"`     // peer ID to use for this node
	// TODO contract information
}

// LocationConstraints provides the node location placement constraints
type LocationConstraints struct {
	Accept []Location `json:"accept,omitempty"` // list of accepted locations
	Reject []Location `json:"reject,omitempty"` // list of rejected locations
}

// Location is a geographical location on Planet Earth
type Location struct {
	Continent string `json:"continent,omitempty"` // geographical region
	Country   string `json:"country,omitempty"`   // country code
	City      string `json:"city,omitempty"`      // city name
	ASN       uint   `json:"asn,omitempty"`       // autonomous system number
	ISP       string `json:"isp,omitempty"`       // internet service provider
}

// PortConfig is the configuration for a port mapping a public port to a private port
// in an allocation
type PortConfig struct {
	Public     int    `json:"public"`     // public port number
	Private    int    `json:"private"`    // private port number
	Allocation string `json:"allocation"` // allocation ID
}

// EdgeConstraint is a constraint for a network edge between two nodes
type EdgeConstraint struct {
	S   string `json:"s"`             // source node ID
	T   string `json:"t"`             // target node ID
	RTT uint   `json:"rtt,omitempty"` // round trip time in milliseconds
	BW  uint   `json:"bw,omitempty"`  // bandwidth in bits per second
}

// SupervisorConfig is the supervisory structure configuration for the ensemble
type SupervisorConfig struct {
	Strategy    SupervisorStrategy `json:"strategy,omitempty"`    // supervision strategy
	Allocations []string           `json:"allocations,omitempty"` // list of allocation IDs
	Children    []SupervisorConfig `json:"children,omitempty"`    // list of child supervisors
}

// SupervisorStrategy is the name of a supervision strategy
type SupervisorStrategy string

const (
	StrategyOneForOne  SupervisorStrategy = "OneForOne"
	StrategyAllForOne  SupervisorStrategy = "AllForOne"
	StrategyRestForOne SupervisorStrategy = "RestForOne"
)

// Validate validates the ensemble configuration
func (e *EnsembleConfig) Validate() error {
	if e == nil || e.V1 == nil {
		return errors.New("invalid ensemble config")
	}

	return nil
}

func (e *EnsembleConfig) Allocations() map[string]AllocationConfig {
	return e.V1.Allocations
}

func (e *EnsembleConfig) Allocation(allocID string) (AllocationConfig, bool) {
	a, ok := e.V1.Allocations[allocID]
	return a, ok
}

func (e *EnsembleConfig) Nodes() map[string]NodeConfig {
	return e.V1.Nodes
}

func (e *EnsembleConfig) Node(nodeID string) (NodeConfig, bool) {
	n, ok := e.V1.Nodes[nodeID]
	return n, ok
}

func (e *EnsembleConfig) EdgeConstraints() []EdgeConstraint {
	return e.V1.Edges
}

func (e *EnsembleConfig) Clone() EnsembleConfig {
	var clone EnsembleConfig

	bytes, err := json.Marshal(e)
	if err != nil {
		log.Errorf("error marshaling ensemble config: %s", err)
		return clone
	}

	if err := json.Unmarshal(bytes, &clone); err != nil {
		log.Errorf("unmarshaling ensemble config: %s", err)
	}

	return clone
}

func (l *Location) Equal(other Location) bool {
	if l.Continent != other.Continent {
		return false
	}

	if l.Country != "" && l.Country != other.Country {
		return false
	}

	if l.City != "" && l.City != other.City {
		return false
	}

	if l.ASN > 0 && l.ASN != other.ASN {
		return false
	}

	if l.ISP != "" && l.ISP != other.ISP {
		return false
	}

	return true
}
