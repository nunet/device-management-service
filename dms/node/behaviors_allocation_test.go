package node

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

func TestHandleSubnetCreate(t *testing.T) {
	observability.SetNoOpMode(true)

	t.Parallel()

	const ensembleID = "test-subnet-ensemble"
	subnetCreateBehavior := fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID)

	t.Run("wrong request message format", func(t *testing.T) {
		t.Parallel()
		node, sActor, _ := newMockNodeWithSender(t, subnetCreateBehavior)

		err := node.addEnsembleBehaviors(ensembleID)
		assert.NoError(t, err)

		// create message
		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			subnetCreateBehavior,
			"wrongMessage",
			actor.WithMessageReplyTo(subnetCreateBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.SubnetCreateResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, "error unmarshalling subnet create request")
	})

	t.Run("successful create", func(t *testing.T) {
		t.Parallel()

		node, sActor, sNet := newMockNodeWithSender(t, subnetCreateBehavior)

		err := node.addEnsembleBehaviors(ensembleID)
		assert.NoError(t, err)

		// create message
		subnetID := "test-subnet-2"
		routingTable := map[string]string{"192.168.1.1": sNet.GetHostID().String()}

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			subnetCreateBehavior,
			orchestrator.SubnetCreateRequest{
				SubnetID:     subnetID,
				RoutingTable: routingTable,
			},
			actor.WithMessageReplyTo(subnetCreateBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.SubnetCreateResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
	})
}

func TestHandleSubnetDestroy(t *testing.T) {
	t.Parallel()
	const ensembleID = "test-subnet-ensemble"
	subnetDestroyBehavior := fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)

	const subnetID = "test-subnet"

	t.Run("wrong request message format", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, subnetDestroyBehavior)

		// add behavior to test
		err := node.addEnsembleBehaviors(ensembleID)
		assert.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			subnetDestroyBehavior,
			"wrong message",
			actor.WithMessageReplyTo(subnetDestroyBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.SubnetDestroyResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, "error unmarshalling subnet destroy")
	})

	t.Run("successful destroy", func(t *testing.T) {
		t.Parallel()

		node, sActor, nVnet := newMockNodeWithSender(t, subnetDestroyBehavior)

		// add behavior to test
		err := node.addEnsembleBehaviors(ensembleID)
		assert.NoError(t, err)

		// create subnet first
		err = node.network.CreateSubnet(
			context.Background(),
			subnetID,
			map[string]string{"192.168.1.1": nVnet.GetHostID().String()})
		require.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			subnetDestroyBehavior,
			orchestrator.SubnetDestroyRequest{
				SubnetID: subnetID,
			},
			actor.WithMessageReplyTo(subnetDestroyBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.SubnetDestroyResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
	})
}

func TestHandleSubnetJoin(t *testing.T) {
	t.Parallel()

	const subnetID = "test-subnet"
	ensembleID := "test-subnet-ensemble"
	subnetJoinBehavior := fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID)

	t.Run("wrong request message format", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, subnetJoinBehavior)

		err := node.actor.AddBehavior(subnetJoinBehavior, node.handleSubnetJoin)
		require.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			subnetJoinBehavior,
			"wrong message",
			actor.WithMessageReplyTo(subnetJoinBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.SubnetJoinResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, "error unmarshalling subnet join")
	})

	t.Run("successful join", func(t *testing.T) {
		t.Parallel()

		node, sActor, sVnet := newMockNodeWithSender(t, subnetJoinBehavior)

		err := node.actor.AddBehavior(subnetJoinBehavior, node.handleSubnetJoin)
		require.NoError(t, err)

		// create subnet first
		err = node.network.CreateSubnet(
			context.Background(),
			subnetID,
			map[string]string{"192.168.1.1": sVnet.GetHostID().String()})
		require.NoError(t, err)

		// create message
		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			subnetJoinBehavior,
			orchestrator.SubnetJoinRequest{
				SubnetID: subnetID,
				IP:       "192.168.1.2",
				PeerID:   sVnet.GetHostID().String(),
				Records:  map[string]string{"name1": "192.168.1.2"},
			},
			actor.WithMessageReplyTo(subnetJoinBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.SubnetCreateResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
	})
}

