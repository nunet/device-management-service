package jobs

import (
	"time"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	BidRequestTopic    = "/nunet/deployment"
	BidRequestBehavior = "/dms/deployment/request"
	BidRequestTimeout  = 5 * time.Second
	BidReplyBehavior   = "/dms/deployment/bid"

	VerifyEdgeConstraintBehavior = "/dms/deployment/constraint/edge"
	VerifyEdgeConstraintTimeout  = 5 * time.Second

	CommitDeploymentBehavior     = "/dms/deployment/commit"
	CommitDeploymentTimeout      = 3 * time.Second
	AllocationDeploymentBehavior = "/dms/deployment/allocate"
	AllocationDeploymentTimeout  = 3 * time.Second
	RevertDeploymentBehavior     = "/dms/deployment/revert"

	MinEnsembleDeploymentTime = 15 * time.Second

	MaxBidMultiplier = 8
)

type VerifyEdgeConstraintRequest struct {
	EnsembleID string // the ensemble identifier
	S, T       string // the peer IDs of the edge S->T
	RTT        uint   //  maximum RTT in ms (if > 0)
	BW         uint   // minim BW in Kbps
}

type VerifyEdgeConstraintResponse struct {
	OK    bool
	Error string
}

type CommitDeploymentRequest struct {
	EnsembleID string
	NodeID     string
}

type CommitDeploymentResponse struct {
	OK    bool
	Error string
}

type AllocationDeploymentRequest struct {
	EnsembleID  string
	NodeID      string
	Allocations map[string]AllocationDeploymentConfig
}

type AllocationDeploymentConfig struct {
	Executor  AllocationExecutor
	Resources types.Resources
}

type AllocationDeploymentResponse struct {
	OK          bool
	Error       string
	Allocations map[string]actor.Handle
}

type RevertDeploymentMessage struct {
	EnsembleID string
	NodeID     string
}
