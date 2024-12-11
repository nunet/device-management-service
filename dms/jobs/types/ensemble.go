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
	"time"

	"gitlab.com/nunet/device-management-service/types"
)

type DeploymentStatus int

const (
	DeploymentStatusPreparing DeploymentStatus = iota
	DeploymentStatusGenerating
	DeploymentStatusCommitting
	DeploymentStatusProvisioning
	DeploymentStatusRunning
	DeploymentStatusFailed
	DeploymentStatusShuttingDown
	DeploymentStatusCompleted
)

func DeploymentStatusString(d DeploymentStatus) string {
	switch d {
	case DeploymentStatusPreparing:
		return "Preparing"
	case DeploymentStatusGenerating:
		return "Generating"
	case DeploymentStatusCommitting:
		return "Committing"
	case DeploymentStatusProvisioning:
		return "Provisioning"
	case DeploymentStatusRunning:
		return "Running"
	case DeploymentStatusFailed:
		return "Failed"
	case DeploymentStatusShuttingDown:
		return "ShuttingDown"
	case DeploymentStatusCompleted:
		return "Completed"
	default:
		return "Unknown"
	}
}

type OrchestratorView struct {
	types.BaseDBModel
	DeploymentID       string
	Cfg                EnsembleConfig
	Manifest           EnsembleManifest
	Status             DeploymentStatus
	DeploymentSnapshot DeploymentSnapshot
	PrivKey            []byte
}

type DeploymentSnapshot struct {
	// candidates keeps state of candidates while committing.
	Candidates map[string]Bid

	// Expiry is the time passed as an argument when calling Deploy()
	Expiry time.Time
}

// EnsembleConfig is the versioned structure that contains the ensemble configuration
type EnsembleConfig struct {
	V1 *EnsembleConfigV1 `json:"v1"`
}

// EnsembleConfigV1 is version 1 of the configuration for an ensemble
type EnsembleConfigV1 struct {
	Allocations map[string]AllocationConfig `json:"allocations"` // (named) allocations in the ensemble
	Nodes       map[string]NodeConfig       `json:"nodes"`       // (named) nodes in the ensemble
	Edges       []EdgeConstraint            `json:"edges"`       // network edge constraints
	Supervisor  SupervisorConfig            `json:"supervisor"`  // supervision structure
	Keys        map[string]string           `json:"keys"`        // (named) ssh public keys relevant to the allocation
	Scripts     map[string][]byte           `json:"scripts"`     // (named) provisioning scripts
}

// AllocationConfig is the configuration of an allocation
type AllocationConfig struct {
	Executor    AllocationExecutor        `json:"executor"`    // the executor of the allocation
	Resources   types.Resources           `json:"resources"`   // the HW resources required by the allocation
	Execution   types.SpecConfig          `json:"execution"`   // the allocation execution configuration
	DNSName     string                    `json:"dns_name"`    // the internal DNS name of the allocation
	Keys        []string                  `json:"keys"`        // names of the authorized ssh keys for the allocation
	Provision   []string                  `json:"provision"`   // names of provisioning scripts to run (in order)
	HealthCheck types.HealthCheckManifest `json:"healthcheck"` // name of the health check script
}

// AllocationExecutor is the executor reoquired for the allocation
type AllocationExecutor string

const (
	ExecutorFirecracker AllocationExecutor = "firecracker"
	ExecutorDocker      AllocationExecutor = "docker"
	ExecutorNull        AllocationExecutor = "null"
)

// NodeConfig is the configuration of a distinct DMS node
type NodeConfig struct {
	Allocations []string            `json:"allocations"` // list of allocation IDs
	Ports       []PortConfig        `json:"ports"`       // list of port mappings
	Location    LocationConstraints `json:"location"`    // location constraints
	Peer        string              `json:"peer"`        // peer ID to use for this node
	// TODO contract information
}

// LocationConstraints provides the node location placement constraints
type LocationConstraints struct {
	Accept []Location `json:"accept"` // list of accepted locations
	Reject []Location `json:"reject"` // list of rejected locations
}

// Location is a geographical location on Planet Earth
type Location struct {
	Region  string `json:"region"`  // geographical region
	Country string `json:"country"` // country code
	City    string `json:"city"`    // city name
	ASN     uint   `json:"asn"`     // autonomous system number
	ISP     string `json:"isp"`     // internet service provider
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
	S   string `json:"s"`   // source node ID
	T   string `json:"t"`   // target node ID
	RTT uint   `json:"rtt"` // round trip time in milliseconds
	BW  uint   `json:"bw"`  // bandwidth in bits per second
}

// SupervisorConfig is the supervisory structure configuration for the ensemble
type SupervisorConfig struct {
	Strategy    SupervisorStrategy `json:"strategy"`    // supervision strategy
	Allocations []string           `json:"allocations"` // list of allocation IDs
	Children    []SupervisorConfig `json:"children"`    // list of child supervisors
}

// SupervisoryStrategy is the name of a supervision strategy
type SupervisorStrategy string

const (
	StrategyOneForOne  SupervisorStrategy = "OneForOne"
	StrategyAllForOne  SupervisorStrategy = "AllForOne"
	StrategyRestForOne SupervisorStrategy = "RestForOne"
)

// config validation
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

func (l *Location) Includes(other Location) bool {
	if l.Region != other.Region {
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

func (e *EnsembleConfig) Clone() EnsembleConfig {
	var clone EnsembleConfig

	bytes, err := json.Marshal(e)
	if err != nil {
		log.Errorf("error marshaling ensemble config: %s", err)
		return clone
	}

	if err := json.Unmarshal(bytes, &clone); err != nil {
		log.Errorf("error unmarshaling ensemble config: %s", err)
	}

	return clone
}