func TestCreateAllocation(t *testing.T) {
	t.Parallel()

	observability.SetNoOpMode(true)

	ctx := context.Background()

	resrc := types.Resources{
		CPU:  types.CPU{Cores: 1},
		RAM:  types.RAM{Size: 1},
		Disk: types.Disk{Size: 1},
	}

	_, orchPub, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	require.NoError(t, err)
	orchHandle, err := actor.HandleFromDID(did.FromPublicKey(orchPub).String())
	require.NoError(t, err)

	t.Run("create allocation with unsupported executor", func(t *testing.T) {
		t.Parallel()

		// only docker executor is supported for now - testing with null and firecracker
		substrate := network.NewSubstrate()
		node, _, _ := newMockNode(t, substrate)
		const allocationName = "test-allocation-1"
		_, err = node.createAllocation(
			allocationName,
			jobtypes.AllocationType("task"),
			jobs.Job{
				Resources: resrc,
				Execution: types.SpecConfig{
					Type: "null",
				},
			},
			orchHandle,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported executor type: null")

		_, err = node.createAllocation(
			allocationName,
			jobtypes.AllocationType("task"),
			jobs.Job{
				Resources: resrc,
				Execution: types.SpecConfig{
					Type: "firecracker",
				},
			},
			orchHandle,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported executor type: firecracker")
	})

	t.Run("uncommitted allocation", func(t *testing.T) {
		t.Parallel()

		// assumes docker is installed and running
		substrate := network.NewSubstrate()
		node, _, _ := newMockNode(t, substrate)

		_, err := docker.NewExecutor(ctx, node.fs, "test-executor-installed")
		if errors.Is(err, docker.ErrNotInstalled) {
			t.Skip("skipping test that requires docker installed")
		}

		alloc, err := node.createAllocation(
			"test-allocation-2",
			jobtypes.AllocationType("task"),
			jobs.Job{
				Resources: resrc,
				Execution: types.SpecConfig{
					Type: "docker",
				},
			},
			orchHandle,
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "allocation not committed")
		assert.Nil(t, alloc)
	})

	t.Run("allocation success", func(t *testing.T) {
		t.Parallel()
		// assumes docker is installed and running
		substrate := network.NewSubstrate()
		node, _, _ := newMockNode(t, substrate)
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		_, err := docker.NewExecutor(ctx, node.fs, "test-executor-installed-2")
		if errors.Is(err, docker.ErrNotInstalled) {
			t.Skip("skipping test that requires docker installed")
		}

		allocationID := "test_allocaction_third"

		err = node.allocator.Commit(
			ctx, allocationID,
			types.CommittedResources{
				AllocationID: allocationID,
				Resources:    resrc,
			},
			nil, 0, 0)
		require.NoError(t, err)

		alloc, err := node.createAllocation(
			allocationID,
			jobtypes.AllocationType("task"),
			jobs.Job{
				Resources: resrc,
				Execution: types.SpecConfig{
					Type: "docker",
				},
			},
			orchHandle,
		)
		assert.NoError(t, err)
		assert.NotNil(t, alloc)
		err = alloc.Terminate(ctx)
		assert.NoError(t, err)
	})
}

func TestCreateAllocations(t *testing.T) {
	t.Parallel()

	observability.SetNoOpMode(true)
	ctx := context.Background()

	ensembleID := "test-ensemble-id"
	_, orchPub, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	require.NoError(t, err)
	supervisorHandle, err := actor.HandleFromDID(did.FromPublicKey(orchPub).String())

	require.NoError(t, err)
	require.NoError(t, err)

	resrc := types.Resources{
		CPU:  types.CPU{Cores: 1},
		RAM:  types.RAM{Size: 1},
		Disk: types.Disk{Size: 1},
	}

	t.Run("empty allocations", func(t *testing.T) {
		t.Parallel()

		substrate := network.NewSubstrate()
		node, _, _ := newMockNode(t, substrate)

		allocHandles, err := node.createAllocations(ensembleID, map[string]jobtypes.AllocationDeploymentConfig{}, supervisorHandle)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no allocations to create")
		assert.Empty(t, allocHandles)
	})

	t.Run("no supervisor DID", func(t *testing.T) {
		t.Parallel()

		substrate := network.NewSubstrate()
		node, _, _ := newMockNode(t, substrate)

		allocations := map[string]jobtypes.AllocationDeploymentConfig{
			"alloc1": {
				Type:      "task",
				Resources: resrc,
				Execution: types.SpecConfig{
					Type: "unsupported-executor",
				},
			},
		}
		allocHandles, err := node.createAllocations(
			ensembleID,
			allocations,
			actor.Handle{
				ID:      supervisorHandle.ID,
				Address: supervisorHandle.Address,
			},
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid supervisor handle")
		assert.Empty(t, allocHandles)
	})

	t.Run("allocation creation failure", func(t *testing.T) {
		t.Parallel()

		substrate := network.NewSubstrate()
		node, _, _ := newMockNode(t, substrate)

		allocations := map[string]jobtypes.AllocationDeploymentConfig{
			"alloc1": {
				Type:      "task",
				Resources: resrc,
				Execution: types.SpecConfig{
					Type: "unsupported-executor",
				},
			},
		}

		allocHandles, err := node.createAllocations(ensembleID, allocations, supervisorHandle)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported executor type")
		assert.Nil(t, allocHandles)
	})

	t.Run("successful allocations creation", func(t *testing.T) {
		t.Parallel()

		substrate := network.NewSubstrate()
		node, _, _ := newMockNode(t, substrate)

		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		// assumes docker is installed and running
		_, err := docker.NewExecutor(ctx, node.fs, "test-executor-installed")
		if errors.Is(err, docker.ErrNotInstalled) {
			t.Skip("skipping test that requires docker installed")
		}

		allocationName := "test-allocation-1"
		allocationID := types.ConstructAllocationID(ensembleID, allocationName)
		err = node.allocator.Commit(
			ctx, allocationID,
			types.CommittedResources{
				AllocationID: allocationID,
				Resources:    resrc,
			},
			nil, 0, 0)
		require.NoError(t, err)

		allocations := map[string]jobtypes.AllocationDeploymentConfig{
			allocationName: {
				Type:      "task",
				Resources: resrc,
				Execution: types.SpecConfig{
					Type: "docker",
				},
			},
		}

		allocHandles, err := node.createAllocations(ensembleID, allocations, supervisorHandle)
		assert.NoError(t, err)
		assert.NotNil(t, allocHandles)
		assert.Len(t, allocHandles, 1)
		assert.Contains(t, allocHandles, allocationName)
	})
}

func TestHandleAllocationDeployment(t *testing.T) {
	t.Parallel()

	observability.SetNoOpMode(true)

	ensembleID := "test-ensemble"

	t.Run("invalid request message format", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.AllocationDeploymentBehavior)
		node.ctx, node.cancel = context.WithCancel(context.Background())

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.AllocationDeploymentBehavior,
			"invalidMessage",
			actor.WithMessageReplyTo(behaviors.AllocationDeploymentBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		assert.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp jobtypes.AllocationDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, "cannot unmarshal string into Go value of type jobtypes.AllocationDeploymentRequest")
	})

	t.Run("no allocations", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.AllocationDeploymentBehavior)
		node.ctx, node.cancel = context.WithCancel(context.Background())

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.AllocationDeploymentBehavior,
			jobtypes.AllocationDeploymentRequest{
				EnsembleID: "ensemble-id",
			},
			actor.WithMessageReplyTo(behaviors.AllocationDeploymentBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		assert.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp jobtypes.AllocationDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, "no allocations to create for ensembleID")
	})

	t.Run("failed to create allocations", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.AllocationDeploymentBehavior)
		node.ctx, node.cancel = context.WithCancel(context.Background())

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.AllocationDeploymentBehavior,
			jobtypes.AllocationDeploymentRequest{
				EnsembleID: ensembleID,
				Allocations: map[string]jobtypes.AllocationDeploymentConfig{
					"alloc1": {
						Type: "task",
						Execution: types.SpecConfig{
							Type: "unsupported-executor",
						},
					},
				},
			},
			actor.WithMessageReplyTo(behaviors.AllocationDeploymentBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		assert.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp jobtypes.AllocationDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, "unsupported executor type")
	})

	t.Run("successful allocation deployment", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.AllocationDeploymentBehavior)

		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		_, err := docker.NewExecutor(context.Background(), node.fs, "test-executor-installed")
		if errors.Is(err, docker.ErrNotInstalled) {
			t.Skip("skipping test that requires Docker installed")
		}

		allocationName := "test-allocation-2"
		allocationID := types.ConstructAllocationID(ensembleID, allocationName)
		resrc := types.Resources{
			CPU:  types.CPU{Cores: 1},
			RAM:  types.RAM{Size: 1},
			Disk: types.Disk{Size: 1},
		}

		err = node.allocator.Commit(
			context.Background(),
			allocationID,
			types.CommittedResources{
				AllocationID: allocationID,
				Resources:    resrc,
			},
			nil, 0, 0,
		)
		require.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.AllocationDeploymentBehavior,
			jobtypes.AllocationDeploymentRequest{
				EnsembleID: ensembleID,
				Allocations: map[string]jobtypes.AllocationDeploymentConfig{
					allocationName: {
						Type:      "task",
						Resources: resrc,
						Execution: types.SpecConfig{
							Type: "docker",
						},
					},
				},
			},
			actor.WithMessageReplyTo(behaviors.AllocationDeploymentBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		assert.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp jobtypes.AllocationDeploymentResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.NotNil(t, resp.Allocations)
		assert.Contains(t, resp.Allocations, allocationName)
	})
}

