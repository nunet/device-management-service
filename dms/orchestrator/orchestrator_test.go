package orchestrator

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node/geolocation"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	workDir    = "/tmp"
	ensembleID = "test-ensemble"
)

func TestOrchestratorDeploy(t *testing.T) {
	BidRequestTimeout = 1 * time.Second
	CommitDeploymentTimeout = 1 * time.Second
	VerifyEdgeConstraintTimeout = 1 * time.Second
	AllocationDeploymentTimeout = 1 * time.Second
	AllocationStartTimeout = 1 * time.Second
	AllocationShutdownTimeout = 1 * time.Second

	substrate := network.NewSubstrate()

	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

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

	provider.MockDeploymentBehaviors(t)

	// Create orchestrator with orchestrator mock
	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Start deployment in a goroutine
	expiry := time.Now().Add(2 * time.Minute)
	deployDone := make(chan error, 1)
	go func() {
		deployDone <- o.Deploy(expiry)
		close(deployDone)
	}()

	// Create a context for status polling
	pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Subscribe to status changes
	statusCh := o.StatusChannel(pollCtx)

	// Track expected status transitions
	expectedStatuses := []jtypes.DeploymentStatus{
		jtypes.DeploymentStatusPreparing,
		jtypes.DeploymentStatusGenerating,
		jtypes.DeploymentStatusCommitting,
		jtypes.DeploymentStatusProvisioning,
		jtypes.DeploymentStatusRunning,
	}
	statusIndex := 0

	// Wait for status changes
	for status := range statusCh {
		t.Logf("Deployment status changed to: %s", status)
		if statusIndex < len(expectedStatuses) {
			assert.Equal(t, expectedStatuses[statusIndex], status)
			statusIndex++
		}
	}

	select {
	case err := <-deployDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Minute):
		t.Fatal("Timeout waiting for deployment to complete")
	}

	// Verify final state
	assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status())

	manifest := o.Manifest()
	assert.NotEmpty(t, manifest.Nodes)
	assert.NotEmpty(t, manifest.Allocations)

	// Verify node was deployed
	node, ok := manifest.Nodes["node1"]
	assert.True(t, ok)
	assert.Equal(t, provider.peerID.String(), node.Peer)

	// Verify allocation was deployed
	alloc, ok := manifest.Allocations["alloc1"]
	assert.True(t, ok)
	assert.Equal(t, "node1", alloc.NodeID)
}

func TestOrchestratorID(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	assert.Equal(t, ensembleID, o.ID())
}

func TestOrchestratorConfig(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	config := o.Config()
	assert.Equal(t, cfg.V1.Nodes["node1"].Allocations, config.V1.Nodes["node1"].Allocations)
	assert.Equal(t, cfg.V1.Allocations["alloc1"].Type, config.V1.Allocations["alloc1"].Type)
}

func TestOrchestratorStatus(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Initial status should be Preparing
	assert.Equal(t, jtypes.DeploymentStatusPreparing, o.status)

	statusCh := o.StatusChannel(context.Background())

	go func() {
		err := o.Shutdown()
		require.NoError(t, err)
	}()

	// Track expected status transitions
	expectedStatuses := []jtypes.DeploymentStatus{
		jtypes.DeploymentStatusPreparing,
		jtypes.DeploymentStatusShuttingDown,
		jtypes.DeploymentStatusCompleted,
	}
	statusIndex := 0

	// Wait for status changes
	for status := range statusCh {
		t.Logf("Deployment status changed to: %s", status)
		if statusIndex < len(expectedStatuses) {
			assert.Equal(t, expectedStatuses[statusIndex], status)
			statusIndex++
		}
	}
}

func TestOrchestratorManifest(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

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

	provider.MockDeploymentBehaviors(t)

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Start deployment in a goroutine
	expiry := time.Now().Add(2 * time.Minute)
	deployDone := make(chan error, 1)
	go func() {
		deployDone <- o.Deploy(expiry)
		close(deployDone)
	}()

	// Create a context for status polling
	pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Subscribe to status changes
	statusCh := o.StatusChannel(pollCtx)

	// Wait for deployment to complete
	for status := range statusCh {
		t.Logf("Deployment status changed to: %s", status)
	}

	select {
	case err := <-deployDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Minute):
		t.Fatal("Timeout waiting for deployment to complete")
	}

	// Manifest assertions after deployment is successful
	manifest := o.Manifest()
	assert.Equal(t, ensembleID, manifest.ID)
	assert.Equal(t, orch.actor.Handle(), manifest.Orchestrator)
	assert.NotEmpty(t, manifest.Nodes)
	assert.NotEmpty(t, manifest.Allocations)

	// Verify node was deployed
	node, ok := manifest.Nodes["node1"]
	assert.True(t, ok)
	assert.Equal(t, provider.peerID.String(), node.Peer)

	// Verify allocation was deployed
	alloc, ok := manifest.Allocations["alloc1"]
	assert.True(t, ok)
	assert.Equal(t, "node1", alloc.NodeID)
}

func TestOrchestratorActorPrivateKey(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	privKey := o.ActorPrivateKey()
	assert.NotNil(t, privKey)
}

func TestOrchestratorDeploymentSnapshot(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	snapshot := o.DeploymentSnapshot()
	assert.Empty(t, snapshot.Candidates)
}

func TestOrchestratorStatusChannel(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	ctx, cancel := context.WithCancel(context.Background())
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	statusCh := o.StatusChannel(ctx)

	// Should receive initial status
	status := <-statusCh
	assert.Equal(t, jtypes.DeploymentStatusPreparing, status)

	// Cancel context and verify channel is closed
	cancel()
	_, ok := <-statusCh
	assert.False(t, ok, "Status channel should be closed after context cancellation")
	// Test multiple subscribers
	ctx = context.Background()
	statusCh1 := o.StatusChannel(ctx)
	statusCh2 := o.StatusChannel(ctx)

	// Both channels should receive initial status
	status1 := <-statusCh1
	status2 := <-statusCh2
	assert.Equal(t, jtypes.DeploymentStatusPreparing, status1)
	assert.Equal(t, jtypes.DeploymentStatusPreparing, status2)
}

