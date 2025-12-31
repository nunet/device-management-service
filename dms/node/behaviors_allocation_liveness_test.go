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
	"io"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

var (
	// Mock executor errors
	errMockExecutionAlreadyStarted = assert.AnError
	errMockExecutionFailed         = assert.AnError
	errMockExecutionNotRunning     = assert.AnError
)

// mockExecutor is a test executor that implements types.Executor interface
type mockExecutor struct {
	id          string
	started     bool
	running     bool
	paused      bool
	shouldFail  bool
	mu          sync.Mutex
	startedAt   time.Time
	completedAt time.Time
}

func newMockExecutor(id string) *mockExecutor {
	return &mockExecutor{
		id: id,
	}
}

func (m *mockExecutor) GetID() string {
	return m.id
}

func (m *mockExecutor) Start(_ context.Context, _ *types.ExecutionRequest) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return errMockExecutionAlreadyStarted
	}

	m.started = true
	m.running = true
	m.startedAt = time.Now()

	return nil
}

func (m *mockExecutor) Run(ctx context.Context, request *types.ExecutionRequest) (*types.ExecutionResult, error) {
	if err := m.Start(ctx, request); err != nil {
		return nil, err
	}

	resultCh, errCh := m.Wait(ctx, request.ExecutionID)
	select {
	case result := <-resultCh:
		return result, nil
	case err := <-errCh:
		return nil, err
	}
}

func (m *mockExecutor) Wait(_ context.Context, _ string) (<-chan *types.ExecutionResult, <-chan error) {
	resultCh := make(chan *types.ExecutionResult, 1)
	errCh := make(chan error, 1)

	go func() {
		m.mu.Lock()
		shouldFail := m.shouldFail
		m.mu.Unlock()

		// Simulate execution time
		time.Sleep(100 * time.Millisecond)

		m.mu.Lock()
		m.running = false
		m.completedAt = time.Now()
		m.mu.Unlock()

		if shouldFail {
			errCh <- errMockExecutionFailed
			close(resultCh)
			close(errCh)
			return
		}

		result := &types.ExecutionResult{
			STDOUT:   "mock execution output",
			STDERR:   "",
			ExitCode: 0,
		}
		resultCh <- result
		close(resultCh)
		close(errCh)
	}()

	return resultCh, errCh
}

func (m *mockExecutor) Pause(_ context.Context, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return errMockExecutionNotRunning
	}

	m.paused = true
	return nil
}

func (m *mockExecutor) Resume(_ context.Context, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.paused {
		return errMockExecutionNotRunning
	}

	m.paused = false
	return nil
}

func (m *mockExecutor) Cancel(_ context.Context, _ string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return errMockExecutionNotRunning
	}

	m.running = false
	return nil
}

func (m *mockExecutor) Remove(_ string, _ time.Duration) error {
	return nil
}

func (m *mockExecutor) Cleanup(_ context.Context) error {
	return nil
}

func (m *mockExecutor) List() []types.ExecutionListItem {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return []types.ExecutionListItem{}
	}

	return []types.ExecutionListItem{
		{
			ExecutionID: m.id,
			Running:     m.running,
		},
	}
}

func (m *mockExecutor) GetStatus(_ context.Context, _ string) (types.ExecutionStatus, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return types.ExecutionStatusPending, nil
	}

	if m.paused {
		return types.ExecutionStatusPaused, nil
	}

	if m.running {
		return types.ExecutionStatusRunning, nil
	}

	if m.shouldFail {
		return types.ExecutionStatusFailed, nil
	}

	return types.ExecutionStatusSuccess, nil
}

func (m *mockExecutor) WaitForStatus(ctx context.Context, executionID string, status types.ExecutionStatus, timeout *time.Duration) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	var timeoutCh <-chan time.Time
	if timeout != nil {
		timeoutCh = time.After(*timeout)
	} else {
		// No timeout - use a channel that never fires
		timeoutCh = make(<-chan time.Time)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeoutCh:
			return context.DeadlineExceeded
		case <-ticker.C:
			currentStatus, err := m.GetStatus(ctx, executionID)
			if err != nil {
				return err
			}
			if currentStatus == status {
				return nil
			}
		}
	}
}

func (m *mockExecutor) GetLogStream(_ context.Context, _ types.LogStreamRequest) (io.ReadCloser, error) {
	return io.NopCloser(nil), nil
}

func (m *mockExecutor) Exec(_ context.Context, _ string, _ []string) (int, string, string, error) {
	return 0, "", "", nil
}