func TestHandleAllocationShutdown(t *testing.T) {
	t.Parallel()

	observability.SetNoOpMode(true)
	ensembleID := "test-ensemble-id"
	allocationShutdownBehavior := fmt.Sprintf(behaviors.AllocationShutdownBehavior.DynamicTemplate, ensembleID)

	t.Run("invalid request message format", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, allocationShutdownBehavior)

		err := node.addEnsembleBehaviors(ensembleID)
		require.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			allocationShutdownBehavior,
			"invalidMessage",
			actor.WithMessageReplyTo(allocationShutdownBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		assert.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp AllocationShutdownResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, "error unmarshalling allocation shutdown request")
	})

	// XXX: Release() doesn't error if the allocation doesn't exist
	//      intentional for now but might need to be revisited
	t.Run("shutdown non-existent allocation", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, allocationShutdownBehavior)

		err := node.addEnsembleBehaviors(ensembleID)
		require.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			allocationShutdownBehavior,
			AllocationShutdownRequest{
				AllocationID: "non-existent-allocation",
			},
			actor.WithMessageReplyTo(allocationShutdownBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		assert.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp AllocationShutdownResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
	})

	t.Run("successful shutdown", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, allocationShutdownBehavior)
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		err := node.addEnsembleBehaviors(ensembleID)
		require.NoError(t, err)

		// Create a mock allocation
		allocationID := "test-alloc"
		_, orchPub, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
		require.NoError(t, err)
		supervisorHandle, err := actor.HandleFromDID(did.FromPublicKey(orchPub).String())
		require.NoError(t, err)

		resrc := types.Resources{
			CPU:  types.CPU{Cores: 1},
			RAM:  types.RAM{Size: 1},
			Disk: types.Disk{Size: 1},
		}

		err = node.allocator.Commit(
			context.Background(),
			allocationID,
			types.CommittedResources{
				AllocationID: allocationID,
				Resources:    resrc,
			},
			nil, 0, 0,
		)
		require.NoError(t, err)

		alloc, err := node.createAllocation(
			allocationID,
			jobtypes.AllocationType("task"),
			jobs.Job{
				Resources: resrc,
				Execution: types.SpecConfig{
					Type: "docker",
				},
			},
			supervisorHandle,
		)
		assert.NoError(t, err)
		assert.NotNil(t, alloc)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			allocationShutdownBehavior,
			AllocationShutdownRequest{
				AllocationID: allocationID,
			},
			actor.WithMessageReplyTo(allocationShutdownBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		assert.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp AllocationShutdownResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
	})
}