func TestOrchestratorGetAllocationLogs(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

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

	provider.MockDeploymentBehaviors(t)

	behavior := fmt.Sprintf(behaviors.AllocationLogsBehavior.DynamicTemplate, "test-ensemble")
	require.NoError(t, provider.actor.AddBehavior(behavior, func(msg actor.Envelope) {
		go func() {
			select {
			case provider.channels[msg.Behavior] <- struct{}{}:
			default:
			}
		}()
		defer msg.Discard()

		reply, err := actor.ReplyTo(msg, AllocationLogsResponse{
			Stdout: []byte("ok"),
			Stderr: []byte{},
			Error:  "",
		})
		if err != nil {
			t.Fatalf("creating reply: %s", err)
		}

		reply.To = msg.From
		reply.From = provider.handle

		if err := provider.actor.Send(reply); err != nil {
			t.Fatalf("sending allocation logs response: %s", err)
		}
	}))

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Start deployment in a goroutine
	expiry := time.Now().Add(2 * time.Minute)
	deployDone := make(chan error, 1)
	go func() {
		deployDone <- o.Deploy(expiry)
		close(deployDone)
	}()

	// Create a context for status polling
	pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	// Subscribe to status changes
	statusCh := o.StatusChannel(pollCtx)

	// Wait for deployment to complete
	go func() {
		for status := range statusCh {
			t.Logf("Deployment status changed to: %s", status)
		}
	}()

	select {
	case err := <-deployDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Minute):
		t.Fatal("Timeout waiting for deployment to complete")
	}

	// Verify final state
	assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status())

	node, ok := o.manifest.Nodes["node1"]
	require.True(t, ok)
	node.Handle = provider.handle
	o.updateNodeManifest("node1", o.manifest.Nodes["node1"])

	// Test GetAllocationLogs
	logs, err := o.GetAllocationLogs("alloc1")
	require.NoError(t, err)
	assert.Equal(t, "ok", string(logs.Stdout))
}

func TestHandleTaskTermination(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	provider.MockDeploymentBehaviors(t)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()
	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Deploy the ensemble
	expiry := time.Now().Add(2 * time.Minute)
	deployDone := make(chan error, 1)
	go func() {
		deployDone <- o.Deploy(expiry)
		close(deployDone)
	}()

	select {
	case err := <-deployDone:
		require.NoError(t, err)
	case <-time.After(5 * time.Minute):
		t.Fatal("Timeout waiting for deployment to complete")
	}

	tests := []struct {
		name         string
		notification behaviors.TaskTerminationNotification
		checkStatus  func(t *testing.T, o *BasicOrchestrator)
	}{
		{
			name: "successful task termination",
			notification: behaviors.TaskTerminationNotification{
				AllocationID: "test-ensemble_alloc1",
				Status:       string(jtypes.AllocationCompleted),
			},
			checkStatus: func(t *testing.T, o *BasicOrchestrator) {
				alloc, ok := o.manifest.Allocations["alloc1"]
				if !ok {
					t.Fatal("allocation not found in manifest")
				}
				if alloc.Status != jtypes.AllocationCompleted {
					t.Fatalf("expected status %v, got %v", jtypes.AllocationCompleted, alloc.Status)
				}
			},
		},
		{
			name: "task termination with error",
			notification: behaviors.TaskTerminationNotification{
				AllocationID: "test-ensemble_alloc1",
				Status:       string(jtypes.AllocationFailed),
				Error: behaviors.TerminationError{
					ExitCode: 1,
					Err:      "execution exit code != 0, exit code: 1",
				},
			},
			checkStatus: func(t *testing.T, o *BasicOrchestrator) {
				alloc, ok := o.manifest.Allocations["alloc1"]
				if !ok {
					t.Fatal("allocation not found in manifest")
				}
				if alloc.Status != jtypes.AllocationFailed {
					t.Fatalf("expected status %v, got %v", jtypes.AllocationFailed, alloc.Status)
				}
			},
		},
		{
			name: "task termination with logs",
			notification: behaviors.TaskTerminationNotification{
				AllocationID: "test-ensemble_alloc1",
				Status:       string(jtypes.AllocationCompleted),
				Stdout:       []byte("test stdout"),
				Stderr:       []byte("test stderr"),
			},
			checkStatus: func(t *testing.T, o *BasicOrchestrator) {
				alloc, ok := o.manifest.Allocations["alloc1"]
				if !ok {
					t.Fatal("allocation not found in manifest")
				}
				if alloc.Status != jtypes.AllocationCompleted {
					t.Fatalf("expected status %v, got %v", jtypes.AllocationCompleted, alloc.Status)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := actor.Message(
				provider.handle,
				orch.actor.Handle(),
				behaviors.NotifyTaskTerminationBehavior,
				tt.notification,
				actor.WithMessageSource(provider.handle),
				actor.WithMessageReplyTo("replyto/123"),
			)
			if err != nil {
				t.Fatalf("failed to create message: %v", err)
			}

			o.handleTaskTermination(msg)

			if tt.checkStatus != nil {
				tt.checkStatus(t, o)
			}
		})
	}
}

func TestWriteAllocationLogs(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	ctx := context.Background()
	fs := afero.Afero{Fs: afero.NewMemMapFs()}
	o, err := NewOrchestrator(ctx, fs, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Write allocation logs
	stdout := []byte("test stdout")
	stderr := []byte("test stderr")
	allocDir, err := o.WriteAllocationLogs("alloc1", stdout, stderr)
	require.NoError(t, err)
	assert.NotEmpty(t, allocDir)

	stdoutContent, err := fs.ReadFile("/tmp/deployments/test-ensemble/alloc1/stdout.logs")
	require.NoError(t, err)
	stderrContent, err := fs.ReadFile("/tmp/deployments/test-ensemble/alloc1/stderr.logs")
	require.NoError(t, err)
	assert.Equal(t, stdout, stdoutContent)
	assert.Equal(t, stderr, stderrContent)
}

func TestAllocNameFromID(t *testing.T) {
	// Test allocation name extraction
	allocID := "test-ensemble_alloc1"
	allocName := allocNameFromID(allocID)
	assert.Equal(t, "alloc1", allocName)

	// Test invalid allocation ID
	invalidID := "invalid-id"
	allocName = allocNameFromID(invalidID)
	assert.Empty(t, allocName)
}

func TestVerifyEdgeConstraints(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

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
				"node2": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "UK"},
						},
					},
					Allocations: []string{"alloc2"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
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
				"alloc2": {
					Type: jtypes.AllocationTypeService,
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
			Edges: []jtypes.EdgeConstraint{
				{
					S:   "node1",
					T:   "node2",
					RTT: 100,
					BW:  1000,
				},
			},
		},
	}

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Create test bids
	bid1 := jtypes.Bid{
		V1: &jtypes.BidV1{
			EnsembleID: ensembleID,
			NodeID:     "node1",
			Peer:       provider.peerID.String(),
			Location:   jtypes.Location{Country: "US"},
			Handle:     provider.handle,
		},
	}

	bid2 := jtypes.Bid{
		V1: &jtypes.BidV1{
			EnsembleID: ensembleID,
			NodeID:     "node2",
			Peer:       provider.peerID.String(),
			Location:   jtypes.Location{Country: "UK"},
			Handle:     provider.handle,
		},
	}

	// Test valid edge constraints
	candidate := map[string]jtypes.Bid{
		"node1": bid1,
		"node2": bid2,
	}

	t.Run("valid edge constraints", func(t *testing.T) {
		// Mock the provider's behavior for edge constraint verification
		provider.channels[behaviors.VerifyEdgeConstraintBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.VerifyEdgeConstraintBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()

			reply, err := actor.ReplyTo(msg, VerifyEdgeConstraintResponse{
				OK: true,
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider.handle
			require.NoError(t, provider.actor.Send(reply))
		}))

		// Test valid edge constraints
		result := o.verifyEdgeConstraints(cfg, candidate, map[string]bool{})
		assert.True(t, result)
	})

	t.Run("invalid edge constraints (timeout)", func(t *testing.T) {
		// Test invalid edge constraints (timeout)
		VerifyEdgeConstraintTimeout = 1 * time.Second
		provider.channels[behaviors.VerifyEdgeConstraintBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.VerifyEdgeConstraintBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()
			// Don't send a reply to simulate timeout
		}))

		result := o.verifyEdgeConstraints(cfg, candidate, map[string]bool{})
		assert.False(t, result)
	})

	t.Run("invalid edge constraints (error response)", func(t *testing.T) {
		provider.channels[behaviors.VerifyEdgeConstraintBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.VerifyEdgeConstraintBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()

			reply, err := actor.ReplyTo(msg, VerifyEdgeConstraintResponse{
				OK:    false,
				Error: "constraint violation",
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider.handle
			require.NoError(t, provider.actor.Send(reply))
		}))

		result := o.verifyEdgeConstraints(cfg, candidate, map[string]bool{})
		assert.False(t, result)
	})
}

