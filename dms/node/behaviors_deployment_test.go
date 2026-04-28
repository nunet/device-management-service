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
	"gitlab.com/nunet/device-management-service/tokenomics/eventhandler"
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

	ensembleConfig := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{},
			Nodes:       map[string]jobtypes.NodeConfig{},
			Supervisor:  jobtypes.SupervisorConfig{},
			Subnet:      jobtypes.SubnetConfig{},
			Contracts:   map[string]types.ContractConfig{},
		},
	}

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

		msg, err = actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.NewDeploymentBehavior,
			NewDeploymentRequest{
				Ensemble: jobtypes.EnsembleConfig{},
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

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

	t.Run("without metadata filtering expect all", func(t *testing.T) {
		t.Parallel()

		// Add behavior to test
		node, sActor, _ := newMockNodeWithOrchestratorRegistryAndSender(t, behaviors.DeploymentListBehavior)

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
		mockOrch, err := node.createOrchestrator(context.Background(), eCfgWithMetadata, nil)
		require.NoError(t, err)
		require.NotNil(t, mockOrch)

		eCfgWithoutMetadata := jobtypes.EnsembleConfig{
			V1: &jobtypes.EnsembleConfigV1{
				Allocations: map[string]jobtypes.AllocationConfig{},
				Nodes:       map[string]jobtypes.NodeConfig{},
				Supervisor:  jobtypes.SupervisorConfig{},
				Subnet:      jobtypes.SubnetConfig{},
			},
		}
		mockOrchWithoutM, err := node.createOrchestrator(context.Background(), eCfgWithoutMetadata, nil)
		require.NoError(t, err)
		require.NotNil(t, mockOrchWithoutM)

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

		// Check that both deployments are in the response
		require.Equal(t, 2, len(resp.Deployments))
		assert.Contains(t, []string{resp.Deployments[0].OrchestratorID, resp.Deployments[1].OrchestratorID}, mockOrch.ID())
		assert.Contains(t, []string{resp.Deployments[0].OrchestratorID, resp.Deployments[1].OrchestratorID}, mockOrchWithoutM.ID())
		assert.Equal(t, jobtypes.DeploymentStatusPreparing.String(), resp.Deployments[0].Status)
		assert.Equal(t, jobtypes.DeploymentStatusPreparing.String(), resp.Deployments[1].Status)
	})

	t.Run("with metadata", func(t *testing.T) {
		t.Parallel()

		// Add behavior to test
		node, sActor, _ := newMockNodeWithOrchestratorRegistryAndSender(t, behaviors.DeploymentListBehavior)

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
		mockOrch, err := node.createOrchestrator(context.Background(), eCfgWithMetadata, nil)
		require.NoError(t, err)
		require.NotNil(t, mockOrch)

		eCfgWithoutMetadata := jobtypes.EnsembleConfig{
			V1: &jobtypes.EnsembleConfigV1{
				Allocations: map[string]jobtypes.AllocationConfig{},
				Nodes:       map[string]jobtypes.NodeConfig{},
				Supervisor:  jobtypes.SupervisorConfig{},
				Subnet:      jobtypes.SubnetConfig{},
			},
		}
		mockOrchWithoutM, err := node.createOrchestrator(context.Background(), eCfgWithoutMetadata, nil)
		require.NoError(t, err)
		require.NotNil(t, mockOrchWithoutM)

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
		assert.Equal(t, mockOrch.ID(), resp.Deployments[0].OrchestratorID)
		assert.Equal(t, jobtypes.DeploymentStatusPreparing.String(), resp.Deployments[0].Status)
	})

	t.Run("with pagination", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithOrchestratorRegistryAndSender(t, behaviors.DeploymentListBehavior)

		// Create multiple deployments
		for i := 0; i < 5; i++ {
			eCfg := jobtypes.EnsembleConfig{
				V1: &jobtypes.EnsembleConfigV1{
					Allocations: map[string]jobtypes.AllocationConfig{},
					Nodes:       map[string]jobtypes.NodeConfig{},
					Supervisor:  jobtypes.SupervisorConfig{},
					Subnet:      jobtypes.SubnetConfig{},
				},
			}

			_, err := node.createOrchestrator(context.Background(), eCfg, nil)
			require.NoError(t, err)
		}

		// Test first page
		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentListBehavior,
			DeploymentListRequest{
				Limit:  2,
				Offset: 0,
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
		assert.Equal(t, 2, len(resp.Deployments))
		assert.GreaterOrEqual(t, resp.Total, 5)
		assert.True(t, resp.HasMore)

		// Test second page
		msg2, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentListBehavior,
			DeploymentListRequest{
				Limit:  2,
				Offset: 2,
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan2, err := sActor.Invoke(msg2)
		assert.NoError(t, err)

		reply2 := <-replyChan2
		defer reply2.Discard()

		var resp2 DeploymentListResponse
		err = json.Unmarshal(reply2.Message, &resp2)
		assert.NoError(t, err)
		assert.Equal(t, 2, len(resp2.Deployments))
		// Ensure different deployments
		assert.NotEqual(t, resp.Deployments[0].OrchestratorID, resp2.Deployments[0].OrchestratorID)
	})

	t.Run("with status filter", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithOrchestratorRegistryAndSender(t, behaviors.DeploymentListBehavior)

		// Create deployments with different statuses
		eCfg := jobtypes.EnsembleConfig{
			V1: &jobtypes.EnsembleConfigV1{
				Allocations: map[string]jobtypes.AllocationConfig{},
				Nodes:       map[string]jobtypes.NodeConfig{},
				Supervisor:  jobtypes.SupervisorConfig{},
				Subnet:      jobtypes.SubnetConfig{},
			},
		}
		_, err := node.createOrchestrator(context.Background(), eCfg, nil)
		require.NoError(t, err)

		// expect only "Preparing" status deployments
		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentListBehavior,
			DeploymentListRequest{
				Status: []jobtypes.DeploymentStatus{jobtypes.DeploymentStatusPreparing},
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

		// there should be at least one in "Preparing" status
		assert.Equal(t, 1, len(resp.Deployments))
		assert.Equal(t, jobtypes.DeploymentStatusPreparing.String(), resp.Deployments[0].Status)

		// expect only "Completed" status deployments
		msg, err = actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentListBehavior,
			DeploymentListRequest{
				Status: []jobtypes.DeploymentStatus{jobtypes.DeploymentStatusCompleted},
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

		// not expecting any in "Completed" status
		assert.Equal(t, 0, len(resp.Deployments))
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
	mockOrchPrep, err := node.createOrchestrator(context.Background(), eCfg, nil)
	require.NoError(t, err)
	require.NotNil(t, mockOrchPrep)
	err = mockOrchPrep.Deploy(time.Now().Add(2 * time.Minute))
	require.NoError(t, err)

	mockOrchRun, err := node.createOrchestrator(context.Background(), eCfg, nil)
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
	mockOrch, err := node.createOrchestrator(context.Background(), eCfg, nil)
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

func TestHandleDeploymentInfo(t *testing.T) {
	t.Parallel()

	// Add behavior to test
	node, sActor, _ := newMockNodeWithSender(t, behaviors.DeploymentInfoBehavior)

	// seed deployment data
	eCfg := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{},
			Nodes:       map[string]jobtypes.NodeConfig{},
			Supervisor:  jobtypes.SupervisorConfig{},
			Subnet:      jobtypes.SubnetConfig{},
		},
	}
	mockOrch, err := node.createOrchestrator(context.Background(), eCfg, nil)
	require.NoError(t, err)
	require.NotNil(t, mockOrch)
	err = mockOrch.Deploy(time.Now().Add(2 * time.Minute))
	require.NoError(t, err)

	t.Run("non existent", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentInfoBehavior,
			DeploymentInfoRequest{
				ID: "non-existent-id",
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DeploymentInfoResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp.Error, orchestrator.ErrOrchestratorNotFound.Error())
		assert.Empty(t, resp.ID)
	})

	t.Run("existing deployment basic info", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentInfoBehavior,
			DeploymentInfoRequest{
				ID: mockOrch.ID(),
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DeploymentInfoResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Empty(t, resp.Error)
		assert.Equal(t, mockOrch.ID(), resp.ID)
		assert.NotNil(t, resp.Manifest)
		assert.Contains(t, resp.Manifest.ID, mockOrch.ID())
	})

	t.Run("empty ID", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentInfoBehavior,
			DeploymentInfoRequest{
				ID: "",
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp DeploymentInfoResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp.Error, "deployment ID is required")
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
	mockOrch, err := node.createOrchestrator(context.Background(), eCfg, nil)
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
		mockOrchRunning, err := node.createOrchestrator(context.Background(), eCfg, nil)
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

		mockOrch, err := node.createOrchestrator(context.Background(), eCfg, nil)
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

		mockOrch, err := node.createOrchestrator(context.Background(), eCfg, nil)
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
		msg.Options.ReplyTo = fmt.Sprintf("/dms/actor/replyto/%d", 1)

		node.handleDeploymentRevert(msg)
		time.Sleep(2 * time.Second) // give some time for the revert to process
		assert.Equal(t, 0, len(node.allocator.(*allocator).getCommits()))
	})

	t.Run("terminate an allocation", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		node, sActor, _ := newMockNodeWithSender(t, behaviors.DeploymentRevertBehavior)

		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		mockOrch, err := node.createOrchestrator(context.Background(), eCfg, nil)
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
			map[string]types.ContractConfig{},
			eventhandler.New(context.Background(), 1, 1, time.Second, time.Second, func(_ eventhandler.Event) error { return nil }),
			"deployment-id",
			func(_ string, _ jobtypes.AllocationStatus) {},
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
		assert.Equal(t, jobtypes.AllocationTerminated, alloc.Status().Status)
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

	mockOrch, err := node.createOrchestrator(context.Background(), eCfg, nil)
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
		mockOrch, err := node.createOrchestrator(context.Background(), eCfg, nil)
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
		mockOrch, err := node.createOrchestrator(context.Background(), eCfg, nil)
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

// Prune tests using real orchestrators and registry
func TestHandleDeploymentPrune_Before_RFC3339(t *testing.T) {
	t.Parallel()
	node, sActor, _ := newMockNodeWithOrchestratorRegistryAndSender(t, behaviors.DeploymentPruneBehavior)

	// Create three deployments with spacing to get different CreatedAt
	eCfg := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{},
			Nodes:       map[string]jobtypes.NodeConfig{},
			Supervisor:  jobtypes.SupervisorConfig{},
			Subnet:      jobtypes.SubnetConfig{},
		},
	}

	o1, err := node.createOrchestrator(context.Background(), eCfg, nil)
	require.NoError(t, err)
	require.NotNil(t, o1)
	o1.(*orchestrator.BasicOrchestrator).SetStatus(jobtypes.DeploymentStatusCompleted)
	err = node.saveDeployment(o1)
	require.NoError(t, err)

	o2, err := node.createOrchestrator(context.Background(), eCfg, nil)
	require.NoError(t, err)
	require.NotNil(t, o2)
	o2.(*orchestrator.BasicOrchestrator).SetStatus(jobtypes.DeploymentStatusFailed)
	err = node.saveDeployment(o2)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)
	o3, err := node.createOrchestrator(context.Background(), eCfg, nil)
	require.NoError(t, err)
	require.NotNil(t, o3)
	o3.(*orchestrator.BasicOrchestrator).SetStatus(jobtypes.DeploymentStatusCompleted)
	err = node.saveDeployment(o3)
	require.NoError(t, err)

	dep, err := node.orchestratorRegistry.GetAllDeployments()
	require.NoError(t, err)
	require.NotNil(t, dep)
	var cutoff time.Time
	for _, d := range dep {
		if d.OrchestratorID == o3.ID() {
			cutoff = d.CreatedAt.Add(-1 * time.Second)
		}
	}

	msg, err := actor.Message(
		sActor.Handle(),
		node.actor.Handle(),
		behaviors.DeploymentPruneBehavior,
		DeploymentPruneRequest{Before: cutoff.Format(time.RFC3339)},
	)
	require.NoError(t, err)
	replyCh, err := sActor.Invoke(msg)
	require.NoError(t, err)
	reply := <-replyCh
	defer reply.Discard()

	var resp DeploymentPruneResponse
	require.NoError(t, json.Unmarshal(reply.Message, &resp))
	require.True(t, resp.OK)

	remaining, err := node.orchestratorRegistry.GetAllDeployments()
	require.NoError(t, err)
	ids := map[string]bool{}
	for _, v := range remaining {
		ids[v.OrchestratorID] = true
	}
	assert.False(t, ids[o1.ID()])
	assert.False(t, ids[o2.ID()])
	assert.True(t, ids[o3.ID()])
}

