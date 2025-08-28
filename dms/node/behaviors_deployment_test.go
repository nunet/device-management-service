// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	"gitlab.com/nunet/device-management-service/executor/null"
	"gitlab.com/nunet/device-management-service/types"
)

func TestCommitDeployment(t *testing.T) {
	t.Parallel()

	ensembleID := "test-ensemble"
	allocationName := "test-allocation"
	nodeID := "node-id-1"

	resrc := types.Resources{
		CPU:  types.CPU{Cores: 1},
		RAM:  types.RAM{Size: 1},
		Disk: types.Disk{Size: 1},
	}

	mockBidRequest := jobtypes.BidRequest{
		V1: &jobtypes.BidRequestV1{
			NodeID: nodeID,
			Executors: []jobtypes.AllocationExecutor{
				jobtypes.ExecutorDocker,
			},
			Resources: resrc,
		},
	}

	// template message
	msg, err := actor.Message(
		actor.Handle{},
		actor.Handle{},
		behaviors.CommitDeploymentBehavior,
		orchestrator.CommitDeploymentRequest{
			EnsembleID:     ensembleID,
			AllocationName: allocationName,
			NodeID:         nodeID,
			Resources: types.CommittedResources{
				Resources:    resrc,
				AllocationID: allocationName,
			},
		},
		actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
	)
	require.NoError(t, err)

	t.Run("successfully commit deployment", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.CommitDeploymentBehavior)
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		node.storeBid(ensembleID, 1, mockBidRequest)

		msg.From = sActor.Handle()
		msg.To = node.actor.Handle()
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.CommitDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
	})

	t.Run("bid not found", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.CommitDeploymentBehavior)

		msg.From = sActor.Handle()
		msg.To = node.actor.Handle()
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.CommitDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, "no bid requests for ensemble id: "+ensembleID)
	})

	t.Run("bid expired", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.CommitDeploymentBehavior)

		node.bids[ensembleID] = &bidState{
			expire:  time.Now().Add(3 * time.Second),
			nonce:   1,
			request: mockBidRequest,
		}

		// wait for the bid to expire
		time.Sleep(4 * time.Second)

		// create message
		msg.From = sActor.Handle()
		msg.To = node.actor.Handle()
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.CommitDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		require.Contains(t, resp.Error, fmt.Sprintf("%s has expired", ensembleID))
	})

	t.Run("allocator commit error", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.CommitDeploymentBehavior)

		tooMuchResources := types.Resources{
			CPU:  types.CPU{Cores: 1000},
			RAM:  types.RAM{Size: 1000},
			Disk: types.Disk{Size: 1000},
		}

		node.storeBid(ensembleID, 1, mockBidRequest)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.CommitDeploymentBehavior,
			orchestrator.CommitDeploymentRequest{
				EnsembleID:     ensembleID,
				AllocationName: allocationName,
				NodeID:         nodeID,
				Resources: types.CommittedResources{
					Resources:    tooMuchResources,
					AllocationID: allocationName,
				},
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		assert.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.CommitDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		require.Contains(t, resp.Error, fmt.Sprintf("%s: check hardware capacity: no free resources", allocationName))
	})
}