func TestRevertNodeDeployment(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Set up test manifest
	manifest := jtypes.EnsembleManifest{
		ID:           ensembleID,
		Orchestrator: orch.actor.Handle(),
		Allocations: map[string]jtypes.AllocationManifest{
			"alloc1": {
				ID:     "test-ensemble_alloc1",
				Type:   jtypes.AllocationTypeService,
				Status: jtypes.AllocationRunning,
			},
		},
		Nodes: map[string]jtypes.NodeManifest{
			"node1": {
				ID:          "node1",
				Allocations: []string{"alloc1"},
				Handle:      provider.handle,
			},
		},
	}
	o.manifest = manifest

	// Test successful revert
	t.Run("successful revert", func(t *testing.T) {
		provider.channels[behaviors.DeploymentRevertBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.DeploymentRevertBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()
		}))

		o.revertNodeDeployment(cfg, "node1", provider.handle)
		<-provider.channels[behaviors.DeploymentRevertBehavior]

		// Verify node was removed from manifest
		_, ok := o.manifest.Nodes["node1"]
		assert.False(t, ok)
		_, ok = o.manifest.Allocations["alloc1"]
		assert.False(t, ok)
	})

	// Test revert failure
	t.Run("revert failure", func(t *testing.T) {
		// Reset manifest
		o.manifest = manifest

		provider.channels[behaviors.DeploymentRevertBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.DeploymentRevertBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()
		}))

		o.revertNodeDeployment(cfg, "node1", provider.handle)
		<-provider.channels[behaviors.DeploymentRevertBehavior]

		// Verify node was still removed from manifest despite failure
		_, ok := o.manifest.Nodes["node1"]
		assert.False(t, ok)
		_, ok = o.manifest.Allocations["alloc1"]
		assert.False(t, ok)
	})

	// Test non-existent node
	t.Run("non-existent node", func(t *testing.T) {
		// Reset manifest
		o.manifest = manifest

		o.revertNodeDeployment(cfg, "non-existent", provider.handle)

		// Verify manifest is unchanged
		assert.Equal(t, manifest, o.manifest)
	})
}

