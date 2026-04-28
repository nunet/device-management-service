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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/executor/null"
	"gitlab.com/nunet/device-management-service/tokenomics/eventhandler"
	"gitlab.com/nunet/device-management-service/types"
)

func TestHandleAllocatedResources(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("no allocation", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.ResourcesAllocatedBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.ResourcesAllocatedBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp ResourcesResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
		assert.Equal(t, float32(0), resp.Resources.CPU.Cores)
		assert.Equal(t, uint64(0), resp.Resources.RAM.Size)
	})

	t.Run("with allocation", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.ResourcesAllocatedBehavior)

		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		// allocate resources
		allocName := "alloc1"
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
		require.Equal(t, 1, len(node.allocator.(*allocator).getCommits()))

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
			eventhandler.New(ctx, 1, 1, time.Second, time.Second, node.handleContractEvents),
			"deployment-id",
			func(_ string, _ jobtypes.AllocationStatus) {},
		)
		require.NoError(t, err)
		require.NotNil(t, alloc)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.ResourcesAllocatedBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp ResourcesResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
		assert.Equal(t, float32(1), resp.Resources.CPU.Cores)
		assert.Equal(t, uint64(1), resp.Resources.RAM.Size)
	})
}

func TestHandleFreeResources(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("not onboarded", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.ResourcesFreeBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.ResourcesFreeBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp ResourcesResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.NotEmpty(t, resp.Error)
		assert.Contains(t, resp.Error, "getting onboarded resources")
	})

	t.Run("with onboarding and no allocation", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.ResourcesFreeBehavior)

		// onboard half the mocked resources
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.ResourcesFreeBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp ResourcesResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
		assert.Equal(t, float32(MockTotalCPU/2), resp.Resources.CPU.Cores)
		assert.Equal(t, uint64(MockTotalRAM/2), resp.Resources.RAM.Size)
	})

	t.Run("with onboarding and allocation", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.ResourcesFreeBehavior)
		// onboard half the mocked resources
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		// allocate resources
		allocName := "alloc1"
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
		require.Equal(t, 1, len(node.allocator.(*allocator).getCommits()))

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

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.ResourcesFreeBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp ResourcesResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
		assert.Equal(t, float32(MockTotalCPU/2)-1, resp.Resources.CPU.Cores)
		assert.Equal(t, uint64(MockTotalRAM/2)-1, resp.Resources.RAM.Size)
	})
}

func TestHandleOnboardedResources(t *testing.T) {
	t.Parallel()

	t.Run("not onboarded", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.ResourcesOnboardedBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.ResourcesOnboardedBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp ResourcesResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.NotEmpty(t, resp.Error)
		assert.Contains(t, resp.Error, "failed to get onboarded resources")
	})

	t.Run("with onboarding", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.ResourcesOnboardedBehavior)

		// onboard half the mocked resources
		mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.ResourcesOnboardedBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp ResourcesResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
		assert.Equal(t, float32(MockTotalCPU/2), resp.Resources.CPU.Cores)
		assert.Equal(t, uint64(MockTotalRAM/2), resp.Resources.RAM.Size)
	})
}

func TestHandleHardwareUsage(t *testing.T) {
	t.Parallel()

	t.Run("successful", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.HardwareUsageBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.HardwareUsageBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp ResourcesResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
		assert.Equal(t, float32(2), resp.Resources.CPU.Cores)
		assert.Equal(t, uint64(2*1024*1024*1024), resp.Resources.RAM.Size)
	})
}

func TestHandleHardwareSpec(t *testing.T) {
	t.Parallel()

	t.Run("successful", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.HardwareSpecBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.HardwareSpecBehavior,
			nil,
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp ResourcesResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
		assert.Equal(t, float32(MockTotalCPU), resp.Resources.CPU.Cores)
		assert.Equal(t, uint64(MockTotalRAM), resp.Resources.RAM.Size)
	})
}