func TestHandleNewDeployment(t *testing.T) {
	t.Parallel()

	ensembleConfig := jobtypes.EnsembleConfig{}

	msg, err := actor.Message(
		actor.Handle{},
		actor.Handle{},
		behaviors.NewDeploymentBehavior,
		NewDeploymentRequest{
			Ensemble: ensembleConfig,
		},
		actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
	)
	require.NoError(t, err)

	t.Run("empty ensemble config", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.NewDeploymentBehavior)

		msg.From = sActor.Handle()
		msg.To = node.actor.Handle()
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp NewDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, "ERROR", resp.Status)
		assert.Contains(t, resp.Error, "empty ensemble config")
	})

	t.Run("deployment time too short", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.NewDeploymentBehavior)

		shortExpiryMsg, err := actor.Message(
			actor.Handle{},
			actor.Handle{},
			behaviors.NewDeploymentBehavior,
			NewDeploymentRequest{
				Ensemble: ensembleConfig,
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(1*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		shortExpiryMsg.From = sActor.Handle()
		shortExpiryMsg.To = node.actor.Handle()
		replyChan, err := sActor.Invoke(shortExpiryMsg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp NewDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, "ERROR", resp.Status)
		assert.Contains(t, resp.Error, "requested deployment time too short")
	})

	t.Run("successful", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.NewDeploymentBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.NewDeploymentBehavior,
			NewDeploymentRequest{
				Ensemble: jobtypes.EnsembleConfig{
					V1: &jobtypes.EnsembleConfigV1{
						Allocations: map[string]jobtypes.AllocationConfig{},
						Nodes:       map[string]jobtypes.NodeConfig{},
						Supervisor:  jobtypes.SupervisorConfig{},
						Subnet:      jobtypes.SubnetConfig{},
					},
				},
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp NewDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, "OK", resp.Status)
		assert.Empty(t, resp.Error)
	})
}

func TestHandleDeploymentList(t *testing.T) {
	t.Parallel()

	// Add behavior to test
	node, sActor, _ := newMockNodeWithSender(t, behaviors.DeploymentListBehavior)

	// seed deployment data
	eCfgWithMetadata := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Metadata:    map[string]string{"name": "test-deploy"},
			Allocations: map[string]jobtypes.AllocationConfig{},
			Nodes:       map[string]jobtypes.NodeConfig{},
			Supervisor:  jobtypes.SupervisorConfig{},
			Subnet:      jobtypes.SubnetConfig{},
		},
	}
	mockOrch, err := node.createOrchestrator(context.Background(), eCfgWithMetadata)
	require.NoError(t, err)
	require.NotNil(t, mockOrch)
	err = mockOrch.Deploy(time.Now().Add(2 * time.Minute))
	require.NoError(t, err)

	eCfgWithoutMetadata := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{},
			Nodes:       map[string]jobtypes.NodeConfig{},
			Supervisor:  jobtypes.SupervisorConfig{},
			Subnet:      jobtypes.SubnetConfig{},
		},
	}
	mockOrchWithoutM, err := node.createOrchestrator(context.Background(), eCfgWithoutMetadata)
	require.NoError(t, err)
	require.NotNil(t, mockOrchWithoutM)
	err = mockOrchWithoutM.Deploy(time.Now().Add(2 * time.Minute))
	require.NoError(t, err)

	t.Run("without metadata filtering expect all", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentListBehavior,
			DeploymentListRequest{}, // without metadata filtering
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DeploymentListResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(resp.Deployments))
		status, ok := resp.Deployments[mockOrch.ID()]
		assert.True(t, ok)
		assert.Equal(t, jobtypes.DeploymentStatusPreparing.String(), status)
		status, ok = resp.Deployments[mockOrchWithoutM.ID()]
		assert.True(t, ok)
		assert.Equal(t, jobtypes.DeploymentStatusPreparing.String(), status)
	})

	t.Run("with metadata", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentListBehavior,
			DeploymentListRequest{
				Metadata: map[string]string{
					"name": "test-deploy",
				},
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DeploymentListResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Equal(t, 1, len(resp.Deployments))
		status, ok := resp.Deployments[mockOrch.ID()]
		assert.True(t, ok)
		assert.Equal(t, jobtypes.DeploymentStatusPreparing.String(), status)
	})
}