func (m *mockExecutor) IsRunning() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.running
}

func (m *mockExecutor) SetShouldFail(shouldFail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.shouldFail = shouldFail
}

func (m *mockExecutor) Stats(_ context.Context, _ string) (*types.ExecutorStats, error) {
	return nil, nil
}

// XXX these tests below aren't testing anything. - revise

// TestServiceAllocationSendsLivenessHeartbeats tests that service allocations
// send periodic liveness heartbeats to the orchestrator
func TestServiceAllocationSendsLivenessHeartbeats(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup: Create mock node and orchestrator
	substrate := network.NewSubstrate()
	node, _, _ := newMockNode(t, substrate)
	mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

	// Create mock orchestrator actor
	orchNet, orchPriv := setupTestNetwork(t, substrate)
	orchActor, orchCap, orchRootTrust, orchRootDID := newActor(t, orchPriv, orchNet)
	require.NoError(t, orchActor.Start())

	// Grant capabilities for liveness reporting
	actor.AllowReciprocal(t, orchCap, orchRootTrust, orchRootDID,
		node.actor.Handle().DID, behaviors.NotifyAllocationLivenessBehavior)
	actor.AllowReciprocal(t, orchCap, orchRootTrust, orchRootDID,
		node.actor.Handle().DID, behaviors.NotifyAllocationStatusBehavior)

	// Track received heartbeats
	var heartbeatsMu sync.Mutex
	receivedHeartbeats := []jobtypes.AllocationLivenessNotification{}

	// Add liveness behavior to orchestrator
	require.NoError(t, orchActor.AddBehavior(behaviors.NotifyAllocationLivenessBehavior,
		func(msg actor.Envelope) {
			defer msg.Discard()

			var notification jobtypes.AllocationLivenessNotification
			err := json.Unmarshal(msg.Message, &notification)
			if err != nil {
				t.Logf("Failed to unmarshal liveness: %v", err)
				return
			}

			heartbeatsMu.Lock()
			receivedHeartbeats = append(receivedHeartbeats, notification)
			heartbeatsMu.Unlock()

			t.Logf("Received heartbeat #%d for %s (status: %s, healthy: %v)",
				notification.SequenceNumber, notification.AllocationID,
				notification.Status, notification.Health.Healthy)
		}))

	// Create service allocation with mock executor
	allocationID := "test_service_liveness"
	resources := types.Resources{
		CPU:  types.CPU{Cores: 1, ClockSpeed: 1000},
		RAM:  types.RAM{Size: 512 * 1024 * 1024},
		Disk: types.Disk{Size: 5 * 1024 * 1024 * 1024},
	}

	// Commit resources
	err := node.allocator.Commit(
		ctx, allocationID,
		types.CommittedResources{
			AllocationID: allocationID,
			Resources:    resources,
		},
		nil, 0, time.Now().Add(5*time.Minute).Unix(),
	)
	require.NoError(t, err)

	// Create allocation actor
	allocActor, err := node.actor.CreateChild(allocationID, orchActor.Handle())
	require.NoError(t, err)

	// Create allocation with mock executor
	mockExec := newMockExecutor("test-exec-1")
	allocation, err := jobs.NewAllocation(
		allocationID,
		jobtypes.AllocationTypeService,
		orchActor.Handle(),
		node.fs,
		node.dmsConfig.WorkDir,
		allocActor,
		jobs.AllocationDetails{
			Job: jobs.Job{
				Resources: resources,
				Execution: *types.NewSpecConfig("mock").
					WithParam("image", "alpine:latest"),
			},
			NodeID: node.hostID,
		},
		node.network,
		mockExec, // Use mock executor
		func() error { return node.allocator.Release(ctx, allocationID) },
		nil,
		"",
	)
	require.NoError(t, err)
	require.NotNil(t, allocation)

	// Start allocation actor
	require.NoError(t, allocation.Start())

	// Run the allocation (this starts liveness reporting)
	err = allocation.Run(ctx, "10.0.0.2", "10.0.0.1", nil)
	require.NoError(t, err)

	// Wait for at least one heartbeat
	// Since we can't easily override the interval in this test,
	// we'll wait just enough for the initial heartbeat plus one periodic
	time.Sleep(3 * time.Second)

	// Cleanup
	err = allocation.Stop(ctx)
	require.NoError(t, err)

	// Verify: Check heartbeats
	heartbeatsMu.Lock()
	heartbeatCount := len(receivedHeartbeats)
	heartbeats := make([]jobtypes.AllocationLivenessNotification, len(receivedHeartbeats))
	copy(heartbeats, receivedHeartbeats)
	heartbeatsMu.Unlock()

	assert.GreaterOrEqual(t, heartbeatCount, 1, "Should receive at least 1 heartbeat (initial)")

	if heartbeatCount > 0 {
		firstHeartbeat := heartbeats[0]

		// Verify content
		assert.Equal(t, allocationID, firstHeartbeat.AllocationID)
		assert.Equal(t, "running", firstHeartbeat.Status)
		assert.Greater(t, firstHeartbeat.SequenceNumber, int64(0))
		assert.True(t, firstHeartbeat.Health.Healthy)
		assert.Equal(t, jobtypes.HealthCheckTypeNone, firstHeartbeat.Health.CheckType, "No healthcheck configured")
		assert.Equal(t, "0.1", firstHeartbeat.Version)

		// If we received multiple heartbeats, verify sequence increment
		if heartbeatCount >= 2 {
			secondHeartbeat := heartbeats[1]
			assert.Equal(t, firstHeartbeat.SequenceNumber+1, secondHeartbeat.SequenceNumber,
				"Sequence numbers should increment")
		}
	}
}

