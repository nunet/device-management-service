package node

import (
	"context"
	"testing"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/executor/null"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

func TestMonitorEnsembleAllocations(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	resrc := types.Resources{
		CPU:  types.CPU{Cores: 1},
		RAM:  types.RAM{Size: 1},
		Disk: types.Disk{Size: 1},
	}

	job := jobs.Job{
		Resources: resrc,
		Execution: types.SpecConfig{
			Type: "null",
		},
	}

	mockNode := &Node{}

	subs := network.NewSubstrate()
	mockAllocator, nVNet, _ := newMockAllocator(t, subs)
	mockNode.allocator = mockAllocator
	mockNode.network = nVNet

	_, orchPub, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
	require.NoError(t, err)
	orchHandle, err := actor.HandleFromDID(did.FromPublicKey(orchPub).String())
	require.NoError(t, err)

	nullExecutor, err := null.NewExecutor(ctx, "test-executor")
	require.NoError(t, err)

	t.Run("releasing a stopped allocation", func(t *testing.T) {
		t.Parallel()

		allocationID1 := "test-alloc-1"
		resourceToCommit1 := types.CommittedResources{
			AllocationID: allocationID1,
			Resources:    resrc,
		}
		allocationID2 := "test-alloc-2"
		resourceToCommit2 := types.CommittedResources{
			AllocationID: allocationID2,
			Resources:    resrc,
		}

		priv1, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
		require.NoError(t, err)
		allocActor1, _, _, _ := newActor(t, priv1, nVNet)
		defer func() {
			err := allocActor1.Stop()
			assert.NoError(t, err, "Stop should not return an error")
		}()

		priv2, _, err := crypto.GenerateKeyPair(crypto.Ed25519, 0)
		require.NoError(t, err)
		allocActor2, _, _, _ := newActor(t, priv2, nVNet)
		defer func() {
			err := allocActor2.Stop()
			assert.NoError(t, err, "Stop should not return an error")
		}()

		err = mockNode.allocator.Commit(
			ctx, allocationID1, resourceToCommit1, nil, 0, 0,
		)
		require.NoError(t, err, "commit should not return an error")

		allocation1, err := mockNode.allocator.Allocate(
			ctx, allocationID1, "service", allocActor1, orchHandle, job, nullExecutor)
		require.NoError(t, err, "allocate should not return an error")
		require.NotNil(t, allocation1, "allocation1 should not be nil on success")

		err = mockNode.allocator.Commit(
			ctx, allocationID2, resourceToCommit2, nil, 0, 0,
		)
		require.NoError(t, err, "commit should not return an error")

		allocation2, err := mockNode.allocator.Allocate(
			ctx, allocationID2, "service", allocActor2, orchHandle, job, nullExecutor)
		require.NoError(t, err, "allocate should not return an error")
		require.NotNil(t, allocation2, "allocation should not be nil on success")

		go mockNode.monitorEnsembleAllocations(
			"test-ensemble", []string{allocationID1, allocationID2},
		)

		// both allocations should be in pending state
		assert.Equal(t, 2, len(mockAllocator.GetAllocations()))
		alloc1, err := mockAllocator.GetAllocation(allocationID1)
		require.NoError(t, err, "GetAllocation should not return an error")
		assert.Equal(t, jobs.AllocationPending, alloc1.Status(ctx).Status)

		alloc2, err := mockAllocator.GetAllocation(allocationID2)
		require.NoError(t, err, "GetAllocation should not return an error")
		assert.Equal(t, jobs.AllocationPending, alloc2.Status(ctx).Status)

		// stop both allocations
		err = allocation1.Stop(ctx)
		require.NoError(t, err, "stop should not return an error")
		err = allocation2.Stop(ctx)
		require.NoError(t, err, "stop should not return an error")

		// after atleast 10 seconds(checkInterval), both allocs should still be in stopped state and
		// exist within the allocator
		time.Sleep(10 * time.Second)
		assert.Equal(t, 2, len(mockAllocator.GetAllocations()))
		alloc1, err = mockAllocator.GetAllocation(allocationID1)
		require.NoError(t, err, "GetAllocation should not return an error")
		assert.Equal(t, jobs.AllocationStopped, alloc1.Status(ctx).Status)
		alloc2, err = mockAllocator.GetAllocation(allocationID2)
		require.NoError(t, err, "GetAllocation should not return an error")
		assert.Equal(t, jobs.AllocationStopped, alloc2.Status(ctx).Status)

		// terminate allocation1 and delete allocation2 so it doesn't exist anymore
		err = allocation1.Terminate(ctx)
		require.NoError(t, err, "terminate should not return an error")
		mockAllocator.lock.Lock()
		delete(mockAllocator.allocations, allocationID2)
		mockAllocator.lock.Unlock()

		assert.Equal(t, 1, len(mockAllocator.GetAllocations()), "only allocation1 should remain after deletion of allocation2")

		// after atleast another 10 seconds, the remaining alloc should be removed from the allocator
		// by monitorEnsembleAllocations->cleanupFinishedEnsemble
		assert.Eventually(t, func() bool {
			if len(mockAllocator.GetAllocations()) == 0 {
				return true // allocations are removed
			}
			return false // allocations are still present
		}, 12*time.Second, 1*time.Second, "allocations should be removed by cleanupFinishedEnsemble after stop status")
	})
}