func TestRevert(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

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
				"node2": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "UK"},
						},
					},
					Allocations: []string{"alloc2"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
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
				"alloc2": {
					Type: jtypes.AllocationTypeService,
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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Set up test manifest
	manifest := jtypes.EnsembleManifest{
		ID:           ensembleID,
		Orchestrator: orch.actor.Handle(),
		Allocations: map[string]jtypes.AllocationManifest{
			"alloc1": {
				ID:     "test-ensemble_alloc1",
				Type:   jtypes.AllocationTypeService,
				Status: jtypes.AllocationRunning,
			},
			"alloc2": {
				ID:     "test-ensemble_alloc2",
				Type:   jtypes.AllocationTypeService,
				Status: jtypes.AllocationRunning,
			},
		},
		Nodes: map[string]jtypes.NodeManifest{
			"node1": {
				ID:          "node1",
				Allocations: []string{"alloc1"},
				Handle:      provider.handle,
			},
			"node2": {
				ID:          "node2",
				Allocations: []string{"alloc2"},
				Handle:      provider.handle,
			},
		},
	}
	o.manifest = manifest

	// Mock the provider's behavior for deployment revert
	provider.channels[behaviors.DeploymentRevertBehavior] = make(chan struct{}, 2)
	require.NoError(t, provider.actor.AddBehavior(behaviors.DeploymentRevertBehavior, func(msg actor.Envelope) {
		defer msg.Discard()
		defer func() {
			provider.channels[msg.Behavior] <- struct{}{}
		}()
	}))

	// Test successful revert
	o.revert(cfg, manifest)

	// Wait for both nodes to be reverted
	<-provider.channels[behaviors.DeploymentRevertBehavior]
	<-provider.channels[behaviors.DeploymentRevertBehavior]

	// Verify all nodes and allocations were removed
	assert.Empty(t, o.manifest.Nodes)
	assert.Empty(t, o.manifest.Allocations)
}

func TestRemoveNodeFromManifest(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "US"},
						},
					},
					Allocations: []string{"alloc1", "alloc2"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
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
				"alloc2": {
					Type: jtypes.AllocationTypeService,
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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Set up test manifest
	manifest := jtypes.EnsembleManifest{
		ID:           ensembleID,
		Orchestrator: orch.actor.Handle(),
		Allocations: map[string]jtypes.AllocationManifest{
			"alloc1": {
				ID:     "test-ensemble_alloc1",
				Type:   jtypes.AllocationTypeService,
				Status: jtypes.AllocationRunning,
			},
			"alloc2": {
				ID:     "test-ensemble_alloc2",
				Type:   jtypes.AllocationTypeService,
				Status: jtypes.AllocationRunning,
			},
		},
		Nodes: map[string]jtypes.NodeManifest{
			"node1": {
				ID:          "node1",
				Allocations: []string{"alloc1", "alloc2"},
			},
		},
	}
	o.manifest = manifest

	// Test removing existing node
	o.removeNodeFromManifest("node1")
	assert.Empty(t, o.manifest.Nodes)
	assert.Empty(t, o.manifest.Allocations)

	// Test removing non-existent node
	o.removeNodeFromManifest("non-existent")
	assert.Empty(t, o.manifest.Nodes)
	assert.Empty(t, o.manifest.Allocations)
}

func TestShutdown(t *testing.T) {
	t.Skip("Skipping testShutdown due to a bug with consecutive deployments causing a deadlock. To be investigated.")
	SubnetDestroyTimeout = time.Second * 1
	AllocationShutdownTimeout = time.Second * 1

	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	provider.MockDeploymentBehaviors(t)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	deploy := func() {
		t.Helper()
		// Start deployment in a goroutine
		expiry := time.Now().Add(2 * time.Minute)
		deployDone := make(chan error, 1)
		go func() {
			deployDone <- o.Deploy(expiry)
			close(deployDone)
		}()

		// Create a context for status polling
		pollCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		defer cancel()

		// Subscribe to status changes
		statusCh := o.StatusChannel(pollCtx)

		// Wait for status changes
		go func() {
			for status := range statusCh {
				t.Logf("Deployment status changed to: %s", status)
			}
		}()

		select {
		case err := <-deployDone:
			require.NoError(t, err)
		case <-time.After(60 * time.Second):
			t.Fatal("Timeout waiting for deployment to complete")
		}

		// Verify final state
		assert.Equal(t, jtypes.DeploymentStatusRunning, o.Status())
	}

	manifest := o.Manifest()

	t.Run("happy path", func(t *testing.T) {
		provider.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)] <- struct{}{}
			}()
			reply, err := actor.ReplyTo(msg, SubnetDestroyResponse{
				OK: true,
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider.handle
			require.NoError(t, provider.actor.Send(reply))
		}))

		provider.channels[fmt.Sprintf(behaviors.AllocationShutdownBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(fmt.Sprintf(behaviors.AllocationShutdownBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[fmt.Sprintf(behaviors.AllocationShutdownBehavior.DynamicTemplate, ensembleID)] <- struct{}{}
			}()
			reply, err := actor.ReplyTo(msg, AllocationStopResponse{
				OK: true,
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider.handle
			require.NoError(t, provider.actor.Send(reply))
		}))

		deploy()

		// Test shutdown from running state
		o.setStatus(jtypes.DeploymentStatusRunning)
		require.NoError(t, o.Shutdown())
		<-provider.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)]
		<-provider.channels[fmt.Sprintf(behaviors.AllocationShutdownBehavior.DynamicTemplate, ensembleID)]
		assert.Equal(t, jtypes.DeploymentStatusCompleted, o.Status())

		t.Run("already shutdown", func(t *testing.T) {
			o.setStatus(jtypes.DeploymentStatusCompleted)
			o.manifest = manifest
			require.NoError(t, o.Shutdown())
			assert.Equal(t, jtypes.DeploymentStatusCompleted, o.Status())

			o.setStatus(jtypes.DeploymentStatusShuttingDown)
			o.manifest = manifest
			require.NoError(t, o.Shutdown())
		})
	})

	t.Run("failed state", func(t *testing.T) {
		// Test shutdown from failed state
		o.setStatus(jtypes.DeploymentStatusFailed)
		o.manifest = manifest

		o.cancel = func() {
		}
		require.NoError(t, o.Shutdown())
		assert.Equal(t, jtypes.DeploymentStatusCompleted, o.Status())
	})

	// Test shutdown with subnet destroy failure
	t.Run("subnet destroy failure", func(t *testing.T) {
		CommitDeploymentTimeout = time.Second * 1
		AllocationShutdownTimeout = time.Second * 1
		SubnetCreateTimeout = time.Second * 1

		provider.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer msg.Discard()
			go func() {
				provider.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)] <- struct{}{}
			}()
			reply, err := actor.ReplyTo(msg, SubnetDestroyResponse{
				OK:    false,
				Error: "subnet destroy failed",
			})
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = provider.handle
			require.NoError(t, provider.actor.Send(reply))
		}))

		deploy()
		require.Error(t, o.Shutdown())
		assert.Equal(t, jtypes.DeploymentStatusCompleted, o.Status())
	})

	// Test shutdown with subnet destroy timeout
	t.Run("subnet destroy timeout", func(t *testing.T) {
		provider.channels[fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID)] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID), func(msg actor.Envelope) {
			defer msg.Discard()
			// Don't send a reply to simulate timeout
		}))

		deploy()

		require.Error(t, o.Shutdown())
		assert.Equal(t, jtypes.DeploymentStatusCompleted, o.Status())
	})
}

func TestContainsExecutor(t *testing.T) {
	// Test with executor in list
	executors := []jtypes.AllocationExecutor{jtypes.ExecutorDocker, jtypes.ExecutorFirecracker, jtypes.ExecutorNull}
	assert.True(t, containsExecutor(executors, jtypes.ExecutorDocker))
	assert.True(t, containsExecutor(executors, jtypes.ExecutorFirecracker))
	assert.True(t, containsExecutor(executors, jtypes.ExecutorNull))

	// Test with executor not in list
	assert.False(t, containsExecutor([]jtypes.AllocationExecutor{
		jtypes.ExecutorFirecracker,
		jtypes.ExecutorNull,
	}, jtypes.ExecutorDocker))

	// Test with empty list
	assert.False(t, containsExecutor([]jtypes.AllocationExecutor{}, jtypes.ExecutorDocker))
}

