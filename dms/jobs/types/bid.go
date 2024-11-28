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

	"github.com/libp2p/go-libp2p/core/peer"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/types"
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
	GeneralRequirements struct {
		PrivilegedDocker bool
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
	Signature  []byte
}

const bidPrefix = "dms-bid-"

func (b *EnsembleBidRequest) Validate() error {
	if b == nil {
		return errors.New("ensemble bid request is nil")
	}

	if b.ID == "" {
		return errors.New("ensemble id is empty")
	}

	if len(b.Request) == 0 {
		return errors.New("ensemble with empty requests")
	}

	return nil
}

func (b *Bid) SignatureData() ([]byte, error) {
	bidV1Copy := *b.V1
	bidV1Copy.Signature = nil
	data, err := json.Marshal(&bidV1Copy)
	if err != nil {
		return nil, fmt.Errorf("signature data: %w", err)
	}

	result := make([]byte, len(bidPrefix)+len(data))
	copy(result, []byte(bidPrefix))
	copy(result[len(bidPrefix):], data)

	return result, nil
}

func (b *Bid) Sign(key did.Provider) error {
	data, err := b.SignatureData()
	if err != nil {
		return fmt.Errorf("unable to create bid signature data")
	}
	sig, err := key.Sign(data)
	if err != nil {
		return fmt.Errorf("unable to sign the bid")
	}
	b.V1.Signature = sig
	return nil
}

func (b *Bid) Validate() error {
	if b.V1 == nil {
		return fmt.Errorf("bid V1 is nil")
	}

	p, err := peer.Decode(b.V1.Peer)
	if err != nil {
		return fmt.Errorf("failed to decode bid's peer id: %w", err)
	}

	pubKey, err := p.ExtractPublicKey()
	if err != nil {
		return fmt.Errorf("failed to extract public key: %w", err)
	}

	signData, err := b.SignatureData()
	if err != nil {
		return fmt.Errorf("unable to get bid signature data")
	}

	ok, err := pubKey.Verify(signData, b.V1.Signature)
	if err != nil {
		return fmt.Errorf("failed to verify signature: %w", err)
	}

	if !ok {
		return errors.New("signature verification failed")
	}

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

func (b *Bid) Handle() actor.Handle {
	return b.V1.Handle
}

func (b *Bid) Location() Location {
	return b.V1.Location
}
