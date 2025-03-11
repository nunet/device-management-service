package orchestrator

import (
	"context"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/types"
)

func getMockEnsembleManifest(t *testing.T, id string) jobtypes.EnsembleManifest {
	t.Helper()

	allocations := make(map[string]jobtypes.AllocationManifest)
	allocations["allocation1"] = jobtypes.AllocationManifest{
		ID:     "allocation1",
		NodeID: "node1",
		Handle: actor.Handle{},
		Healthcheck: types.HealthCheckManifest{
			Type: "http",
		},
	}
	allocations["allocation2"] = jobtypes.AllocationManifest{
		ID:     "allocation2",
		NodeID: "node2",
		Handle: actor.Handle{},
		Healthcheck: types.HealthCheckManifest{
			Type: "http",
		},
	}

	nodes := make(map[string]jobtypes.NodeManifest)
	nodes["node1"] = jobtypes.NodeManifest{
		ID:          "node1",
		Peer:        "peer1",
		Handle:      actor.Handle{},
		Allocations: []string{"allocation1"},
	}
	nodes["node2"] = jobtypes.NodeManifest{
		ID:          "node2",
		Peer:        "peer2",
		Handle:      actor.Handle{},
		Allocations: []string{"allocation2"},
	}

	return jobtypes.EnsembleManifest{
		ID:           id,
		Orchestrator: actor.Handle{},
		Allocations:  allocations,
		Nodes:        nodes,
	}
}

func TestSupervisor(t *testing.T) {
	t.Parallel()

	// We test the following scenarios:
	// 1. Supervisor must be able to register healthchecks for all allocations
	// 2. Supervisor must be able to escalate failure only if healthcheck fails
	t.Run("must be able to escalate failure if healthcheck fails", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockOrchestratorActor := NewMockActor(ctrl)
		mockManifest := getMockEnsembleManifest(t, "ensemble1")
		supervisor := NewSupervisor(context.Background(), mockOrchestratorActor, "ensemble1")
		require.NotNil(t, supervisor)

		mockOrchestratorActor.EXPECT().Handle().Return(actor.Handle{}).AnyTimes()
		// We handle for the below events
		// 1. RegisterHealthcheckBehavior x 2 allocations
		// 2. HealthCheckBehavior x 3 fails
		// 3. HealthCheckBehavior x 2-3 success depending on allocation order
		// 4. AllocationRestartBehavior x 1 - Indicates escalation
		var wg sync.WaitGroup
		wg.Add(4) // 3 fails and a restart
		mockOrchestratorActor.EXPECT().Invoke(gomock.Any()).DoAndReturn(func(msg actor.Envelope) (<-chan actor.Envelope, error) {
			require.NotNil(t, msg)
			respChan := make(chan actor.Envelope, 1)
			go func() {
				switch msg.Behavior {
				case behaviors.RegisterHealthcheckBehavior:
					time.Sleep(1 * time.Second)
					reply := behaviors.RegisterHealthcheckResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.RegisterHealthcheckBehavior, reply)
					require.NoError(t, err)
					respChan <- env

				case actor.HealthCheckBehavior:
					allocationID, err := strconv.Unquote(string(msg.Message))
					require.NoError(t, err)

					// fail the healthcheck for only allocation2
					reply := behaviors.HealthCheckResponse{OK: true}
					if allocationID == "allocation2" {
						defer wg.Done()
						reply.OK = false
					}
					env, err := actor.Message(actor.Handle{}, msg.From, actor.HealthCheckBehavior, reply)
					require.NoError(t, err)
					respChan <- env

				case behaviors.AllocationRestartBehavior:
					defer wg.Done()
					allocationID, err := strconv.Unquote(string(msg.Message))
					require.NoError(t, err)
					require.Equal(t, "allocation2", allocationID)

					reply := behaviors.AllocationRestartResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationRestartBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				}
			}()

			return respChan, nil
		}).AnyTimes()
		go supervisor.Supervise(mockManifest)

		wg.Wait()

		// Ensure failure is escalated only for allocation2
		// Ensure failure is escalated only for allocation1
		require.Eventually(t, func() bool {
			return len(supervisor.escalations) == 1 && supervisor.escalations["allocation2"] == 1
		}, 5*time.Second, 1*time.Second)
	})

	// We test the following scenarios:
	// 1. Supervisor must be able to register healthchecks for all allocations
	// 2. Supervisor must be able to escalate failure only if Supervisor fails
	t.Run("must be able to escalate failure if Supervisor fails", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		mockOrchestratorActor := NewMockActor(ctrl)
		mockManifest := getMockEnsembleManifest(t, "ensemble1")
		supervisor := NewSupervisor(context.Background(), mockOrchestratorActor, "ensemble1")
		require.NotNil(t, supervisor)

		mockOrchestratorActor.EXPECT().Handle().Return(actor.Handle{}).AnyTimes()
		// We handle for the below events
		// 1. RegisterHealthcheckBehavior x 2 allocations
		// 2. HealthCheckBehavior x 3 Supervisor fails
		// 3. HealthCheckBehavior x 2-3 success depending on allocation order
		// 4. AllocationRestartBehavior x 1 - Indicates escalation
		var wg sync.WaitGroup
		wg.Add(4) // 3 fails and a restart
		mockOrchestratorActor.EXPECT().Invoke(gomock.Any()).DoAndReturn(func(msg actor.Envelope) (<-chan actor.Envelope, error) {
			require.NotNil(t, msg)
			respChan := make(chan actor.Envelope, 1)
			go func() {
				switch msg.Behavior {
				case behaviors.RegisterHealthcheckBehavior:
					reply := behaviors.RegisterHealthcheckResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.RegisterHealthcheckBehavior, reply)
					require.NoError(t, err)
					respChan <- env

				case actor.HealthCheckBehavior:
					allocationID, err := strconv.Unquote(string(msg.Message))
					require.NoError(t, err)

					// pass the healthcheck for allocation2
					if allocationID == "allocation2" {
						reply := behaviors.HealthCheckResponse{OK: true}
						env, err := actor.Message(actor.Handle{}, msg.From, actor.HealthCheckBehavior, reply)
						require.NoError(t, err)
						respChan <- env
						return
					}

					if allocationID == "allocation1" {
						// do not respond to healthcheck for allocation1
						defer wg.Done()
					}

				case behaviors.AllocationRestartBehavior:
					defer wg.Done()
					allocationID, err := strconv.Unquote(string(msg.Message))
					require.NoError(t, err)
					require.Equal(t, "allocation1", allocationID)

					reply := behaviors.AllocationRestartResponse{OK: true}
					env, err := actor.Message(actor.Handle{}, msg.From, behaviors.AllocationRestartBehavior, reply)
					require.NoError(t, err)
					respChan <- env
				}
			}()

			return respChan, nil
		}).AnyTimes()
		go supervisor.Supervise(mockManifest)

		wg.Wait()

		// Ensure failure is escalated only for allocation1
		require.Eventually(t, func() bool {
			return len(supervisor.escalations) == 1 && supervisor.escalations["allocation1"] == 1
		}, 5*time.Second, 1*time.Second)
	})
}