func TestRequestBidPeer(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	// Set up test configuration
	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Peer: provider.peerID.String(),
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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Create test bid request
	bidRequest := jtypes.EnsembleBidRequest{
		ID:    ensembleID,
		Nonce: o.getNonce(),
		Request: []jtypes.BidRequest{
			{
				V1: &jtypes.BidRequestV1{
					NodeID: "node1",
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "US"},
						},
					},
				},
			},
		},
	}

	// Test successful bid request
	t.Run("successful bid request", func(t *testing.T) {
		provider.channels[behaviors.BidRequestBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.BidRequestBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()

			var request jtypes.EnsembleBidRequest
			if err := json.Unmarshal(msg.Message, &request); err != nil {
				t.Fatalf("unmarshal bid request: %s", err)
			}

			// Create and sign bid response
			bid := jtypes.Bid{
				V1: &jtypes.BidV1{
					EnsembleID: request.ID,
					NodeID:     "node1",
					Peer:       provider.peerID.String(),
					Location:   jtypes.Location{Country: "US"},
					Handle:     provider.handle,
				},
			}

			reply, err := actor.Message(provider.handle, msg.From, msg.Options.ReplyTo, bid)
			if err != nil {
				t.Fatalf("creating reply: %s", err)
			}

			if err := provider.actor.Send(reply); err != nil {
				t.Fatalf("sending bid response: %s", err)
			}
		}))

		err := o.requestBidPeer(bidRequest, cfg.V1.Nodes["node1"], uint64(time.Now().Add(BidRequestTimeout).UnixNano()))
		require.NoError(t, err)
		<-provider.channels[behaviors.BidRequestBehavior]
	})

	// Test invalid peer ID
	t.Run("invalid peer ID", func(t *testing.T) {
		invalidNodeConfig := jtypes.NodeConfig{
			Peer: "invalid-peer-id",
		}

		err := o.requestBidPeer(bidRequest, invalidNodeConfig, uint64(time.Now().Add(BidRequestTimeout).UnixNano()))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "getting handle of selected peer")
	})

	// Test message sending error
	t.Run("message sending error", func(t *testing.T) {
		provider.channels[behaviors.BidRequestBehavior] = make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.BidRequestBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			defer func() {
				provider.channels[msg.Behavior] <- struct{}{}
			}()
			// Don't send a reply to simulate sending error
		}))

		err := o.requestBidPeer(bidRequest, cfg.V1.Nodes["node1"], uint64(time.Now().Add(BidRequestTimeout).UnixNano()))
		require.NoError(t, err) // The function itself doesn't return an error for sending failures
		<-provider.channels[behaviors.BidRequestBehavior]
	})
}

func TestMakeCandidateDeploymentBig(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	// Set up test configuration
	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Peer: provider.peerID.String(),
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "US"},
						},
					},
					Allocations: []string{"alloc1", "alloc2"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
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
				"alloc2": {
					Type: jtypes.AllocationTypeService,
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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Create test bids
	bid := jtypes.Bid{
		V1: &jtypes.BidV1{
			EnsembleID: ensembleID,
			NodeID:     "node1",
			Peer:       provider.peerID.String(),
			Location:   jtypes.Location{Country: "US"},
			Handle:     provider.handle,
		},
	}

	bids := map[string][]jtypes.Bid{
		"node1": {bid},
	}

	// Test successful deployment creation
	t.Run("successful deployment creation", func(t *testing.T) {
		nextCandidate, ok := o.makeCandidateDeploymentBig(cfg, bids)
		require.True(t, ok)
		require.NotNil(t, nextCandidate)

		candidate, ok := nextCandidate()
		require.True(t, ok)
		assert.NotNil(t, candidate)
		assert.Equal(t, bid, candidate["node1"])
	})

	// Test deployment creation with invalid bid
	t.Run("invalid bid", func(t *testing.T) {
		invalidBids := map[string][]jtypes.Bid{
			"node1": {{
				V1: &jtypes.BidV1{
					EnsembleID: "invalid-ensemble",
					NodeID:     "node1",
					Peer:       provider.peerID.String(),
					Location:   jtypes.Location{Country: "US"},
					Handle:     provider.handle,
				},
			}},
		}

		nextCandidate, ok := o.makeCandidateDeploymentBig(cfg, invalidBids)
		require.True(t, ok)
		require.NotNil(t, nextCandidate)

		candidate, ok := nextCandidate()
		require.True(t, ok)
		assert.NotNil(t, candidate)
		assert.Equal(t, invalidBids["node1"][0], candidate["node1"])
	})
}

func TestMakeResidualBidRequest(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	// Set up test configuration
	cfg := jtypes.EnsembleConfig{
		V1: &jtypes.EnsembleConfigV1{
			Nodes: map[string]jtypes.NodeConfig{
				"node1": {
					Peer: provider.peerID.String(),
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "US"},
						},
					},
					Allocations: []string{"alloc1"},
				},
				"node2": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "US"},
						},
					},
					Allocations: []string{"alloc2"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
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
				"alloc2": {
					Type: jtypes.AllocationTypeService,
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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Create test bids
	bid := jtypes.Bid{
		V1: &jtypes.BidV1{
			EnsembleID: ensembleID,
			NodeID:     "node1",
			Peer:       provider.peerID.String(),
			Location:   jtypes.Location{Country: "US"},
			Handle:     provider.handle,
		},
	}

	bids := map[string][]jtypes.Bid{
		"node1": {bid},
	}

	// Test successful residual bid request creation
	t.Run("successful residual bid request", func(t *testing.T) {
		rmBid := func(_ jtypes.Bid) {
			// This is a no-op for testing
		}

		request, err := o.makeResidualBidRequest(cfg, bids, rmBid)
		require.NoError(t, err)
		assert.NotNil(t, request)
		assert.Equal(t, ensembleID, request.ID)
		assert.Equal(t, 1, len(request.Request))
		assert.Equal(t, "node2", request.Request[0].V1.NodeID)
	})

	// Test residual bid request with complete candidate
	t.Run("complete candidate", func(t *testing.T) {
		completeBids := map[string][]jtypes.Bid{
			"node1": {bid},
			"node2": {{
				V1: &jtypes.BidV1{
					EnsembleID: ensembleID,
					NodeID:     "node2",
					Peer:       provider.peerID.String(),
					Location:   jtypes.Location{Country: "US"},
					Handle:     provider.handle,
				},
			}},
		}

		rmBid := func(_ jtypes.Bid) {
			// This is a no-op for testing
		}

		request, err := o.makeResidualBidRequest(cfg, completeBids, rmBid)
		require.NoError(t, err)
		assert.Empty(t, request.Request)
	})
}

func TestMonitorOnlyTaskManifest(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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
					Type: jtypes.AllocationTypeTask,
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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Test with only task manifest
	manifest := jtypes.EnsembleManifest{
		ID:           ensembleID,
		Orchestrator: orch.actor.Handle(),
		Allocations: map[string]jtypes.AllocationManifest{
			"alloc1": {
				ID:     "test-ensemble_alloc1",
				Type:   jtypes.AllocationTypeTask,
				Status: jtypes.AllocationRunning,
			},
		},
		Nodes: map[string]jtypes.NodeManifest{
			"node1": {
				ID:          "node1",
				Allocations: []string{"alloc1"},
			},
		},
	}
	o.manifest = manifest

	MonitorOnlyTaskManifestInterval = time.Millisecond * 200
	// Test successful task termination
	t.Run("successful task termination", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		o.ctx = ctx
		o.cancel = cancel
		defer cancel()

		go o.monitorOnlyTaskManifest(manifest)
		time.Sleep(250 * time.Millisecond)
		assert.NotEqual(t, jtypes.DeploymentStatusCompleted, o.Status())

		o.lock.Lock()
		alloc, ok := o.manifest.Allocations["alloc1"]
		require.True(t, ok)
		alloc.Status = jtypes.AllocationCompleted
		o.manifest.Allocations["alloc1"] = alloc
		o.lock.Unlock()

		time.Sleep(250 * time.Millisecond)
		assert.Equal(t, jtypes.DeploymentStatusCompleted, o.Status())
	})

	// reset the allocation status and orchestrator status
	o.lock.Lock()
	alloc, ok := o.manifest.Allocations["alloc1"]
	require.True(t, ok)
	alloc.Status = jtypes.AllocationRunning
	o.manifest.Allocations["alloc1"] = alloc
	o.lock.Unlock()

	o.setStatus(jtypes.DeploymentStatusRunning)

	// Test context cancellation
	t.Run("context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		o.ctx = ctx
		o.cancel = cancel

		done := make(chan struct{})
		go func() {
			o.monitorOnlyTaskManifest(manifest)
			close(done)
		}()

		time.Sleep(50 * time.Millisecond)
		cancel()

		select {
		case <-done:
			assert.NotEqual(t, jtypes.DeploymentStatusCompleted, o.Status())
		case <-time.After(2 * time.Second):
			t.Fatal("timeout waiting for task monitoring to stop")
		}
	})

	// Test with non-task manifest
	t.Run("non-task manifest", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		o.ctx = ctx
		o.cancel = cancel
		defer cancel()

		// Create a new manifest with service type
		serviceManifest := jtypes.EnsembleManifest{
			ID:           ensembleID,
			Orchestrator: orch.actor.Handle(),
			Allocations: map[string]jtypes.AllocationManifest{
				"alloc1": {
					ID:     "test-ensemble_alloc1",
					Type:   jtypes.AllocationTypeService,
					Status: jtypes.AllocationRunning,
				},
			},
			Nodes: map[string]jtypes.NodeManifest{
				"node1": {
					ID:          "node1",
					Allocations: []string{"alloc1"},
				},
			},
		}

		o.manifest = serviceManifest
		go o.monitorOnlyTaskManifest(serviceManifest)

		time.Sleep(250 * time.Millisecond)
		assert.NotEqual(t, jtypes.DeploymentStatusCompleted, o.Status())
	})
}

