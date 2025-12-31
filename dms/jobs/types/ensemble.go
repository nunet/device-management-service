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
	"gitlab.com/nunet/device-management-service/utils"
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
	EscalationStrategy EscalationStrategy              `json:"escalation_strategy" yaml:"escalation_strategy"`         // escalation strategy (redeploy|teardown)
	Allocations        map[string]AllocationConfig     `json:"allocations" yaml:"allocations"`                         // (named) allocations in the ensemble
	Nodes              map[string]NodeConfig           `json:"nodes" yaml:"nodes"`                                     // (named) nodes in the ensemble
	Edges              []EdgeConstraint                `json:"edges,omitempty" yaml:"edges,omitempty"`                 // network edge constraints
	Supervisor         SupervisorConfig                `json:"supervisor,omitempty" yaml:"supervisor,omitempty"`       // supervision structure
	Keys               map[string]string               `json:"keys,omitempty" yaml:"keys,omitempty"`                   // (named) ssh public keys relevant to the allocation
	Scripts            map[string][]byte               `json:"scripts,omitempty" yaml:"scripts,omitempty"`             // (named) provisioning scripts
	Subnet             SubnetConfig                    `json:"subnet,omitempty" yaml:"subnet,omitempty"`               // subnet config
	ExcludePeers       []string                        `json:"exclude_peers,omitempty" yaml:"exclude_peers,omitempty"` // list of peers to not deploy on
	Metadata           map[string]string               `json:"metadata,omitempty" yaml:"metadata,omitempty"`           // metadata (arbitrary key-value pairs)
	Contracts          map[string]types.ContractConfig `json:"contracts,omitempty" yaml:"contracts,omitempty"`         // (named) contracts between parties
}

type EscalationStrategy string

const (
	EscalationStrategyRedeploy EscalationStrategy = "redeploy"
	EscalationStrategyTeardown EscalationStrategy = "teardown"
)

// AllocationConfig is the configuration of an allocation
type AllocationConfig struct {
	Executor        AllocationExecutor        `json:"executor" yaml:"executor"`                                     // the executor of the allocation
	Type            AllocationType            `json:"type" yaml:"type"`                                             // the type of allocation (service vs task)
	Resources       types.Resources           `json:"resources" yaml:"resources"`                                   // the HW resources required by the allocation
	Execution       types.SpecConfig          `json:"execution" yaml:"execution"`                                   // the allocation execution configuration
	DNSName         string                    `json:"dns_name,omitempty" yaml:"dns_name,omitempty"`                 // the internal DNS name of the allocation
	Keys            []types.AllocationKey     `json:"keys,omitempty" yaml:"keys,omitempty"`                         // names of the authorized ssh keys for the allocation
	Provision       []string                  `json:"provision,omitempty" yaml:"provision,omitempty"`               // names of provisioning scripts to run (in order)
	HealthCheck     types.HealthCheckManifest `json:"healthcheck,omitempty" yaml:"healthcheck,omitempty"`           // name of the health check script
	Volume          []types.VolumeConfig      `json:"volume,omitempty" yaml:"volume,omitempty"`                     // unified storage configuration (optional)
	FailureRecovery AllocationFailureRecovery `json:"failure_recovery,omitempty" yaml:"failure_recovery,omitempty"` // failure recovery (stay_down|one_for_one|one_for_all|rest_for_one)
	DependsOn       []string                  `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`             // list of allocations that this allocation depends on
}

// AllocationFailureRecovery
type AllocationFailureRecovery string

const (
	AllocationFailureRecoveryStayDown   AllocationFailureRecovery = "stay_down"
	AllocationFailureRecoveryOneForOne  AllocationFailureRecovery = "one_for_one"
	AllocationFailureRecoveryOneForAll  AllocationFailureRecovery = "one_for_all"
	AllocationFailureRecoveryRestForOne AllocationFailureRecovery = "rest_for_one"
)

// NodeConfig is the configuration of a distinct DMS node
type NodeConfig struct {
	Allocations     []string            `json:"allocations" yaml:"allocations"`                               // list of allocation IDs
	Ports           []PortConfig        `json:"ports,omitempty" yaml:"ports,omitempty"`                       // list of port mappings
	Location        LocationConstraints `json:"location,omitempty" yaml:"location,omitempty"`                 // location constraints
	Peer            string              `json:"peer,omitempty" yaml:"peer,omitempty"`                         // peer ID to use for this node
	Redundancy      int                 `json:"redundancy,omitempty" yaml:"redundancy,omitempty"`             // number of redundant nodes
	FailureRecovery NodeFailureRecovery `json:"failure_recovery,omitempty" yaml:"failure_recovery,omitempty"` // failure recovery (stay_down|restart|redeploy)
	// TODO contract information
}