func TestHandleAllocationsList(t *testing.T) {
	t.Parallel()

	observability.SetNoOpMode(true)

	ctx := context.Background()

	t.Run("no allocations", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.AllocationsListBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.AllocationsListBehavior,
			nil,
			actor.WithMessageReplyTo(behaviors.AllocationsListBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		assert.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp AllocationsListResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Empty(t, resp.Allocations)
		assert.Empty(t, resp.Error)
	})

	t.Run("with allocations", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.AllocationsListBehavior)

		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		_, orchPub, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
		require.NoError(t, err)
		supervisorHandle, err := actor.HandleFromDID(did.FromPublicKey(orchPub).String())
		require.NoError(t, err)

		allocationID1 := "test-allocation-1"
		allocationID2 := "test-allocation-2"
		resrc := types.Resources{
			CPU:  types.CPU{Cores: 1},
			RAM:  types.RAM{Size: 1},
			Disk: types.Disk{Size: 1},
		}

		job := jobs.Job{
			Resources: resrc,
			Execution: types.SpecConfig{
				Type: "docker",
			},
		}

		err = node.allocator.Commit(
			ctx, allocationID1,
			types.CommittedResources{
				AllocationID: allocationID1,
				Resources:    resrc,
			},
			nil, 0, 0,
		)
		require.NoError(t, err)

		err = node.allocator.Commit(
			ctx, allocationID2,
			types.CommittedResources{
				AllocationID: allocationID2,
				Resources:    resrc,
			},
			nil, 0, 0,
		)
		require.NoError(t, err)

		alloc1, err := node.createAllocation(
			allocationID1, jobtypes.AllocationType("task"),
			job, supervisorHandle,
		)
		assert.NoError(t, err)
		assert.NotNil(t, alloc1)

		alloc2, err := node.createAllocation(
			allocationID2, jobtypes.AllocationType("task"),
			job, supervisorHandle,
		)
		assert.NoError(t, err)
		assert.NotNil(t, alloc2)

		msg, err := actor.Message(
			sActor.Handle(), node.actor.Handle(),
			behaviors.AllocationsListBehavior, nil,
			actor.WithMessageReplyTo(behaviors.AllocationsListBehavior),
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		assert.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp AllocationsListResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Len(t, resp.Allocations, 2)
		assert.Contains(t, []string{resp.Allocations[0].ID, resp.Allocations[1].ID}, allocationID1)
		assert.Contains(t, []string{resp.Allocations[0].ID, resp.Allocations[1].ID}, allocationID2)
		assert.Empty(t, resp.Error)
	})
}