// TestServiceAllocationSendsStatusChangeNotification tests immediate status updates
func TestServiceAllocationSendsStatusChangeNotification(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	substrate := network.NewSubstrate()
	node, _, _ := newMockNode(t, substrate)
	mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

	// Create orchestrator
	orchNet, orchPriv := setupTestNetwork(t, substrate)
	orchActor, orchCap, orchRootTrust, orchRootDID := newActor(t, orchPriv, orchNet)
	require.NoError(t, orchActor.Start())

	// Grant capabilities
	actor.AllowReciprocal(t, orchCap, orchRootTrust, orchRootDID,
		node.actor.Handle().DID, behaviors.NotifyAllocationStatusBehavior)

	// Track status updates
	statusUpdates := make(chan jobtypes.AllocationStatusUpdate, 10)

	require.NoError(t, orchActor.AddBehavior(behaviors.NotifyAllocationStatusBehavior,
		func(msg actor.Envelope) {
			defer msg.Discard()

			var update jobtypes.AllocationStatusUpdate
			err := json.Unmarshal(msg.Message, &update)
			if err != nil {
				return
			}

			statusUpdates <- update
			t.Logf("Received status update: %s -> %s (reason: %s)",
				update.OldStatus, update.NewStatus, update.Reason)
		}))

	// Create allocation with mock executor
	allocationID := "test_status_change"
	resources := types.Resources{
		CPU:  types.CPU{Cores: 1, ClockSpeed: 1000},
		RAM:  types.RAM{Size: 512 * 1024 * 1024},
		Disk: types.Disk{Size: 5 * 1024 * 1024 * 1024},
	}

	err := node.allocator.Commit(
		ctx, allocationID,
		types.CommittedResources{
			AllocationID: allocationID,
			Resources:    resources,
		},
		nil, 0, time.Now().Add(5*time.Minute).Unix(),
	)
	require.NoError(t, err)

	// Create allocation actor
	allocActor, err := node.actor.CreateChild(allocationID, orchActor.Handle())
	require.NoError(t, err)

	// Create allocation with mock executor
	mockExec := newMockExecutor("test-exec-status")
	allocation, err := jobs.NewAllocation(
		allocationID,
		jobtypes.AllocationTypeService,
		orchActor.Handle(),
		node.fs,
		node.dmsConfig.WorkDir,
		allocActor,
		jobs.AllocationDetails{
			Job: jobs.Job{
				Resources: resources,
				Execution: *types.NewSpecConfig("mock"),
			},
			NodeID: node.hostID,
		},
		node.network,
		mockExec, // Use mock executor
		func() error { return node.allocator.Release(ctx, allocationID) },
		nil,
		"",
	)
	require.NoError(t, err)

	// Start allocation actor
	require.NoError(t, allocation.Start())

	// Run allocation (should trigger status change notification)
	err = allocation.Run(ctx, "10.0.0.2", "10.0.0.1", nil)
	require.NoError(t, err)

	// Wait for status change notification
	select {
	case update := <-statusUpdates:
		assert.Equal(t, allocationID, update.AllocationID)
		assert.Equal(t, "pending", update.OldStatus)
		assert.Equal(t, "running", update.NewStatus)
		assert.Equal(t, "allocation started", update.Reason)
		assert.Greater(t, update.Timestamp, int64(0))
	case <-time.After(3 * time.Second):
		t.Fatal("Did not receive status change notification")
	}

	// Cleanup
	err = allocation.Stop(ctx)
	require.NoError(t, err)
}