// NodeFailureRecovery is the failure recovery strategy for a node
type NodeFailureRecovery string

const (
	NodeFailureRecoveryStayDown NodeFailureRecovery = "stay_down"
	NodeFailureRecoveryRestart  NodeFailureRecovery = "restart"
	NodeFailureRecoveryRedeploy NodeFailureRecovery = "redeploy"
)

// LocationConstraints provides the node location placement constraints
type LocationConstraints struct {
	Accept []Location `json:"accept,omitempty" yaml:"accept,omitempty"` // list of accepted locations
	Reject []Location `json:"reject,omitempty" yaml:"reject,omitempty"` // list of rejected locations
}

// Location is a geographical location on Planet Earth
type Location struct {
	Continent string `json:"continent,omitempty" yaml:"continent,omitempty"` // geographical region
	Country   string `json:"country,omitempty" yaml:"country,omitempty"`     // country code
	City      string `json:"city,omitempty" yaml:"city,omitempty"`           // city name
	ASN       uint   `json:"asn,omitempty" yaml:"asn,omitempty"`             // autonomous system number
	ISP       string `json:"isp,omitempty" yaml:"isp,omitempty"`             // internet service provider
}

// PortConfig is the configuration for a port mapping a public port to a private port
// in an allocation
type PortConfig struct {
	Public     int    `json:"public" yaml:"public"`         // public port number
	Private    int    `json:"private" yaml:"private"`       // private port number
	Allocation string `json:"allocation" yaml:"allocation"` // allocation ID
}

// EdgeConstraint is a constraint for a network edge between two nodes
type EdgeConstraint struct {
	S   string `json:"s" yaml:"s"`                         // source node ID
	T   string `json:"t" yaml:"t"`                         // target node ID
	RTT uint   `json:"rtt,omitempty" yaml:"rtt,omitempty"` // round trip time in milliseconds
	BW  uint   `json:"bw,omitempty" yaml:"bw,omitempty"`   // bandwidth in bits per second
}

// SupervisorConfig is the supervisory structure configuration for the ensemble
type SupervisorConfig struct {
	Strategy    SupervisorStrategy `json:"strategy,omitempty" yaml:"strategy,omitempty"`       // supervision strategy
	Allocations []string           `json:"allocations,omitempty" yaml:"allocations,omitempty"` // list of allocation IDs
	Children    []SupervisorConfig `json:"children,omitempty" yaml:"children,omitempty"`       // list of child supervisors
}

// SupervisorStrategy is the name of a supervision strategy
type SupervisorStrategy string

type SubnetConfig struct {
	Join bool `json:"join,omitempty" yaml:"join,omitempty"` // for orchestrator to join the subnet
}

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

func (e *EnsembleConfig) Contracts() map[string]types.ContractConfig {
	return e.V1.Contracts
}

func (e *EnsembleConfig) Contract(contractID string) (types.ContractConfig, bool) {
	c, ok := e.V1.Contracts[contractID]
	return c, ok
}

func (e *EnsembleConfig) Allocations() map[string]AllocationConfig {
	return e.V1.Allocations
}

func (e *EnsembleConfig) Allocation(name string) (AllocationConfig, bool) {
	a, ok := e.V1.Allocations[name]
	return a, ok
}

// buildStandbyNodes constructs standby node configurations for nodes with redundancy
func buildStandbyNodes(nodes map[string]NodeConfig, nodeIDGenerator types.NodeIDGenerator) map[string]NodeConfig {
	standbyNodes := make(map[string]NodeConfig)
	for nodeID, nodeConfig := range nodes {
		if nodeConfig.Redundancy == 0 {
			continue
		}
		for i := 1; i <= nodeConfig.Redundancy; i++ {
			standbyNodeID, err := nodeIDGenerator.GenerateStandbyNodeID(nodeID, i)
			if err != nil {
				// Log error and skip this standby node
				continue
			}
			ncfg := NodeConfig{
				Allocations:     nodeConfig.Allocations,
				Ports:           nodeConfig.Ports,
				Location:        nodeConfig.Location,
				FailureRecovery: nodeConfig.FailureRecovery,
			}
			standbyNodes[standbyNodeID] = ncfg
		}
	}
	return standbyNodes
}