func TestHandleDeploymentPrune_Before_Durations(t *testing.T) {
	t.Parallel()
	node, sActor, _ := newMockNodeWithOrchestratorRegistryAndSender(t, behaviors.DeploymentPruneBehavior)

	eCfg := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{},
			Nodes:       map[string]jobtypes.NodeConfig{},
			Supervisor:  jobtypes.SupervisorConfig{},
			Subnet:      jobtypes.SubnetConfig{},
		},
	}

	// Create older
	old, err := node.createOrchestrator(context.Background(), eCfg, nil)
	require.NoError(t, err)
	old.(*orchestrator.BasicOrchestrator).SetStatus(jobtypes.DeploymentStatusCompleted)
	err = node.saveDeployment(old)
	require.NoError(t, err)

	// some delay between two deployments
	time.Sleep(10 * time.Second)

	newer, err := node.createOrchestrator(context.Background(), eCfg, nil)
	require.NoError(t, err)
	newer.(*orchestrator.BasicOrchestrator).SetStatus(jobtypes.DeploymentStatusCompleted)
	err = node.saveDeployment(newer)
	require.NoError(t, err)

	// should remove old, keep newer - old is at least 10s old
	msg, err := actor.Message(
		sActor.Handle(),
		node.actor.Handle(),
		behaviors.DeploymentPruneBehavior,
		DeploymentPruneRequest{Before: "6s"},
	)
	require.NoError(t, err)
	replyCh, err := sActor.Invoke(msg)
	require.NoError(t, err)
	<-replyCh

	rem, err := node.orchestratorRegistry.GetAllDeployments()
	require.NoError(t, err)
	ids := map[string]bool{}
	for _, v := range rem {
		ids[v.OrchestratorID] = true
	}
	assert.False(t, ids[old.ID()])
	assert.True(t, ids[newer.ID()])

	// 1m, 1h, 1d should not delete newer (and old is already gone)
	for _, dur := range []string{"1m", "1h", "1d"} {
		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.DeploymentPruneBehavior,
			DeploymentPruneRequest{Before: dur},
		)
		require.NoError(t, err)
		replyCh, err := sActor.Invoke(msg)
		require.NoError(t, err)
		<-replyCh
		rem, err = node.orchestratorRegistry.GetAllDeployments()
		require.NoError(t, err)

		require.NotEmpty(t, rem)
	}
}