func TestHandleDeploymentStatus(t *testing.T) {
	t.Parallel()

	// Add behavior to test
	node, sActor, _ := newMockNodeWithSender(t, behaviors.DeploymentStatusBehavior)

	// seed deployment data
	eCfg := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{},
			Nodes:       map[string]jobtypes.NodeConfig{},
			Supervisor:  jobtypes.SupervisorConfig{},
			Subnet:      jobtypes.SubnetConfig{},
		},
	}
	mockOrchPrep, err := node.createOrchestrator(context.Background(), eCfg)
	require.NoError(t, err)
	require.NotNil(t, mockOrchPrep)
	err = mockOrchPrep.Deploy(time.Now().Add(2 * time.Minute))
	require.NoError(t, err)

	mockOrchRun, err := node.createOrchestrator(context.Background(), eCfg)
	require.NoError(t, err)
	require.NotNil(t, mockOrchRun)
	err = mockOrchRun.Deploy(time.Now().Add(2 * time.Minute))
	require.NoError(t, err)
	mockOrchRun.(*orchestrator.MockOrchestrator).SetStatus(jobtypes.DeploymentStatusRunning)

	t.Run("non existent", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentStatusBehavior,
			DeploymentStatusRequest{
				ID: "non-existent-id",
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DeploymentStatusResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp.Error, orchestrator.ErrOrchestratorNotFound.Error())
		assert.Empty(t, resp.Status)
	})

	t.Run("existing deployments", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentStatusBehavior,
			DeploymentStatusRequest{
				ID: mockOrchPrep.ID(), // orchestrator in preparing status
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DeploymentStatusResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Empty(t, resp.Error)
		assert.Equal(t, jobtypes.DeploymentStatusPreparing.String(), resp.Status)

		// query running deployment
		msg, err = actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentStatusBehavior,
			DeploymentStatusRequest{
				ID: mockOrchRun.ID(), // orchestrator in preparing status
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err = sActor.Invoke(msg)
		assert.NoError(t, err)

		reply = <-replyChan
		defer reply.Discard()

		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Empty(t, resp.Error)
		assert.Equal(t, jobtypes.DeploymentStatusRunning.String(), resp.Status)
	})
}

func TestHandleDeploymentManifest(t *testing.T) {
	t.Parallel()

	// Add behavior to test
	node, sActor, _ := newMockNodeWithSender(t, behaviors.DeploymentManifestBehavior)

	// seed deployment data
	eCfg := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{},
			Nodes:       map[string]jobtypes.NodeConfig{},
			Supervisor:  jobtypes.SupervisorConfig{},
			Subnet:      jobtypes.SubnetConfig{},
		},
	}
	mockOrch, err := node.createOrchestrator(context.Background(), eCfg)
	require.NoError(t, err)
	require.NotNil(t, mockOrch)
	err = mockOrch.Deploy(time.Now().Add(2 * time.Minute))
	require.NoError(t, err)

	t.Run("non existent", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentManifestBehavior,
			DeploymentManifestRequest{
				ID: "non-existent-id",
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DeploymentManifestResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp.Error, orchestrator.ErrOrchestratorNotFound.Error())
		assert.Empty(t, resp.Manifest.ID)
	})

	t.Run("existing deployments", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentManifestBehavior,
			DeploymentManifestRequest{
				ID: mockOrch.ID(),
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DeploymentManifestResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Empty(t, resp.Error)
		assert.Contains(t, resp.Manifest.ID, mockOrch.ID())
	})
}

func TestHandleDeploymentShutdown(t *testing.T) {
	t.Parallel()

	// Add behavior to test
	node, sActor, _ := newMockNodeWithSender(t, behaviors.DeploymentShutdownBehavior)

	// seed deployment data
	eCfg := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{},
			Nodes:       map[string]jobtypes.NodeConfig{},
			Supervisor:  jobtypes.SupervisorConfig{},
			Subnet:      jobtypes.SubnetConfig{},
		},
	}
	mockOrch, err := node.createOrchestrator(context.Background(), eCfg)
	require.NoError(t, err)
	require.NotNil(t, mockOrch)
	err = mockOrch.Deploy(time.Now().Add(2 * time.Minute))
	require.NoError(t, err)

	t.Run("non existent", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentShutdownBehavior,
			DeploymentShutdownRequest{
				ID: "non-existent-id",
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DeploymentShutdownResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, orchestrator.ErrOrchestratorNotFound.Error())
	})

	t.Run("not running deployment", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentShutdownBehavior,
			DeploymentShutdownRequest{
				ID: mockOrch.ID(),
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DeploymentShutdownResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, ErrorDeploymentNotRunning.Error())
	})

	t.Run("successful shutdown", func(t *testing.T) {
		t.Parallel()
		mockOrchRunning, err := node.createOrchestrator(context.Background(), eCfg)
		require.NoError(t, err)
		require.NotNil(t, mockOrchRunning)
		err = mockOrchRunning.Deploy(time.Now().Add(2 * time.Minute))
		require.NoError(t, err)
		// set to running status
		mockOrchRunning.(*orchestrator.MockOrchestrator).SetStatus(jobtypes.DeploymentStatusRunning)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentShutdownBehavior,
			DeploymentShutdownRequest{
				ID: mockOrchRunning.ID(),
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DeploymentShutdownResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
	})
}

