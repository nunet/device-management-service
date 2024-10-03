package jobs

import (
	"gitlab.com/nunet/device-management-service/types"
)

// EnsembleConfig is the versioned structure that contains the ensemble configuration
type EnsembleConfig struct {
	V1 *EnsembleConfigV1
}

// EnsembleConfigV1 is version 1 of the configuration for an ensemble
type EnsembleConfigV1 struct {
	Allocations map[string]AllocationConfig // (named) allocations in the ensemble
	Nodes       map[string]NodeConfig       // (named) nodes in the ensemble
	Edges       []EdgeConstraint            // network edge constraints
	Supervisor  SupervisorConfig            // supervision structure
	Keys        map[string]string           // (named) ssh public keys relevant to the allocation
	Scripts     map[string]string           // (named) provisioning scripts
}

// AllocationConfig is the configuration of an allocation
type AllocationConfig struct {
	Executor    AllocationExecutor // the executor of the allocation
	Resources   types.Resources    // the HW resources required by the allocation
	Execution   types.SpecConfig   // the allocation execution configuration
	Volumes     []VolumeConfig     // premounted external volumes
	DNSName     string             // the internal DNS name of the allocation
	Keys        []string           // names of the authorized ssh keys for the allocation
	Provision   []string           // names of provisioning scripts to run (in order)
	HealthCheck string             // name of the script to run for health checks
}

// AllocationExecutor is the executor reoquired for the allocation
type AllocationExecutor string

const (
	ExecutorFirecracker AllocationExecutor = "firecracker"
	ExecutorDocker      AllocationExecutor = "docker"
)

// VolumeConfig is an externally mounted volume
type VolumeConfig struct {
	Name       string           // the name of the volume
	Type       string           // the type of the volume
	Remote     types.SpecConfig // the remout mount config
	Mountpoint string           // the mountpoint
}

// NodeConfig is the configuration of a distinct DMS node
type NodeConfig struct {
	Allocations []string            // the list of (named) allocations in the node
	Ports       []PortConfig        // the port mapping configuration for the node
	Location    LocationConstraints // the geographical location constraints for the node
	Peer        string              // (optional) a fixed peer for the node
	// TODO contract information
}

// LocationConstraints provides the node location placement constraints
type LocationConstraints struct {
	Accept []Location // acceptable location constraints (disjunction)
	Reject []Location // negative location constraints (conjunction); eg !USA for GPDR purposes
}

// Location is a geographical location on Planet Earth
type Location struct {
	Region  string // geographic region of the location (required)
	Country string // country (code or name) of the location (optional)
	City    string // city of the location; optional but country must be specified if not empty
	ASN     uint   // Autonomous System Number for the location (optional)
	ISP     string // Internet Service Provider name for the location (optional)
}

// PortConfig is the configuration for a port mapping a public port to a private port
// in an allocation
type PortConfig struct {
	Public     int    // the public port 0 for any
	Private    int    // the private mapping
	Allocation string // the allocation where the port is mapped
}

// EdgeConstraint is a constraint for a network edge between two nodes
type EdgeConstraint struct {
	S, T      string // (named) nodes connected by the edge
	RTT       uint   // maximum edge RTT in milliseconds
	BW        uint   // minimum edge bandwidth in Kbps
	Symmetric bool   // whether the constraint is symmetric (bidirectional)
}

// SupervisorConfig is the supervisory structure configuration for the ensemble
type SupervisorConfig struct {
	Strategy    SupervisorStrategy // the strategy for the supervision group
	Allocations []string           // allocations in this supervision group
	Children    []SupervisorConfig // allocation children for recursive groups
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
	// TODO
	return nil
}
