package jobtypes

import (
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/types"
)

// TODO: keeping here temporarily. We must organize types and behavior payloads.
// issue: https://gitlab.com/nunet/device-management-service/-/issues/893 (part 2)
// TODO (wrong nomenclature): AllocationDeploymentRequest -> EnsembleDeploymentRequest
type AllocationDeploymentRequest struct {
	EnsembleID  string
	NodeID      string
	Allocations map[string]AllocationDeploymentConfig
}

type AllocationDeploymentConfig struct {
	Type             AllocationType
	Executor         AllocationExecutor
	Resources        types.Resources
	Execution        types.SpecConfig
	ProvisionScripts map[string][]byte
	Keys             []types.AllocationKey
	Volume           []types.VolumeConfig
}

type AllocationDeploymentResponse struct {
	OK          bool
	Error       string
	Allocations map[string]actor.Handle
}
