// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

func TestSupervision(t *testing.T) {
	actor.HealthCheckInterval = 1 * time.Second

	substrate := network.NewSubstrate()

	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	// Set up the behaviors first
	provider.MockDeploymentBehaviors(t, ensembleID, nil, orch.actor)

	healthcheckCh := make(chan struct{}, 1)

	// Override the AllocationDeployment behavior to create allocation actors
	require.NoError(t, provider.actor.AddBehavior(behaviors.AllocationDeploymentBehavior, func(msg actor.Envelope) {
		defer msg.Discard()
		go func() {
			select {
			case provider.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()

		// Create a new actor for each allocation
		allocationActor, err := actor.NewMockActor(
			orch.actor.Handle(),
			provider.net,
			provider.actor.Security(),
			nil,
			actor.Handle{
				ID:  provider.handle.ID,
				DID: provider.handle.DID,
				Address: actor.Address{
					HostID:       provider.peerID.String(),
					InboxAddress: "allocation-actor",
				},
			},
		)
		require.NoError(t, err)

		require.NoError(t, allocationActor.AddBehavior(behaviors.SubnetAddPeerBehavior, func(msg actor.Envelope) {
			reply, err := actor.ReplyTo(msg, behaviors.SubnetAddPeerResponse{
				OK: true,
			})
			if err != nil {
				log.Errorf("Failed to create reply: %v", err)
				return
			}

			reply.To = msg.From
			reply.From = provider.handle

			if err := allocationActor.Send(reply); err != nil {
				log.Errorf("Failed to send subnet add peer response: %v", err)
				return
			}
		}))

		require.NoError(t, allocationActor.AddBehavior(behaviors.SubnetDNSAddRecordsBehavior, func(msg actor.Envelope) {
			reply, err := actor.ReplyTo(msg, behaviors.SubnetDNSAddRecordsResponse{
				OK: true,
			})
			if err != nil {
				log.Errorf("Failed to create reply: %v", err)
				return
			}

			reply.To = msg.From
			reply.From = provider.handle

			if err := allocationActor.Send(reply); err != nil {
				log.Errorf("Failed to send subnet dns add records response: %v", err)
				return
			}
		}))

		require.NoError(t, allocationActor.AddBehavior(behaviors.SubnetMapPortBehavior, func(msg actor.Envelope) {
			reply, err := actor.ReplyTo(msg, behaviors.SubnetMapPortResponse{
				OK: true,
			})
			if err != nil {
				log.Errorf("Failed to create reply: %v", err)
				return
			}

			reply.To = msg.From
			reply.From = provider.handle

			if err := allocationActor.Send(reply); err != nil {
				log.Errorf("Failed to send subnet map port response: %v", err)
				return
			}
		}))

		// Add healthcheck behavior to the allocation actor
		require.NoError(t, allocationActor.AddBehavior(actor.HealthCheckBehavior, func(msg actor.Envelope) {
			select {
			case healthcheckCh <- struct{}{}:
			default:
			}

			reply, err := actor.ReplyTo(msg, behaviors.HealthCheckResponse{
				OK: true,
			})
			if err != nil {
				log.Errorf("Failed to create reply: %v", err)
				return
			}

			if err := allocationActor.Send(reply); err != nil {
				log.Errorf("Failed to send healthcheck response: %v", err)
				return
			}
		}))

		require.NoError(t, allocationActor.AddBehavior(behaviors.AllocationStartBehavior, func(msg actor.Envelope) {
			reply, err := actor.ReplyTo(msg, behaviors.AllocationStartResponse{
				OK: true,
			})
			if err != nil {
				log.Errorf("Failed to create reply: %v", err)
				return
			}

			reply.To = msg.From
			reply.From = provider.handle

			if err := allocationActor.Send(reply); err != nil {
				log.Errorf("Failed to send allocation start response: %v", err)
				return
			}
		}))
		require.NoError(t, allocationActor.AddBehavior(behaviors.RegisterHealthcheckBehavior, func(msg actor.Envelope) {
			reply, err := actor.ReplyTo(msg, behaviors.RegisterHealthcheckResponse{
				OK: true,
			})
			if err != nil {
				log.Errorf("Failed to create reply: %v", err)
				return
			}

			reply.To = msg.From
			reply.From = provider.handle

			if err := allocationActor.Send(reply); err != nil {
				log.Errorf("Failed to send register healthcheck response: %v", err)
				return
			}
		}))
		require.NoError(t, allocationActor.AddBehavior(actor.HealthCheckBehavior, func(msg actor.Envelope) {
			go func() {
				select {
				case healthcheckCh <- struct{}{}:
				default:
				}
			}()
			defer msg.Discard()

			reply, err := actor.ReplyTo(msg, behaviors.HealthCheckResponse{
				OK: true,
			})
			if err != nil {
				log.Errorf("Failed to create reply: %v", err)
				return
			}

			reply.To = msg.From
			reply.From = provider.handle

			if err := allocationActor.Send(reply); err != nil {
				log.Errorf("Failed to send healthcheck response: %v", err)
				return
			}
		}))

		require.NoError(t, allocationActor.Start())

		reply, err := actor.ReplyTo(msg, jtypes.AllocationDeploymentResponse{
			OK:          true,
			Allocations: map[string]actor.Handle{"test-ensemble_node1.alloc1": allocationActor.Handle()},
		})
		if err != nil {
			log.Errorf("Failed to create reply: %v", err)
			return
		}

		reply.To = msg.From
		reply.From = provider.handle

		if err := provider.actor.Send(reply); err != nil {
			log.Errorf("Failed to send allocation deployment response: %v", err)
			return
		}
	}))

	// Create test configuration
	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "US"},
						},
					},
					Allocations: []string{"alloc1"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
					HealthCheck: types.HealthCheckManifest{
						Type:     types.HealthCheckTypeHTTP,
						Endpoint: "http://localhost:8080/health",
						Response: types.HealthCheckResponse{
							Value: "OK",
						},
						Interval: time.Second,
					},
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      1,
							ClockSpeed: 1000,
						},
						RAM: types.RAM{
							Size: 1024,
						},
						Disk: types.Disk{
							Size: 1024,
						},
					},
				},
			},
		},
	}

	// Create orchestrator
	ctx := context.Background()
	fs := afero.NewMemMapFs()
	workDir := "/tmp"
	ensembleID := "test-ensemble"

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator(), nil, nil)
	require.NoError(t, err)

	// Start deployment
	expiry := time.Now().Add(2 * time.Minute)
	deployDone := make(chan error, 1)
	go func() {
		deployDone <- o.Deploy(expiry)
		close(deployDone)
	}()

	// Wait for deployment to complete
	select {
	case err := <-deployDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Minute):
		t.Fatal("Timeout waiting for deployment to complete")
	}

	// Verify deployment status
	assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status())

	// Wait for 5 healthchecks
	for i := 0; i < 5; i++ {
		select {
		case <-healthcheckCh:
			t.Logf("Received healthcheck %d", i+1)
		case <-time.After(60 * time.Second): // Slightly longer than the 30s interval
			t.Fatalf("Timeout waiting for healthcheck %d", i+1)
		}
	}
}