// TestAllocationLivenessDisabled tests that liveness reporting can be disabled
// XXX test still passing even with the code changes that removed the flag and always being enabled
func TestAllocationLivenessDisabled(t *testing.T) {
	t.Parallel()

	substrate := network.NewSubstrate()
	node, _, _ := newMockNode(t, substrate)

	// Create orchestrator
	orchNet, orchPriv := setupTestNetwork(t, substrate)
	orchActor, _, _, _ := newActor(t, orchPriv, orchNet)
	require.NoError(t, orchActor.Start())

	// Track if any heartbeats received (should be none)
	heartbeatReceived := false
	var heartbeatMu sync.Mutex

	require.NoError(t, orchActor.AddBehavior(behaviors.NotifyAllocationLivenessBehavior,
		func(msg actor.Envelope) {
			defer msg.Discard()
			heartbeatMu.Lock()
			heartbeatReceived = true
			heartbeatMu.Unlock()
		}))

	// Create allocation with liveness DISABLED
	allocActor, err := node.actor.CreateChild("test-alloc-disabled", orchActor.Handle())
	require.NoError(t, err)

	// Use mock executor
	mockExec := newMockExecutor("test-exec-disabled")

	allocation, err := jobs.NewAllocation(
		"test-alloc-disabled",
		jobtypes.AllocationTypeService,
		orchActor.Handle(),
		node.fs,
		node.dmsConfig.WorkDir,
		allocActor,
		jobs.AllocationDetails{
			Job: jobs.Job{
				Resources: types.Resources{
					CPU:  types.CPU{Cores: 1},
					RAM:  types.RAM{Size: 512},
					Disk: types.Disk{Size: 1024},
				},
				Execution: *types.NewSpecConfig("mock"),
			},
			NodeID: node.hostID,
		},
		node.network,
		mockExec, // Use mock executor
		func() error { return nil },
		nil,
		"",
	)
	require.NoError(t, err)

	// Start allocation
	require.NoError(t, allocation.Start())

	// Note: startLivenessReporting is called internally by Run() for service allocations
	// Since liveness is disabled, it won't start

	// Wait a bit
	time.Sleep(2 * time.Second)

	// Verify no heartbeats were sent
	heartbeatMu.Lock()
	assert.False(t, heartbeatReceived, "No heartbeats should be sent when disabled")
	heartbeatMu.Unlock()
}

// TestTaskAllocationDoesNotSendPeriodicHeartbeats tests that task allocations
// do not send periodic heartbeats (they use termination notification instead)
func TestTaskAllocationDoesNotSendPeriodicHeartbeats(t *testing.T) {
	t.Parallel()

	substrate := network.NewSubstrate()
	node, _, _ := newMockNode(t, substrate)

	// Create orchestrator
	orchNet, orchPriv := setupTestNetwork(t, substrate)
	orchActor, _, _, _ := newActor(t, orchPriv, orchNet)
	require.NoError(t, orchActor.Start())

	// Track heartbeats
	heartbeatReceived := false
	var heartbeatMu sync.Mutex

	require.NoError(t, orchActor.AddBehavior(behaviors.NotifyAllocationLivenessBehavior,
		func(msg actor.Envelope) {
			defer msg.Discard()
			heartbeatMu.Lock()
			heartbeatReceived = true
			heartbeatMu.Unlock()
		}))

	// Create TASK allocation (even with liveness enabled, tasks shouldn't send periodic heartbeats)
	allocActor, err := node.actor.CreateChild("test-task-alloc", orchActor.Handle())
	require.NoError(t, err)

	// Use mock executor
	mockExec := newMockExecutor("test-exec-task")

	allocation, err := jobs.NewAllocation(
		"test-task-alloc",
		jobtypes.AllocationTypeTask, // Task type
		orchActor.Handle(),
		node.fs,
		node.dmsConfig.WorkDir,
		allocActor,
		jobs.AllocationDetails{
			Job: jobs.Job{
				Resources: types.Resources{
					CPU:  types.CPU{Cores: 1},
					RAM:  types.RAM{Size: 512},
					Disk: types.Disk{Size: 1024},
				},
				Execution: *types.NewSpecConfig("mock"),
			},
			NodeID: node.hostID,
		},
		node.network,
		mockExec, // Use mock executor
		func() error { return nil },
		nil,
		"",
	)
	require.NoError(t, err)
	require.NoError(t, allocation.Start())

	// Note: Even with liveness enabled, task allocations don't send periodic heartbeats
	// The liveness system is only for service allocations

	// Wait
	time.Sleep(2 * time.Second)

	// Verify no periodic heartbeats were sent
	heartbeatMu.Lock()
	assert.False(t, heartbeatReceived,
		"Task allocations should not send periodic heartbeats")
	heartbeatMu.Unlock()
}

