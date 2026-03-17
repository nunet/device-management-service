// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

// TestHandleBidRequest tests the handleBidRequest function of the Node struct.
// it needs to be covered more with integration tests
func TestHandleBidRequest(t *testing.T) {
	t.Parallel()

	observability.SetNoOpMode(true)
	// ctx := context.Background()
	t.Run("broadcast to self", func(t *testing.T) {
		t.Parallel()

		netSubstrate := network.NewSubstrate()
		node, _, _ := newMockNode(t, netSubstrate)

		emptyBidRequest := getBidRequest()
		emptyBidRequest.Request = []jobtypes.BidRequest{}

		msg := getBroadcastMsg(t, node.actor.Handle(), emptyBidRequest)

		node.handleBidRequest(msg)

		// shouldn't exist in answered bids
		bidState, exists := node.getBid(emptyBidRequest.ID)
		assert.False(t, exists)
		assert.Nil(t, bidState)
		assert.False(t, node.bidAnswered(emptyBidRequest.ID, emptyBidRequest.Nonce))
	})

	t.Run("Ignore bid if node is not onboarded", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.BidRequestBehavior)

		emptyBidRequest := getBidRequest()
		emptyBidRequest.Request = []jobtypes.BidRequest{}

		// create message
		msg := getBroadcastMsg(t, sActor.Handle(), emptyBidRequest)

		node.handleBidRequest(msg)

		// shouldn't exist in answered bids
		bidState, exists := node.getBid(emptyBidRequest.ID)
		assert.False(t, exists)
		assert.Nil(t, bidState)
		assert.False(t, node.bidAnswered(emptyBidRequest.ID, emptyBidRequest.Nonce))
	})

	t.Run("Ignore bid if unmarshalling fails", func(t *testing.T) {
		t.Parallel()

		// setup node and sendor actor
		node, sActor, _ := newMockNodeWithSender(t, behaviors.BidRequestBehavior)

		// onboard node
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		// create message
		msg := getBroadcastMsg(t, sActor.Handle(), "invalid-bid-request")

		node.handleBidRequest(msg)

		// shouldn't exist in answered bids
		assert.Empty(t, node.answeredBids)
		assert.Empty(t, node.bids)
	})

	t.Run("peer exclusion", func(t *testing.T) {
		t.Parallel()

		// setup node and sendor actor
		node, sActor, _ := newMockNodeWithSender(t, behaviors.BidRequestBehavior)

		// onboard node
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)
		// create message
		bidRequest := getBidRequest()
		bidRequest.PeerExclusion = []string{node.network.Stat().ID}
		msg := getBroadcastMsg(t, sActor.Handle(), bidRequest)

		node.handleBidRequest(msg)

		// shouldn't exist in answered bids
		assert.Empty(t, node.answeredBids)
		assert.Empty(t, node.bids)
	})

	t.Run("desired executor not found", func(t *testing.T) {
		t.Parallel()

		// setup node and sendor actor
		node, sActor, _ := newMockNodeWithSender(t, behaviors.BidRequestBehavior)

		// onboard node
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		// remove executor
		node.lock.Lock()
		for k := range node.executors {
			delete(node.executors, k)
		}
		node.lock.Unlock()

		// create message
		bidRequest := getBidRequest()
		msg := getBroadcastMsg(t, sActor.Handle(), bidRequest)

		node.handleBidRequest(msg)

		// shouldn't exist in answered bids
		assert.Empty(t, node.answeredBids)
		assert.Empty(t, node.bids)
	})

	t.Run("bid already answered", func(t *testing.T) {
		t.Parallel()

		// setup node and sendor actor
		node, sActor, _ := newMockNodeWithSender(t, behaviors.BidRequestBehavior)

		// onboard node
		mockOnboarding(t, node, MockTotalCPU/3, MockTotalRAM/3, MockTotalDisk/3)
		// create message
		bidRequest := getBidRequest()

		// simulate bid already answered
		node.answeredBids = make(map[string][]uint64)
		node.answeredBids[bidRequest.ID] = []uint64{bidRequest.Nonce}

		msg := getBroadcastMsg(t, sActor.Handle(), bidRequest)

		node.handleBidRequest(msg)

		// shouldn't exist in answered bids
		assert.Equal(t, 1, len(node.answeredBids)) // only the simulated
		assert.Empty(t, node.bids)
	})

	t.Run("privileged docker", func(t *testing.T) {
		t.Parallel()

		// setup node and sendor actor
		node, sActor, _ := newMockNodeWithSender(t, behaviors.BidRequestBehavior)

		// onboard node
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		// create message
		bidRequest := getBidRequest()

		// set privileged docker when node doesn't allow it
		node.dmsConfig.Job.AllowPrivilegedDocker = false
		bidRequest.Request[0].V1.GeneralRequirements.PrivilegedDocker = true
		msg := getBroadcastMsg(t, sActor.Handle(), bidRequest)

		node.handleBidRequest(msg)

		// shouldn't exist in answered bids
		assert.Empty(t, node.answeredBids)
		assert.Empty(t, node.bids)
	})

	t.Run("successful bid reply then repeated same request", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.BidRequestBehavior)

		// grant broadcast cap
		actor.AllowBroadcast(t,
			sActor.Security().Capability(), node.actor.Security().Capability(),
			sActor.Security().Capability().Trust(), node.actor.Security().Capability().Trust(),
			sActor.Handle().DID, node.actor.Handle().DID,
			behaviors.BidRequestTopic, ucan.Capability(behaviors.BidRequestBehavior),
		)

		// sender grant bid reply cap
		actor.AllowReciprocal(t,
			sActor.Security().Capability(),
			sActor.Security().Capability().Trust(),
			sActor.Handle().DID,
			node.actor.Handle().DID,
			behaviors.BidReplyBehavior)

		err := sActor.Security().Grant(
			node.actor.Handle().DID,
			sActor.Handle().DID,
			[]ucan.Capability{
				ucan.Capability(behaviors.BidReplyBehavior),
			},
			time.Minute,
		)
		require.NoError(t, err)

		// start
		err = sActor.Start()
		require.NoError(t, err)

		err = node.Start()
		require.NoError(t, err)
		defer func() {
			err := node.Stop()
			assert.NoError(t, err, "Stop should not return an error")
		}()

		// onboard node
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		msg := getBroadcastMsg(t, sActor.Handle(), getBidRequest())
		require.NoError(t, err)

		replyChan := make(chan actor.Envelope, 1)
		defer close(replyChan)
		// sender receiver behavior
		err = sActor.AddBehavior(behaviors.BidReplyBehavior, func(msg actor.Envelope) {
			defer msg.Discard()

			replyChan <- msg
		})
		require.NoError(t, err)

		err = sActor.Publish(msg)
		require.NoError(t, err)

		// wait for reply
		reply := <-replyChan
		assert.NotNil(t, reply)
		assert.Equal(t, reply.From.Address.HostID, node.actor.Handle().Address.HostID)
		assert.Equal(t, reply.To.Address.HostID, sActor.Handle().Address.HostID)

		// should exist in answered bids
		assert.NotEmpty(t, node.answeredBids)
		assert.NotEmpty(t, node.bids)
	})

	t.Run("contracts required but missing - no bid placed", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.BidRequestBehavior)

		// mark node as onboarded so it can normally bid
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		// enable contract enforcement
		node.dmsConfig.Job.RequireContractsForDeployment = true

		// standard bid request from helper has no contracts
		bidRequest := getBidRequest()
		require.Len(t, bidRequest.Request, 1)
		require.Nil(t, bidRequest.Request[0].V1.Contracts)

		msg := getBroadcastMsg(t, sActor.Handle(), bidRequest)

		node.handleBidRequest(msg)

		// With enforcement enabled and no contracts, node must not store a bid or mark it answered
		assert.Empty(t, node.answeredBids)
		assert.Empty(t, node.bids)
	})

	t.Run("contracts required and it exist - bid placed", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.BidRequestBehavior)

		// mark node as onboarded so it can normally bid
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		// enable contract enforcement
		node.dmsConfig.Job.RequireContractsForDeployment = false

		// standard bid request from helper has no contracts
		bidRequest := getBidRequest()
		require.Len(t, bidRequest.Request, 1)
		require.Nil(t, bidRequest.Request[0].V1.Contracts)

		msg := getBroadcastMsg(t, sActor.Handle(), bidRequest)

		node.handleBidRequest(msg)

		// With enforcement enabled and no contracts, node must not store a bid or mark it answered
		assert.Len(t, node.answeredBids, 1)
		assert.Len(t, node.bids, 1)
	})

	t.Run("contracts required but bid request from self", func(t *testing.T) {
		t.Parallel()

		node, _, _ := newMockNodeWithSender(t, behaviors.BidRequestBehavior)

		// mark node as onboarded so it can normally bid
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		// enable contract enforcement
		node.dmsConfig.Job.RequireContractsForDeployment = true

		// standard bid request from helper has no contracts
		bidRequest := getBidRequest()
		require.Len(t, bidRequest.Request, 1)
		require.Nil(t, bidRequest.Request[0].V1.Contracts)

		msg, err := actor.Message(
			node.actor.Handle(), // self as sender
			node.actor.Handle(), // self as receiver
			behaviors.BidRequestBehavior,
			bidRequest,
			actor.WithMessageReplyTo(behaviors.BidReplyBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		node.handleBidRequest(msg)

		// enforcement enabled but since request is from self, the node should go
		// through with hanlding the bid request
		assert.NotEmpty(t, node.answeredBids)
		assert.NotEmpty(t, node.bids)
	})
}

func getBidRequest() jobtypes.EnsembleBidRequest {
	return jobtypes.EnsembleBidRequest{
		ID:    "ensemble-id",
		Nonce: 1,
		Request: []jobtypes.BidRequest{
			{
				V1: &jobtypes.BidRequestV1{
					NodeID: "node-id-1",
					Executors: []jobtypes.AllocationExecutor{
						jobtypes.ExecutorDocker,
					},
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 1},
						RAM:  types.RAM{Size: 1},
						Disk: types.Disk{Size: 1},
					},
				},
			},
		},
	}
}

func getBroadcastMsg(t *testing.T, sender actor.Handle, payload interface{}) actor.Envelope {
	t.Helper()

	msg, err := actor.Message(
		sender,
		actor.Handle{},
		behaviors.BidRequestBehavior,
		payload,
		actor.WithMessageTopic(behaviors.BidRequestTopic),
		actor.WithMessageReplyTo(behaviors.BidReplyBehavior),
		actor.WithMessageExpiry(uint64(time.Now().Add(time.Minute).UnixNano())),
	)
	require.NoError(t, err)
	return msg
}
