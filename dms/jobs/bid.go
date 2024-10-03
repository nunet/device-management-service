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

	BidReplyBehavior = "/dms/deployment/bid"

	MinEnsembleDeploymentTime = 15 * time.Second
)

// EnsembleBidRequest is a request for a bids pertaining to an ensemble
//
// Note: At the moment, we embed a bid request for each node
// This is fine for small deployments, and a small network, which is what we have.
// For large deployments however, this won't scale and we will have to create aggregate
// bid requests for related group of nodes and also handle them with bid request
// aggregators who control multiple nodes.
type EnsembleBidRequest struct {
	ID            string       // unique identifier of an ensemble (in the context of the orchestrator)
	Request       []BidRequest // list of node bid requests
	PeerExclusion []string     // list of peers to exclude from bidding
}

// BidRequest is a versioned bid request
type BidRequest struct {
	V1 *BidRequestV1
}

// BidRequestV1 is v1 of bid requests for a node to use for deployment
type BidRequestV1 struct {
	NodeID      string               // unique identifier for a node, within the context of an ensemble
	Executors   []AllocationExecutor // list of required executors to support the allocation(s)
	Resources   types.Resources      // (aggregate) required hardware resources
	Location    LocationConstraints  // node location constraints
	PublicPorts struct {
		Static  []int // statically configured public ports
		Dynamic int   // number of dynamic ports
	}
}

// Bid is the version struct for Bids in response to a bid request
type Bid struct {
	V1 *BidV1
}

// BidV1 is v1 of the bid structure
type BidV1 struct {
	EnsembleID string       // unique identifier for the ensemble
	NodeID     string       // unique identifier for a node; matches the id of the BidRequest to which this bid pertains
	Peer       string       // the peer ID of the node
	Location   Location     // the location of the node
	Handle     actor.Handle // the handle of the actor submitting the bid
	// TODO signature from Peer
}

func (b *EnsembleBidRequest) Validate() error {
	// TODO
	return nil
}

// TODO pass the envelope for verification
func (b *Bid) Validate() error {
	// TODO
	return nil
}

func (b *Bid) EnsembleID() string {
	return b.V1.EnsembleID
}

func (b *Bid) NodeID() string {
	return b.V1.NodeID
}

func (b *Bid) Peer() string {
	return b.V1.Peer
}

func (b *Bid) Location() Location {
	return b.V1.Location
}