func TestSupervisorUpdate(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{{Country: "US"}},
					},
					Allocations: []string{"alloc1", "alloc2"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 1, ClockSpeed: 1000},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 1024},
					},
					HealthCheck: types.HealthCheckManifest{
						Type:     "http",
						Endpoint: "/health",
						Response: types.HealthCheckResponse{
							Type:  "string",
							Value: "OK",
						},
						Interval: time.Second,
					},
				},
				"alloc2": {
					Type: jtypes.AllocationTypeService,
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 1, ClockSpeed: 1000},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 1024},
					},
					HealthCheck: types.HealthCheckManifest{
						Type:     "http",
						Endpoint: "/health",
						Response: types.HealthCheckResponse{
							Type:  "string",
							Value: "OK",
						},
						Interval: time.Second,
					},
				},
			},
		},
	}

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator(), nil, nil)
	require.NoError(t, err)

	// Initial manifest with one allocation
	initialManifest := jtypes.EnsembleManifest{
		ID:           ensembleID,
		Orchestrator: orch.actor.Handle(),
		Allocations: map[string]jtypes.AllocationManifest{
			"node1.alloc1": {
				ID:     "test-ensemble_node1.alloc1",
				Type:   jtypes.AllocationTypeService,
				Status: jtypes.AllocationRunning,
				Handle: provider.handle,
				Healthcheck: types.HealthCheckManifest{
					Type:     "http",
					Endpoint: "/health",
					Response: types.HealthCheckResponse{
						Type:  "string",
						Value: "OK",
					},
					Interval: time.Second,
				},
			},
		},
		Nodes: map[string]jtypes.NodeManifest{
			"node1": {
				ID:          "node1",
				Allocations: []string{"node1.alloc1"},
			},
		},
	}

	// Start supervisor with initial manifest
	go o.supervisor.Supervise(jtypes.NewManifestReader(initialManifest))

	// Updated manifest with new allocation
	updatedManifest := jtypes.EnsembleManifest{
		ID:           ensembleID,
		Orchestrator: orch.actor.Handle(),
		Allocations: map[string]jtypes.AllocationManifest{
			"node1.alloc1": {
				ID:     "test-ensemble_node1.alloc1",
				Type:   jtypes.AllocationTypeService,
				Status: jtypes.AllocationRunning,
				Handle: provider.handle,
				Healthcheck: types.HealthCheckManifest{
					Type:     "http",
					Endpoint: "/health",
					Response: types.HealthCheckResponse{
						Type:  "string",
						Value: "OK",
					},
					Interval: time.Second,
				},
			},
			"node1.alloc2": {
				ID:     "test-ensemble_node1.alloc2",
				Type:   jtypes.AllocationTypeService,
				Status: jtypes.AllocationRunning,
				Handle: provider.handle,
				Healthcheck: types.HealthCheckManifest{
					Type:     "http",
					Endpoint: "/health",
					Response: types.HealthCheckResponse{
						Type:  "string",
						Value: "OK",
					},
					Interval: time.Second,
				},
			},
		},
		Nodes: map[string]jtypes.NodeManifest{
			"node1": {
				ID:          "node1",
				Allocations: []string{"node1.alloc1", "node1.alloc2"},
			},
		},
	}

	t.Run("update with new allocation", func(t *testing.T) {
		// Mock healthcheck registration for new allocation
		orch.channels[behaviors.RegisterHealthcheckBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.RegisterHealthcheckBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			go func() {
				orch.channels[behaviors.RegisterHealthcheckBehavior] <- struct{}{}
			}()
			resp := behaviors.RegisterHealthcheckResponse{OK: true}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			require.NoError(t, provider.actor.Send(reply))
		}))

		time.Sleep(100 * time.Millisecond) // Allow time for supervisor to start

		alloc, ok := o.supervisor.getAllocation("node1.alloc1")
		assert.True(t, ok)
		assert.Equal(t, "test-ensemble_node1.alloc1", alloc.ID)

		// alloc2 should not exist yet
		alloc, ok = o.supervisor.getAllocation("node1.alloc2")
		assert.False(t, ok)

		// Update supervisor with new manifest
		o.supervisor.Update(jtypes.NewManifestReader(updatedManifest))

		// Wait for healthcheck registration
		select {
		case <-orch.channels[behaviors.RegisterHealthcheckBehavior]:
			// Successfully registered healthcheck for new allocation
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for healthcheck registration")
		}

		// Verify manifest was updated
		alloc, ok = o.supervisor.getAllocation("node1.alloc2")
		assert.True(t, ok)
		assert.Equal(t, "test-ensemble_node1.alloc2", alloc.ID)
	})

	t.Run("update with removed allocation", func(t *testing.T) {
		// Create manifest with alloc2 removed
		removedManifest := jtypes.EnsembleManifest{
			ID:           ensembleID,
			Orchestrator: orch.actor.Handle(),
			Allocations: map[string]jtypes.AllocationManifest{
				"alloc1": {
					ID:     "test-ensemble_node1.alloc1",
					Type:   jtypes.AllocationTypeService,
					Status: jtypes.AllocationRunning,
					Handle: provider.handle,
					Healthcheck: types.HealthCheckManifest{
						Type:     "http",
						Endpoint: "/health",
						Response: types.HealthCheckResponse{
							Type:  "string",
							Value: "OK",
						},
						Interval: time.Second,
					},
				},
			},
			Nodes: map[string]jtypes.NodeManifest{
				"node1": {
					ID:          "node1",
					Allocations: []string{"alloc1"},
				},
			},
		}

		// Update supervisor with removed allocation
		o.supervisor.Update(jtypes.NewManifestReader(removedManifest))

		// Verify alloc2 is no longer in manifest
		_, ok := o.supervisor.getAllocation("node1.alloc2")
		assert.False(t, ok)
	})
}