func TestHandleDeploymentRevert(t *testing.T) {
	t.Parallel()
	const allocName = "alloc1"

	// seed deployment data
	eCfg := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{
				"alloc1": {
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 1},
						RAM:  types.RAM{Size: 1},
						Disk: types.Disk{Size: 1},
					},
					Type: jobtypes.AllocationTypeService,
				},
			},
			Nodes:      map[string]jobtypes.NodeConfig{},
			Supervisor: jobtypes.SupervisorConfig{},
			Subnet:     jobtypes.SubnetConfig{},
		},
	}

	t.Run("non existent", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.DeploymentRevertBehavior)

		mockOrch, err := node.createOrchestrator(context.Background(), eCfg)
		require.NoError(t, err)
		require.NotNil(t, mockOrch)
		err = mockOrch.Deploy(time.Now().Add(2 * time.Minute))
		require.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentRevertBehavior,
			orchestrator.DeploymentRevertRequest{
				EnsembleID: "non-existent-id",
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		err = sActor.Send(msg)
		assert.NoError(t, err)
	})

	t.Run("revert a committed only alloc", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		node, sActor, _ := newMockNodeWithSender(t, behaviors.DeploymentRevertBehavior)

		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		mockOrch, err := node.createOrchestrator(context.Background(), eCfg)
		require.NoError(t, err)
		require.NotNil(t, mockOrch)
		err = mockOrch.Deploy(time.Now().Add(2 * time.Minute))
		require.NoError(t, err)

		// create allocs
		resrc := types.Resources{
			CPU:  types.CPU{Cores: 1},
			RAM:  types.RAM{Size: 1},
			Disk: types.Disk{Size: 1},
		}
		require.NoError(t, err)

		err = node.allocator.Commit(
			ctx, allocName, types.CommittedResources{
				Resources:    resrc,
				AllocationID: allocName,
			}, nil, 0, 0,
		)
		require.NoError(t, err)
		assert.Equal(t, 1, len(node.allocator.(*allocator).getCommits()))

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentRevertBehavior,
			orchestrator.DeploymentRevertRequest{
				EnsembleID:   mockOrch.ID(),
				AllocsByName: []string{allocName},
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		node.handleDeploymentRevert(msg)
		time.Sleep(2 * time.Second) // give some time for the revert to process
		assert.Equal(t, 0, len(node.allocator.(*allocator).getCommits()))
	})

	t.Run("terminate an allocation", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		node, sActor, _ := newMockNodeWithSender(t, behaviors.DeploymentRevertBehavior)

		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		mockOrch, err := node.createOrchestrator(context.Background(), eCfg)
		require.NoError(t, err)
		require.NotNil(t, mockOrch)
		err = mockOrch.Deploy(time.Now().Add(2 * time.Minute))
		require.NoError(t, err)

		// create allocs
		resrc := types.Resources{
			CPU:  types.CPU{Cores: 1},
			RAM:  types.RAM{Size: 1},
			Disk: types.Disk{Size: 1},
		}
		nullExecutor, err := null.NewExecutor(ctx, "test-executor")
		require.NoError(t, err)

		err = node.allocator.Commit(
			ctx, allocName, types.CommittedResources{
				Resources:    resrc,
				AllocationID: allocName,
			}, nil, 0, 0,
		)
		require.NoError(t, err)

		alloc, err := node.allocator.Allocate(
			context.Background(),
			allocName,
			jobtypes.AllocationTypeService,
			node.actor,
			sActor.Handle(),
			jobs.Job{
				Resources: resrc,
				Execution: types.SpecConfig{
					Type: "null",
				},
			},
			nullExecutor,
		)
		require.NoError(t, err)
		require.NotNil(t, alloc)

		node.allocator.(*allocator).allocations[types.ConstructAllocationID(mockOrch.ID(), allocName)] = alloc

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentRevertBehavior,
			orchestrator.DeploymentRevertRequest{
				EnsembleID:   mockOrch.ID(),
				AllocsByName: []string{allocName},
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		node.handleDeploymentRevert(msg)
		assert.Equal(t, jobtypes.AllocationTerminated, alloc.Status(ctx).Status)
	})
}

