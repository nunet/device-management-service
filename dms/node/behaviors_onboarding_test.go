package node

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/onboarding"
	"gitlab.com/nunet/device-management-service/types"
)

func TestHandleOnboard(t *testing.T) {
	t.Parallel()

	t.Run("empty onboard request", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.OnboardBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.OnboardBehavior,
			OnboardRequest{}, // Empty request
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)
		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp OnboardResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.NotEmpty(t, resp.Error)
	})

	t.Run("less than 10% RAM", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.OnboardBehavior)

		// mockHardware manager has machine resources as 12Cores, 32GB RAM, 100GB Disk
		// testing with less than 10% RAM and appropriate CPU
		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.OnboardBehavior,
			OnboardRequest{
				NoGPU: true,
				GPUs:  "",
				Config: types.OnboardingConfig{
					OnboardedResources: types.Resources{
						CPU:  types.CPU{Cores: 6},
						RAM:  types.RAM{Size: 1 * 1024 * 1024 * 1024},
						Disk: types.Disk{Size: 10 * 1024 * 1024 * 1024},
					},
				},
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp OnboardResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, onboarding.ErrUnmetCapacity.Error())
	})

	t.Run("more than 90% CPU", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.OnboardBehavior)

		// mockHardware manager has machine resources as 12Cores, 32GB RAM, 100GB Disk
		// testing with more than 90% CPU
		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.OnboardBehavior,
			OnboardRequest{
				NoGPU: true,
				GPUs:  "",
				Config: types.OnboardingConfig{
					OnboardedResources: types.Resources{
						CPU:  types.CPU{Cores: 0.5},
						RAM:  types.RAM{Size: 16 * 1024 * 1024 * 1024},
						Disk: types.Disk{Size: 10 * 1024 * 1024 * 1024},
					},
				},
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp OnboardResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.False(t, resp.OK)
		assert.Contains(t, resp.Error, onboarding.ErrUnmetCapacity.Error())
	})

	t.Run("successful request", func(t *testing.T) {
		t.Parallel()

		node, sActor, _ := newMockNodeWithSender(t, behaviors.OnboardBehavior)

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.OnboardBehavior,
			OnboardRequest{
				NoGPU: true,
				GPUs:  "",
				Config: types.OnboardingConfig{
					OnboardedResources: types.Resources{
						CPU:  types.CPU{Cores: 6},
						RAM:  types.RAM{Size: 16 * 1024 * 1024 * 1024},
						Disk: types.Disk{Size: 10 * 1024 * 1024 * 1024},
					},
				},
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(5*time.Second).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		assert.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp OnboardResponse
		err = json.Unmarshal(reply.Message, &resp)
		assert.NoError(t, err)
		assert.True(t, resp.OK)
		assert.Empty(t, resp.Error)
	})
}