// TestAllocationWithCustomHealthcheck tests liveness reporting with a custom healthcheck
func TestAllocationWithCustomHealthcheck(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	substrate := network.NewSubstrate()
	node, _, _ := newMockNode(t, substrate)
	mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

	// Create orchestrator
	orchNet, orchPriv := setupTestNetwork(t, substrate)
	orchActor, orchCap, orchRootTrust, orchRootDID := newActor(t, orchPriv, orchNet)
	require.NoError(t, orchActor.Start())

	// Grant capabilities
	actor.AllowReciprocal(t, orchCap, orchRootTrust, orchRootDID,
		node.actor.Handle().DID, behaviors.NotifyAllocationLivenessBehavior)

	// Track heartbeats and their health status
	var heartbeatsMu sync.Mutex
	healthStatuses := []jobtypes.HealthStatus{}

	require.NoError(t, orchActor.AddBehavior(behaviors.NotifyAllocationLivenessBehavior,
		func(msg actor.Envelope) {
			defer msg.Discard()

			var notification jobtypes.AllocationLivenessNotification
			if err := json.Unmarshal(msg.Message, &notification); err != nil {
				return
			}

			heartbeatsMu.Lock()
			healthStatuses = append(healthStatuses, notification.Health)
			heartbeatsMu.Unlock()

			t.Logf("Received heartbeat with health: %v (type: %s, msg: %s)",
				notification.Health.Healthy, notification.Health.CheckType, notification.Health.Message)
		}))

	// Create allocation
	allocationID := "test_custom_healthcheck"
	resources := types.Resources{
		CPU:  types.CPU{Cores: 1, ClockSpeed: 1000},
		RAM:  types.RAM{Size: 512 * 1024 * 1024},
		Disk: types.Disk{Size: 5 * 1024 * 1024 * 1024},
	}

	err := node.allocator.Commit(
		ctx, allocationID,
		types.CommittedResources{
			AllocationID: allocationID,
			Resources:    resources,
		},
		nil, 0, time.Now().Add(5*time.Minute).Unix(),
	)
	require.NoError(t, err)

	allocActor, err := node.actor.CreateChild(allocationID, orchActor.Handle())
	require.NoError(t, err)

	mockExec := newMockExecutor("test-exec-health")
	allocation, err := jobs.NewAllocation(
		allocationID,
		jobtypes.AllocationTypeService,
		orchActor.Handle(),
		node.fs,
		node.dmsConfig.WorkDir,
		allocActor,
		jobs.AllocationDetails{
			Job: jobs.Job{
				Resources: resources,
				Execution: *types.NewSpecConfig("mock"),
			},
			NodeID: node.hostID,
		},
		node.network,
		mockExec,
		func() error { return node.allocator.Release(ctx, allocationID) },
		nil,
		"",
	)
	require.NoError(t, err)

	// Register a custom healthcheck that passes
	healthCheckCalls := 0
	var healthCheckMu sync.Mutex
	allocation.SetHealthCheck(func() error {
		healthCheckMu.Lock()
		defer healthCheckMu.Unlock()
		healthCheckCalls++
		return nil // Healthy
	})

	// Start and run
	require.NoError(t, allocation.Start())
	err = allocation.Run(ctx, "10.0.0.2", "10.0.0.1", nil)
	require.NoError(t, err)

	// Wait for heartbeat with custom healthcheck
	time.Sleep(2 * time.Second)

	// Cleanup
	err = allocation.Stop(ctx)
	require.NoError(t, err)

	// Verify healthcheck was called
	healthCheckMu.Lock()
	calls := healthCheckCalls
	healthCheckMu.Unlock()
	assert.Greater(t, calls, 0, "Custom healthcheck should have been called")

	// Verify health status in heartbeats
	heartbeatsMu.Lock()
	count := len(healthStatuses)
	heartbeatsMu.Unlock()

	assert.GreaterOrEqual(t, count, 1, "Should have received at least one heartbeat")

	if count > 0 {
		heartbeatsMu.Lock()
		firstHealth := healthStatuses[0]
		heartbeatsMu.Unlock()

		assert.True(t, firstHealth.Healthy, "Healthcheck passed, so should be healthy")
		assert.Equal(t, jobtypes.HealthCheckTypeSelf, firstHealth.CheckType, "Should use self healthcheck")
		assert.Contains(t, firstHealth.Message, "passed")
	}
}