func TestAggregateErrors(t *testing.T) {
	// Test with no errors
	errCh := make(chan error, 1)
	close(errCh)
	result := aggregateErrors(errCh)
	assert.NoError(t, result)

	// Test with single error
	errCh = make(chan error, 1)
	errCh <- fmt.Errorf("error1")
	close(errCh)
	result = aggregateErrors(errCh)
	assert.Error(t, result)
	assert.Equal(t, "error1", result.Error())

	// Test with multiple errors
	errCh = make(chan error, 3)
	errCh <- fmt.Errorf("error1")
	errCh <- fmt.Errorf("error2")
	errCh <- fmt.Errorf("error3")
	close(errCh)
	result = aggregateErrors(errCh)
	assert.Error(t, result)
	assert.Contains(t, result.Error(), "error1")
	assert.Contains(t, result.Error(), "error2")
	assert.Contains(t, result.Error(), "error3")

	// Test with nil errors
	errCh = make(chan error, 3)
	errCh <- nil
	errCh <- fmt.Errorf("error1")
	errCh <- nil
	close(errCh)
	result = aggregateErrors(errCh)
	assert.Error(t, result)
	assert.Equal(t, "error1", result.Error())
}

func TestOrchestratorJoinSubnet(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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
				},
			},
			Subnet: jtypes.SubnetConfig{Join: true},
		},
	}

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Prepare routing table and DNS records
	indexRoutingTable := map[string]string{"orchestrator": "10.0.0.2"}
	dnsRecords := map[string]string{"orchestrator": "10.0.0.2"}

	t.Run("success", func(t *testing.T) {
		behavior := fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID)
		ch := make(chan struct{}, 1)
		require.NoError(t, orch.super.AddBehavior(behavior, func(msg actor.Envelope) {
			defer msg.Discard()
			ch <- struct{}{}
			resp := SubnetJoinResponse{OK: true}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = orch.handle
			require.NoError(t, orch.actor.Send(reply))
		}))

		o.manifest = jtypes.EnsembleManifest{
			ID:           ensembleID,
			Orchestrator: orch.actor.Handle(),
			Allocations: map[string]jtypes.AllocationManifest{
				"alloc1": {
					ID:     "test-ensemble_alloc1",
					Type:   jtypes.AllocationTypeTask,
					Status: jtypes.AllocationRunning,
				},
			},
		}

		err = o.orchestratorJoinSubnet(indexRoutingTable, dnsRecords)
		assert.NoError(t, err)
		<-ch
	})

	t.Run("error response", func(t *testing.T) {
		behavior := fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID)
		require.NoError(t, orch.super.AddBehavior(behavior, func(msg actor.Envelope) {
			defer msg.Discard()
			resp := SubnetJoinResponse{OK: false, Error: "join failed"}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			reply.To = msg.From
			reply.From = orch.handle
			require.NoError(t, orch.actor.Send(reply))
		}))

		err := o.orchestratorJoinSubnet(indexRoutingTable, dnsRecords)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "join failed")
	})

	t.Run("timeout", func(t *testing.T) {
		behavior := fmt.Sprintf(behaviors.SubnetJoinBehavior.DynamicTemplate, ensembleID)
		require.NoError(t, orch.super.AddBehavior(behavior, func(msg actor.Envelope) {
			defer msg.Discard()
			// Do not reply to simulate timeout
		}))
		orchestratorJoinTimeout = 1 * time.Second

		err := o.orchestratorJoinSubnet(indexRoutingTable, dnsRecords)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout joining orchestrator to subnet")
	})
}