func TestHandleAllocationLogs(t *testing.T) {
	t.Parallel()
	ensembleID := "test-subnet-ensemble"
	allocationLogsBehavior := fmt.Sprintf(behaviors.AllocationLogsBehavior.DynamicTemplate, ensembleID)

	t.Run("wrong request", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, allocationLogsBehavior)

		// add behavior to test
		err := node.addEnsembleBehaviors(ensembleID)
		assert.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			allocationLogsBehavior,
			"wrong message",
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.AllocationLogsResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp.Error, types.ErrUnmarshal.Error())
		assert.Empty(t, resp.Stdout)
		assert.Empty(t, resp.Stderr)
	})

	t.Run("allocation plus logs not exist", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, allocationLogsBehavior)

		// add behavior to test
		err := node.addEnsembleBehaviors(ensembleID)
		assert.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			allocationLogsBehavior,
			orchestrator.AllocationLogsRequest{
				AllocName: "non-existent-allocation",
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.AllocationLogsResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp.Error, "does not exist")
		assert.Empty(t, resp.Stdout)
		assert.Empty(t, resp.Stderr)
	})

	t.Run("empty logs", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, allocationLogsBehavior)

		// add behavior to test
		err := node.addEnsembleBehaviors(ensembleID)
		assert.NoError(t, err)

		// setup log files
		allocName := "test-empty-logs-allocation"
		allocID := types.ConstructAllocationID(ensembleID, allocName)
		resultsDir := filepath.Join(node.dmsConfig.WorkDir, "jobs", allocID)
		err = node.fs.MkdirAll(resultsDir, 0o755)
		require.NoError(t, err)
		stdoutFile := filepath.Join(resultsDir, "stdout.log")
		stderrFile := filepath.Join(resultsDir, "stderr.log")
		err = afero.WriteFile(node.fs, stdoutFile, []byte(""), 0o644)
		require.NoError(t, err)
		err = afero.WriteFile(node.fs, stderrFile, []byte(""), 0o644)
		require.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			allocationLogsBehavior,
			orchestrator.AllocationLogsRequest{
				AllocName: allocName,
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.AllocationLogsResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp.Error, "empty")
		assert.Empty(t, resp.Stdout)
		assert.Empty(t, resp.Stderr)
	})

	t.Run("successful", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, allocationLogsBehavior)

		// add behavior to test
		err := node.addEnsembleBehaviors(ensembleID)
		assert.NoError(t, err)

		// setup log files
		allocName := "test-allocation"
		allocID := types.ConstructAllocationID(ensembleID, allocName)
		resultsDir := filepath.Join(node.dmsConfig.WorkDir, "jobs", allocID)
		err = node.fs.MkdirAll(resultsDir, 0o755)
		require.NoError(t, err)
		stdoutFile := filepath.Join(resultsDir, "stdout.log")
		stderrFile := filepath.Join(resultsDir, "stderr.log")
		err = afero.WriteFile(node.fs, stdoutFile, []byte("test stdout log"), 0o644)
		require.NoError(t, err)
		err = afero.WriteFile(node.fs, stderrFile, []byte("test stderr log"), 0o644)
		require.NoError(t, err)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			allocationLogsBehavior,
			orchestrator.AllocationLogsRequest{
				AllocName: allocName,
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp orchestrator.AllocationLogsResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.Empty(t, resp.Error)
		assert.Equal(t, "test stdout log", string(resp.Stdout))
		assert.Equal(t, "test stderr log", string(resp.Stderr))
	})
}