func TestHandleDeploymentUpdate(t *testing.T) {
	t.Parallel()
	const allocName = "alloc1"

	// Add behavior to test
	node, sActor, _ := newMockNodeWithSender(t, behaviors.DeploymentUpdateBehavior)

	// seed deployment data
	eCfg := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{
				allocName: {
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 1},
						RAM:  types.RAM{Size: 1},
						Disk: types.Disk{Size: 1},
					},
					Type: jobtypes.AllocationTypeService,
				},
			},
			Nodes:      map[string]jobtypes.NodeConfig{},
			Supervisor: jobtypes.SupervisorConfig{},
			Subnet:     jobtypes.SubnetConfig{},
		},
	}

	mockOrch, err := node.createOrchestrator(context.Background(), eCfg)
	require.NoError(t, err)
	require.NotNil(t, mockOrch)
	err = mockOrch.Deploy(time.Now().Add(2 * time.Minute))
	require.NoError(t, err)

	t.Run("non existent", func(t *testing.T) {
		t.Parallel()
		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentUpdateBehavior,
			UpdateDeploymentRequest{
				EnsembleID: "non-existent-id",
				Ensemble:   eCfg,
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp UpdateDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, orchestrator.ErrOrchestratorNotFound.Error())
	})

	t.Run("not running deployment", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentUpdateBehavior,
			UpdateDeploymentRequest{
				EnsembleID: mockOrch.ID(),
				Ensemble:   eCfg,
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp UpdateDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, ErrorDeploymentNotRunning.Error())
	})

	t.Run("deployment time too short", func(t *testing.T) {
		t.Parallel()
		mockOrch, err := node.createOrchestrator(context.Background(), eCfg)
		require.NoError(t, err)
		require.NotNil(t, mockOrch)
		err = mockOrch.Deploy(time.Now().Add(2 * time.Minute))
		require.NoError(t, err)
		// set to running status
		mockOrch.(*orchestrator.MockOrchestrator).SetStatus(jobtypes.DeploymentStatusRunning)

		shortExpiryMsg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentUpdateBehavior,
			UpdateDeploymentRequest{
				EnsembleID: mockOrch.ID(),
				Ensemble:   eCfg,
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(1*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(shortExpiryMsg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp UpdateDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, "requested deployment update time too short")
	})

	t.Run("successful update", func(t *testing.T) {
		t.Parallel()
		mockOrch, err := node.createOrchestrator(context.Background(), eCfg)
		require.NoError(t, err)
		require.NotNil(t, mockOrch)
		err = mockOrch.Deploy(time.Now().Add(2 * time.Minute))
		require.NoError(t, err)
		// set to running status
		mockOrch.(*orchestrator.MockOrchestrator).SetStatus(jobtypes.DeploymentStatusRunning)

		updatedCfg := eCfg.Clone()
		updatedCfg.AddNodeAndAllocations(
			"node2",
			jobtypes.NodeConfig{
				Allocations: []string{"alloc2"},
			},
			map[string]jobtypes.AllocationConfig{
				"alloc2": {
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 1},
						RAM:  types.RAM{Size: 1},
						Disk: types.Disk{Size: 1},
					},
					Type: jobtypes.AllocationTypeService,
				},
			},
		)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentUpdateBehavior,
			UpdateDeploymentRequest{
				EnsembleID: mockOrch.ID(),
				Ensemble:   updatedCfg,
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp UpdateDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
	})
}