func TestEscalateFailure(t *testing.T) {
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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Prepare allocation manifest
	allocManifest := jtypes.AllocationManifest{
		ID:     "test-ensemble_alloc1",
		Type:   jtypes.AllocationTypeTask,
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
	o.supervisor.manifest.Allocations["alloc1"] = allocManifest

	t.Run("success", func(t *testing.T) {
		ch := make(chan struct{}, 1)
		require.NoError(t, provider.actor.AddBehavior(behaviors.AllocationRestartBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			ch <- struct{}{}
			resp := behaviors.AllocationRestartResponse{OK: true}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			require.NoError(t, provider.actor.Send(reply))
		}))

		err := o.supervisor.escalateFailure(allocManifest)
		assert.NoError(t, err)
		<-ch
		assert.Equal(t, 1, o.supervisor.escalations[allocManifest.ID])
	})

	t.Run("error response", func(t *testing.T) {
		require.NoError(t, provider.actor.AddBehavior(behaviors.AllocationRestartBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			resp := behaviors.AllocationRestartResponse{OK: false, Error: "restart failed"}
			reply, err := actor.ReplyTo(msg, resp)
			require.NoError(t, err)
			require.NoError(t, provider.actor.Send(reply))
		}))

		err := o.supervisor.escalateFailure(allocManifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "restart failed")
	})

	t.Run("timeout", func(t *testing.T) {
		require.NoError(t, provider.actor.AddBehavior(behaviors.AllocationRestartBehavior, func(msg actor.Envelope) {
			defer msg.Discard()
			// Do not reply to simulate timeout
		}))

		FailureEscalationTimeout = 1 * time.Second
		err := o.supervisor.escalateFailure(allocManifest)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout waiting for supervisor reply")
	})
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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Initial manifest with one allocation
	initialManifest := jtypes.EnsembleManifest{
		ID:           ensembleID,
		Orchestrator: orch.actor.Handle(),
		Allocations: map[string]jtypes.AllocationManifest{
			"alloc1": {
				ID:     "test-ensemble_alloc1",
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

	// Start supervisor with initial manifest
	go o.supervisor.Supervise(initialManifest)

	// Updated manifest with new allocation
	updatedManifest := jtypes.EnsembleManifest{
		ID:           ensembleID,
		Orchestrator: orch.actor.Handle(),
		Allocations: map[string]jtypes.AllocationManifest{
			"alloc1": {
				ID:     "test-ensemble_alloc1",
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
			"alloc2": {
				ID:     "test-ensemble_alloc2",
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
				Allocations: []string{"alloc1", "alloc2"},
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

		alloc, ok := o.supervisor.getAllocation("alloc1")
		assert.True(t, ok)
		assert.Equal(t, "test-ensemble_alloc1", alloc.ID)

		// alloc2 should not exist yet
		alloc, ok = o.supervisor.getAllocation("alloc2")
		assert.False(t, ok)

		// Update supervisor with new manifest
		o.supervisor.Update(updatedManifest)

		// Wait for healthcheck registration
		select {
		case <-orch.channels[behaviors.RegisterHealthcheckBehavior]:
			// Successfully registered healthcheck for new allocation
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for healthcheck registration")
		}

		// Verify manifest was updated
		alloc, ok = o.supervisor.getAllocation("alloc2")
		assert.True(t, ok)
		assert.Equal(t, "test-ensemble_alloc2", alloc.ID)
	})

	t.Run("update with removed allocation", func(t *testing.T) {
		// Create manifest with alloc2 removed
		removedManifest := jtypes.EnsembleManifest{
			ID:           ensembleID,
			Orchestrator: orch.actor.Handle(),
			Allocations: map[string]jtypes.AllocationManifest{
				"alloc1": {
					ID:     "test-ensemble_alloc1",
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
		o.supervisor.Update(removedManifest)

		// Verify alloc2 is no longer in manifest
		_, ok := o.supervisor.getAllocation("alloc2")
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

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	allocation := jtypes.AllocationManifest{
		ID:     "test-ensemble_alloc1",
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
	o.supervisor.manifest.Allocations["alloc1"] = allocation

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

func TestCheckPermutationEdgeConstraints(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	// Set up test configuration with edge constraints
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
				"node2": {
					Location: jtypes.LocationConstraints{
						Accept: []jtypes.Location{
							{Country: "US"},
						},
					},
					Allocations: []string{"alloc2"},
				},
			},
			Allocations: map[string]jtypes.AllocationConfig{
				"alloc1": {
					Type: jtypes.AllocationTypeService,
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
				"alloc2": {
					Type: jtypes.AllocationTypeService,
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
			Edges: []jtypes.EdgeConstraint{
				{
					S:   "node1",
					T:   "node2",
					RTT: 100,
					BW:  1000,
				},
			},
		},
	}

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Create mock GeoLocator
	mockGeo := geolocation.NewMockGeoLocator()
	// Add test locations
	mockGeo.AddLocation("US", "Los Angeles", 34.0522, -118.2437) // Los Angeles coordinates
	mockGeo.AddLocation("US", "New York", 40.7128, -74.0060)     // New York coordinates
	o.geo = mockGeo

	// Create test bids
	bid1 := jtypes.Bid{
		V1: &jtypes.BidV1{
			EnsembleID: ensembleID,
			NodeID:     "node1",
			Peer:       provider.peerID.String(),
			Location:   jtypes.Location{Country: "US", City: "Los Angeles"},
			Handle:     provider.handle,
		},
	}
	bid2 := jtypes.Bid{
		V1: &jtypes.BidV1{
			EnsembleID: ensembleID,
			NodeID:     "node2",
			Peer:       provider.peerID.String(),
			Location:   jtypes.Location{Country: "US", City: "New York"},
			Handle:     provider.handle,
		},
	}

	t.Run("successful constraint verification", func(t *testing.T) {
		candidate := map[string]jtypes.Bid{
			"node1": bid1,
			"node2": bid2,
		}

		result := o.checkPermutationEdgeConstraints(cfg, candidate)
		assert.True(t, result)
	})

	t.Run("failed constraint verification", func(t *testing.T) {
		candidate := map[string]jtypes.Bid{
			"node1": bid1,
			"node2": bid2,
		}
		cfg.V1.Edges = []jtypes.EdgeConstraint{
			{
				S:   "node1",
				T:   "node2",
				RTT: 1,
				BW:  1000,
			},
		}

		result := o.checkPermutationEdgeConstraints(cfg, candidate)
		assert.False(t, result)
	})

	t.Run("missing node in candidate", func(t *testing.T) {
		candidate := map[string]jtypes.Bid{
			"node1": bid1,
		}

		result := o.checkPermutationEdgeConstraints(cfg, candidate)
		assert.False(t, result)
	})

	t.Run("no edge constraints", func(t *testing.T) {
		candidate := map[string]jtypes.Bid{
			"node1": bid1,
			"node2": bid2,
		}

		cfgNoEdges := jtypes.EnsembleConfig{
			V1: &jtypes.EnsembleConfigV1{
				Nodes:       cfg.V1.Nodes,
				Allocations: cfg.V1.Allocations,
			},
		}

		result := o.checkPermutationEdgeConstraints(cfgNoEdges, candidate)
		assert.True(t, result)
	})

	t.Run("timeout during verification", func(t *testing.T) {
		candidate := map[string]jtypes.Bid{
			"node1": bid1,
			"node2": bid2,
		}

		result := o.checkPermutationEdgeConstraints(cfg, candidate)
		assert.False(t, result)
	})
}

func TestUpdateAllocationStatus(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Initialize manifest
	o.manifest = o.newManifest(cfg)

	// Test updating allocation status
	require.NoError(t, o.updateAllocationStatus("alloc1", jtypes.AllocationRunning))
	assert.Equal(t, jtypes.AllocationRunning, o.manifest.Allocations["alloc1"].Status)

	// Test updating non-existent allocation
	require.Error(t, o.updateAllocationStatus("nonexistent", jtypes.AllocationRunning))
}

func TestUpdateAllocationIP(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Initialize manifest
	o.manifest = o.newManifest(cfg)

	// Test updating allocation IP
	require.NoError(t, o.updateAllocationIP("alloc1", "192.168.1.1"))
	assert.Equal(t, "192.168.1.1", o.manifest.Allocations["alloc1"].PrivAddr)

	// Test updating non-existent allocation
	require.Error(t, o.updateAllocationIP("nonexistent", "192.168.1.1"))
}

func TestGetNonce(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Test getting nonce
	nonce1 := o.getNonce()
	nonce2 := o.getNonce()
	assert.NotEqual(t, nonce1, nonce2)
	assert.Greater(t, nonce2, nonce1)
}

func TestProvisionSubnet(t *testing.T) {
	SubnetCreateTimeout = 1 * time.Second
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	provider.MockDeploymentBehaviors(t)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Initialize manifest
	o.manifest = o.newManifest(cfg)

	// Test provisioning subnet
	subReqs := []subnetRequest{
		{
			handle: provider.handle,
			ip:     "10.0.0.2",
			peerID: "peer1",
			ports:  map[int]int{8080: 8080},
		},
	}
	routingTable := map[string]string{
		"peer1": "10.0.0.2",
	}
	subCreateHandles := []actor.Handle{provider.handle}
	err = o.createSubnet(subReqs, routingTable, subCreateHandles)
	require.NoError(t, err)
	<-provider.channels[fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, "test-ensemble")]
}

func TestIsOnlyTaskManifest(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)

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
					Type: jtypes.AllocationTypeTask,
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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Initialize manifest
	o.manifest = o.newManifest(cfg)

	// Test manifest with only tasks
	assert.True(t, o.isOnlyTaskManifest(o.manifest))

	// Add a service allocation
	cfg.V1.Allocations["alloc2"] = jtypes.AllocationConfig{
		Type: jtypes.AllocationTypeService,
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
	}
	o.manifest = o.newManifest(cfg)

	// Test manifest with mixed allocations
	assert.False(t, o.isOnlyTaskManifest(o.manifest))
}

func TestSubnetAddPeer(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	provider.MockDeploymentBehaviors(t)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Initialize manifest
	o.manifest = o.newManifest(cfg)

	// Test adding peer to subnet
	subReqs := []subnetRequest{
		{
			handle: provider.handle,
			ip:     "10.0.0.2",
			peerID: "peer1",
			ports:  map[int]int{8080: 8080},
		},
	}
	err = o.subnetAddPeer(subReqs)
	require.NoError(t, err)
	<-provider.channels[behaviors.SubnetAddPeerBehavior]
}

func TestAddDNSRecords(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	provider.MockDeploymentBehaviors(t)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Initialize manifest
	o.manifest = o.newManifest(cfg)

	// Test adding DNS records
	subReqs := []subnetRequest{
		{
			handle: provider.handle,
			ip:     "10.0.0.2",
			peerID: "peer1",
			ports:  map[int]int{8080: 8080},
		},
	}
	dnsRecords := map[string]string{
		"alloc1.internal": "10.0.0.2",
	}
	err = o.addDNSRecords(subReqs, dnsRecords)
	require.NoError(t, err)
	<-provider.channels[behaviors.SubnetDNSAddRecordsBehavior]
}

func TestMapPorts(t *testing.T) {
	substrate := network.NewSubstrate()
	orch := MakeOrchestrator(t, substrate)
	provider := MakeProvider(t, substrate)

	provider.MockDeploymentBehaviors(t)

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

	ctx := context.Background()
	fs := afero.NewMemMapFs()

	o, err := NewOrchestrator(ctx, afero.Afero{Fs: fs}, workDir, ensembleID, orch.actor, cfg)
	require.NoError(t, err)

	// Initialize manifest
	o.manifest = o.newManifest(cfg)

	// Test mapping ports
	subReqs := []subnetRequest{
		{
			handle: provider.handle,
			ip:     "10.0.0.2",
			peerID: "peer1",
			ports:  map[int]int{8080: 8080},
		},
	}
	err = o.mapPorts(subReqs)
	require.NoError(t, err)
	<-provider.channels[behaviors.SubnetMapPortBehavior]
}
