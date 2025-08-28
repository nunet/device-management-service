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
	provider.MockDeploymentBehaviors(t)

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
			Allocations: map[string]actor.Handle{"alloc1": allocationActor.Handle()},
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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
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
	case <-time.After(2 * time.Minute):
		t.Fatal("Timeout waiting for deployment to complete")
	}

	// Verify deployment status
	assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status())

	// Wait for 5 healthchecks
	for i := 0; i < 5; i++ {
		select {
		case <-healthcheckCh:
			t.Logf("Received healthcheck %d", i+1)
		case <-time.After(35 * time.Second): // Slightly longer than the 30s interval
			t.Fatalf("Timeout waiting for healthcheck %d", i+1)
		}
	}
}