// Nodes returns a map of all nodes including standby nodes (using default generator)
func (e *EnsembleConfig) Nodes() map[string]NodeConfig {
	return e.NodesWithGenerator(types.NewDefaultNodeIDGenerator())
}

// NodesWithGenerator returns a map of all nodes including standby nodes using the provided generator
func (e *EnsembleConfig) NodesWithGenerator(nodeIDGenerator types.NodeIDGenerator) map[string]NodeConfig {
	result := make(map[string]NodeConfig)

	// First add all primary nodes
	for nodeID, nodeConfig := range e.V1.Nodes {
		result[nodeID] = nodeConfig
	}

	// Then add standby nodes
	for nodeID, nodeConfig := range buildStandbyNodes(e.V1.Nodes, nodeIDGenerator) {
		result[nodeID] = nodeConfig
	}

	return result
}

// PrimaryNodes returns only the primary nodes (no standby nodes)
func (e *EnsembleConfig) PrimaryNodes() map[string]NodeConfig {
	return e.V1.Nodes
}

func (e *EnsembleConfig) Node(nodeID string) (NodeConfig, bool) {
	return e.NodeWithGenerator(nodeID, types.NewDefaultNodeIDGenerator())
}

func (e *EnsembleConfig) NodeWithGenerator(nodeID string, nodeIDGenerator types.NodeIDGenerator) (NodeConfig, bool) {
	// Check if it's a primary node directly from the configuration
	if n, ok := e.V1.Nodes[nodeID]; ok {
		return n, true
	}

	// Check if it's a standby node using the generator
	isStandby, primaryNodeID, standbyIndex, err := nodeIDGenerator.ParseNodeID(nodeID)
	if err != nil || !isStandby {
		return NodeConfig{}, false
	}

	// Get the primary node
	primaryNode, ok := e.V1.Nodes[primaryNodeID]
	if !ok || standbyIndex < 1 || standbyIndex > primaryNode.Redundancy {
		return NodeConfig{}, false
	}

	// Return a copy of the primary node config for this standby
	return primaryNode, true
}

func (e *EnsembleConfig) AllocationsForNode(node string) map[string]AllocationConfig {
	allocations := make(map[string]AllocationConfig)
	nodeConfig, ok := e.Node(node)
	if !ok {
		return allocations
	}

	for _, allocName := range nodeConfig.Allocations {
		if alloc, ok := e.Allocation(allocName); ok {
			allocations[allocName] = alloc
		}
	}

	return allocations
}

func (e *EnsembleConfig) PortsForAllocation(allocation string) []PortConfig {
	var ports []PortConfig

	for _, node := range e.Nodes() {
		for _, port := range node.Ports {
			if port.Allocation == allocation {
				ports = append(ports, port)
			}
		}
	}

	return ports
}

func (e *EnsembleConfig) EdgeConstraints() []EdgeConstraint {
	return e.V1.Edges
}

func (e *EnsembleConfig) Subnet() SubnetConfig {
	return e.V1.Subnet
}

func (e *EnsembleConfig) AddNodeAndAllocations(
	name string, node NodeConfig,
	allocs map[string]AllocationConfig,
) {
	e.V1.Nodes[name] = node

	for k, v := range allocs {
		if utils.SliceContains(node.Allocations, k) {
			e.V1.Allocations[k] = v
		}
	}
}

func (e *EnsembleConfig) RemoveNodeAndAllocations(
	nodeName string,
) {
	if nodeCfg, ok := e.Node(nodeName); ok {
		delete(e.V1.Nodes, nodeName)
		for _, alloc := range nodeCfg.Allocations {
			delete(e.V1.Allocations, alloc)
		}
	}
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

func (l Location) Satisfies(constraints LocationConstraints) bool {
	// Accept list takes precedence
	if len(constraints.Accept) > 0 {
		for _, a := range constraints.Accept {
			if a.Equal(l) {
				return true
			}
		}
		return false
	}

	// Otherwise enforce Reject list
	for _, r := range constraints.Reject {
		if r.Equal(l) {
			return false
		}
	}
	return true
}