func TestHandleDeploymentPrune_All(t *testing.T) {
	t.Parallel()
	node, sActor, _ := newMockNodeWithOrchestratorRegistryAndSender(t, behaviors.DeploymentPruneBehavior)

	eCfg := jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: map[string]jobtypes.AllocationConfig{},
			Nodes:       map[string]jobtypes.NodeConfig{},
			Supervisor:  jobtypes.SupervisorConfig{},
			Subnet:      jobtypes.SubnetConfig{},
		},
	}

	prep, err := node.createOrchestrator(context.Background(), eCfg, nil)
	require.NoError(t, err)

	run, err := node.createOrchestrator(context.Background(), eCfg, nil)
	require.NoError(t, err)
	run.(*orchestrator.BasicOrchestrator).SetStatus(jobtypes.DeploymentStatusRunning)
	err = node.saveDeployment(run)
	require.NoError(t, err)

	fail, _ := node.createOrchestrator(context.Background(), eCfg, nil)
	fail.(*orchestrator.BasicOrchestrator).SetStatus(jobtypes.DeploymentStatusFailed)
	err = node.saveDeployment(fail)
	require.NoError(t, err)

	comp, _ := node.createOrchestrator(context.Background(), eCfg, nil)
	comp.(*orchestrator.BasicOrchestrator).SetStatus(jobtypes.DeploymentStatusCompleted)

	err = node.saveDeployment(comp)
	require.NoError(t, err)

	msg, err := actor.Message(
		sActor.Handle(),
		node.actor.Handle(),
		behaviors.DeploymentPruneBehavior,
		DeploymentPruneRequest{All: true},
	)
	require.NoError(t, err)
	replyCh, err := sActor.Invoke(msg)
	require.NoError(t, err)
	<-replyCh

	rem, err := node.orchestratorRegistry.GetAllDeployments()
	require.NoError(t, err)
	ids := map[string]bool{}
	for _, v := range rem {
		ids[v.OrchestratorID] = true
	}
	assert.True(t, ids[prep.ID()])
	assert.True(t, ids[run.ID()])
	assert.False(t, ids[fail.ID()])
	assert.False(t, ids[comp.ID()])
}