func TestSupervisorPerformHealthCheck(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{{Country: "US"}},
					},
					Allocations: []string{"alloc1"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeTask,
					Resources: types.Resources{
						CPU:  types.CPU{Cores: 1, ClockSpeed: 1000},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 1024},
					},
					HealthCheck: types.HealthCheckManifest{
						Type:     "http",
						Endpoint: "/health",
						Response: types.HealthCheckResponse{
							Type:  "string",
							Value: "OK",
						},
						Interval: time.Second,
					},
				},
			},
		},
	}

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg, types.NewDefaultNodeIDGenerator(), types.NewDefaultAllocationIDGenerator(), nil, nil)
	require.NoError(t, err)

	allocation := jtypes.AllocationManifest{
		ID:     "test-ensemble_node1.alloc1",
		Type:   jtypes.AllocationTypeService,
		Status: jtypes.AllocationRunning,
		Handle: provider.handle,
		Healthcheck: types.HealthCheckManifest{
			Type:     "http",
			Endpoint: "/health",
			Response: types.HealthCheckResponse{
				Type:  "string",
				Value: "OK",
			},
			Interval: time.Second,
		},
	}
	o.supervisor.manifest.Allocations["node1.alloc1"] = allocation

	t.Run("terminated task", func(t *testing.T) {
		require.NoError(t, provider.actor.AddBehavior(actor.HealthCheckBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			go func() { provider.channels[actor.HealthCheckBehavior] <- struct{}{} }()
			resp := behaviors.HealthCheckResponse{OK: true}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			require.NoError(t, provider.actor.Send(reply))
		}))
		// Fix: update struct in map by value
		alloc := o.supervisor.manifest.Allocations["alloc1"]
		alloc.Status = jtypes.AllocationCompleted
		o.supervisor.manifest.Allocations["alloc1"] = alloc
		err := o.supervisor.performHealthCheck(allocation)
		assert.NoError(t, err)
	})

	t.Run("success", func(t *testing.T) {
		HealthCheckTimeout = 1 * time.Second
		provider.channels[actor.HealthCheckBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(actor.HealthCheckBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			go func() { provider.channels[actor.HealthCheckBehavior] <- struct{}{} }()
			resp := behaviors.HealthCheckResponse{OK: true}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			require.NoError(t, provider.actor.Send(reply))
		}))
		err := o.supervisor.performHealthCheck(allocation)
		assert.NoError(t, err)
		<-provider.channels[actor.HealthCheckBehavior]
	})

	t.Run("error response", func(t *testing.T) {
		HealthCheckTimeout = 1 * time.Second
		provider.channels[actor.HealthCheckBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(actor.HealthCheckBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			go func() { provider.channels[actor.HealthCheckBehavior] <- struct{}{} }()
			resp := behaviors.HealthCheckResponse{OK: false, Error: "fail"}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			require.NoError(t, provider.actor.Send(reply))
		}))
		err := o.supervisor.performHealthCheck(allocation)
		assert.NoError(t, err) // error is only logged, not returned
		<-provider.channels[actor.HealthCheckBehavior]
		assert.Equal(t, 1, o.supervisor.failures[allocation.ID])

		// reset state
		o.supervisor.failures = make(map[string]int)
		o.supervisor.escalations = make(map[string]int)
	})

	t.Run("escalation after 3 failures", func(t *testing.T) {
		HealthCheckTimeout = 1 * time.Second
		provider.channels[actor.HealthCheckBehavior] = make(chan struct{}, 1)
		failCount := 0
		require.NoError(t, provider.actor.AddBehavior(actor.HealthCheckBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			go func() { provider.channels[actor.HealthCheckBehavior] <- struct{}{} }()
			resp := behaviors.HealthCheckResponse{OK: false, Error: "fail"}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			require.NoError(t, provider.actor.Send(reply))
			failCount++
		}))

		provider.channels[behaviors.AllocationRestartBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.AllocationRestartBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			go func() { provider.channels[behaviors.AllocationRestartBehavior] <- struct{}{} }()
			resp := behaviors.AllocationRestartResponse{OK: true}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			require.NoError(t, provider.actor.Send(reply))
		}))
		// Simulate 3 failures
		for i := 0; i < 3; i++ {
			err := o.supervisor.performHealthCheck(allocation)
			assert.NoError(t, err)
			<-provider.channels[actor.HealthCheckBehavior]
		}
		<-provider.channels[behaviors.AllocationRestartBehavior]
		assert.Equal(t, 0, o.supervisor.failures[allocation.ID]) // should be reset after escalation
		assert.Equal(t, 1, o.supervisor.escalations[allocation.ID])
	})

	t.Run("timeout", func(t *testing.T) {
		provider.channels[actor.HealthCheckBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(actor.HealthCheckBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			// Do not reply to simulate timeout
		}))
		// Remove HealthCheckTimeout assignment if not possible
		err := o.supervisor.performHealthCheck(allocation)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout waiting for supervisor reply")
	})
}