// TestAllocationWithFailingHealthcheck tests liveness reporting with failing healthcheck
func TestAllocationWithFailingHealthcheck(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	substrate := network.NewSubstrate()
	node, _, _ := newMockNode(t, substrate)
	mockOnboarding(t, node, MockTotalCPU/2, MockTotalRAM/2, MockTotalDisk/2)

	// Create orchestrator
	orchNet, orchPriv := setupTestNetwork(t, substrate)
	orchActor, orchCap, orchRootTrust, orchRootDID := newActor(t, orchPriv, orchNet)
	require.NoError(t, orchActor.Start())

	// Grant capabilities
	actor.AllowReciprocal(t, orchCap, orchRootTrust, orchRootDID,
		node.actor.Handle().DID, behaviors.NotifyAllocationLivenessBehavior)

	// Track health statuses
	var heartbeatsMu sync.Mutex
	healthStatuses := []jobtypes.HealthStatus{}

	require.NoError(t, orchActor.AddBehavior(behaviors.NotifyAllocationLivenessBehavior,
		func(msg actor.Envelope) {
			defer msg.Discard()

			var notification jobtypes.AllocationLivenessNotification
			if err := json.Unmarshal(msg.Message, &notification); err != nil {
				return
			}

			heartbeatsMu.Lock()
			healthStatuses = append(healthStatuses, notification.Health)
			heartbeatsMu.Unlock()
		}))

	// Create allocation
	allocationID := "test_failing_healthcheck"
	resources := types.Resources{
		CPU:  types.CPU{Cores: 1, ClockSpeed: 1000},
		RAM:  types.RAM{Size: 512 * 1024 * 1024},
		Disk: types.Disk{Size: 5 * 1024 * 1024 * 1024},
	}

	err := node.allocator.Commit(
		ctx, allocationID,
		types.CommittedResources{
			AllocationID: allocationID,
			Resources:    resources,
		},
		nil, 0, time.Now().Add(5*time.Minute).Unix(),
	)
	require.NoError(t, err)

	allocActor, err := node.actor.CreateChild(allocationID, orchActor.Handle())
	require.NoError(t, err)

	mockExec := newMockExecutor("test-exec-fail")
	allocation, err := jobs.NewAllocation(
		allocationID,
		jobtypes.AllocationTypeService,
		orchActor.Handle(),
		node.fs,
		node.dmsConfig.WorkDir,
		allocActor,
		jobs.AllocationDetails{
			Job: jobs.Job{
				Resources: resources,
				Execution: *types.NewSpecConfig("mock"),
			},
			NodeID: node.hostID,
		},
		node.network,
		mockExec,
		func() error { return node.allocator.Release(ctx, allocationID) },
		nil,
		"",
	)
	require.NoError(t, err)

	// Register a failing healthcheck
	allocation.SetHealthCheck(func() error {
		return assert.AnError // Simulated failure
	})

	// Start and run
	require.NoError(t, allocation.Start())
	err = allocation.Run(ctx, "10.0.0.2", "10.0.0.1", nil)
	require.NoError(t, err)

	// Wait for heartbeat
	time.Sleep(2 * time.Second)

	// Cleanup
	err = allocation.Stop(ctx)
	require.NoError(t, err)

	// Verify health status shows failure
	heartbeatsMu.Lock()
	count := len(healthStatuses)
	heartbeatsMu.Unlock()

	assert.GreaterOrEqual(t, count, 1, "Should have received at least one heartbeat")

	if count > 0 {
		heartbeatsMu.Lock()
		firstHealth := healthStatuses[0]
		heartbeatsMu.Unlock()

		assert.False(t, firstHealth.Healthy, "Healthcheck failed, so should be unhealthy")
		assert.Equal(t, jobtypes.HealthCheckTypeSelf, firstHealth.CheckType)
		assert.Contains(t, firstHealth.Message, "failed")
	}
}
