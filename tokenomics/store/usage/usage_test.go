// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package usage

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/document"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/events"
)

func TestAddUsageEventAndGetAllEvents(t *testing.T) {
	store := setupTestDB(t)

	usage := Usage{
		ContractDID: "contract-123",
		Data:        []byte("sample data"),
	}

	err := store.AddUsageEvent(usage)
	require.NoError(t, err, "failed to add usage")

	events, err := store.GetAllEvents()
	require.NoError(t, err, "failed to get all events")
	require.Len(t, events, 1, "expected 1 event")

	assert.Equal(t, usage.ContractDID, events[0].ContractDID, "ContractDID should match")
	assert.Equal(t, string(usage.Data), string(events[0].Data), "Data should match")
}

func TestGetEventsByContract(t *testing.T) {
	store := setupTestDB(t)

	contract1 := "contract-111"
	contract2 := "contract-222"

	_ = store.AddUsageEvent(Usage{ContractDID: contract1, Data: []byte("event-1")})
	_ = store.AddUsageEvent(Usage{ContractDID: contract1, Data: []byte("event-2")})
	_ = store.AddUsageEvent(Usage{ContractDID: contract2, Data: []byte("event-3")})

	events1, err := store.GetEventsByContract(contract1)
	require.NoError(t, err, "failed to get events by contract")
	require.Len(t, events1, 2, "expected 2 events for contract1")

	events2, err := store.GetEventsByContract(contract2)
	require.NoError(t, err, "failed to get events by contract")
	require.Len(t, events2, 1, "expected 1 event for contract2")
}

func TestGetEventsByDateRange(t *testing.T) {
	store := setupTestDB(t)

	firstUsage := Usage{ContractDID: "contract-1", Data: []byte("data-1")}
	secondUsage := Usage{ContractDID: "contract-2", Data: []byte("data-2")}

	err := store.AddUsageEvent(firstUsage)
	require.NoError(t, err, "failed to insert first usage")

	time.Sleep(1 * time.Second)

	err = store.AddUsageEvent(secondUsage)
	require.NoError(t, err, "failed to insert second usage")

	start := time.Now().Add(-500 * time.Millisecond)
	end := time.Now().Add(1 * time.Second)

	events, err := store.GetEventsByDateRange(start, end)
	require.NoError(t, err, "failed to get events by date range")
	require.Len(t, events, 1, "expected 1 event")

	assert.Equal(t, secondUsage.ContractDID, events[0].ContractDID, "ContractDID should match")
}

func TestCountAllocationsByContract(t *testing.T) {
	store := setupTestDB(t)

	startEvent1 := []byte(`{"type":"START_ALLOCATION_EVENT","allocation_id":"a1"}`)
	startEvent2 := []byte(`{"type":"START_ALLOCATION_EVENT","allocation_id":"a2"}`)
	completeEvent := []byte(`{"type":"COMPLETE_ALLOCATION_EVENT","allocation_id":"a1"}`)
	startEvent3 := []byte(`{"type":"START_ALLOCATION_EVENT","allocation_id":"a3"}`)

	usages := []Usage{
		{ContractDID: "contract-123", Data: startEvent1},   // should count (unique a1)
		{ContractDID: "contract-123", Data: completeEvent}, // should NOT count (not START event)
		{ContractDID: "contract-123", Data: startEvent2},   // should count (unique a2)
		{ContractDID: "contract-456", Data: startEvent3},   // should count (unique a3)
	}

	for _, u := range usages {
		err := store.AddUsageEvent(u)
		require.NoError(t, err, "failed to insert usage")
	}

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(1 * time.Hour)

	counts, err := store.CountAllocationsByContract(start, end)
	require.NoError(t, err, "CountAllocationsByContract failed")

	assert.Equal(t, 2, counts["contract-123"], "expected 2 allocations for contract-123")
	assert.Equal(t, 1, counts["contract-456"], "expected 1 allocation for contract-456")
	_, ok := counts["contract-789"]
	assert.False(t, ok, "did not expect any count for contract-789")
}

func TestSaveAndGetLastProcessedAt(t *testing.T) {
	store := setupTestDB(t)

	// Test global timestamp
	ts, err := store.GetLastProcessedAt("")
	require.NoError(t, err, "unexpected error")
	assert.True(t, ts.Equal(time.Unix(0, 0)), "expected initial timestamp to be Unix(0)")

	now := time.Now().Truncate(time.Second) // truncate for equality
	err = store.SaveLastProcessedAt("", now)
	require.NoError(t, err, "failed to save last processed at")

	ts, err = store.GetLastProcessedAt("")
	require.NoError(t, err, "failed to get last processed at")
	assert.True(t, ts.Equal(now), "expected timestamp to match")

	newer := now.Add(15 * time.Minute).Truncate(time.Second)
	err = store.SaveLastProcessedAt("", newer)
	require.NoError(t, err, "failed to update last processed at")

	ts, err = store.GetLastProcessedAt("")
	require.NoError(t, err, "failed to get updated last processed at")
	assert.True(t, ts.Equal(newer), "expected updated timestamp to match")

	// Test contract-specific timestamp
	contractDID := "did:key:test123"
	contractTime := now.Add(30 * time.Minute).Truncate(time.Second)
	err = store.SaveLastProcessedAt(contractDID, contractTime)
	require.NoError(t, err, "failed to save contract-specific last processed at")

	ts, err = store.GetLastProcessedAt(contractDID)
	require.NoError(t, err, "failed to get contract-specific last processed at")
	assert.True(t, ts.Equal(contractTime), "expected contract timestamp to match")

	// Verify global timestamp is unchanged
	ts, err = store.GetLastProcessedAt("")
	require.NoError(t, err, "failed to get global last processed at")
	assert.True(t, ts.Equal(newer), "expected global timestamp to remain unchanged")
}

func TestCalculateTimeUtilizationByContract_TaskAllocation(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"  //nolint:goconst
	deploymentID := "deployment-1" //nolint:goconst
	allocationID := "allocation-task-1"

	baseTime := time.Now().Truncate(time.Second)
	startTime := baseTime.Add(10 * time.Second)
	completeTime := baseTime.Add(70 * time.Second) // 60 seconds duration

	// Add Start event
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(allocationID, deploymentID), startTime)

	// Add Complete event
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocationID, deploymentID), completeTime)

	// Calculate utilization
	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateTimeUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateTimeUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	assert.Equal(t, deploymentID, result.DeploymentID, "deployment ID should match")
	require.Len(t, result.Allocations, 1, "expected 1 allocation")

	alloc := result.Allocations[0]
	assert.Equal(t, allocationID, alloc.AllocationID, "allocation ID should match")

	expectedDuration := 60 * time.Second
	assert.Equal(t, expectedDuration, alloc.Duration, "duration should match")
	assert.Equal(t, startTime.Unix(), alloc.StartTime.Unix(), "start time should match")
	assert.False(t, alloc.EndTime.IsZero(), "EndTime should be set for completed allocation")
	assert.Equal(t, completeTime.Unix(), alloc.EndTime.Unix(), "end time should match")
	assert.Equal(t, 60.0, result.TotalUtilizationSec, "total utilization should be 60.0 seconds")
}

func TestCalculateTimeUtilizationByContract_ServiceAllocation(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocationID := "allocation-service-1"

	baseTime := time.Now().Truncate(time.Second)
	startTime := baseTime.Add(10 * time.Second)
	stopTime := baseTime.Add(130 * time.Second) // 120 seconds duration

	// Add Start event
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(allocationID, deploymentID), startTime)

	// Add Stop event
	addEventWithTimestamp(t, store, contractDID, events.StopAllocationEvent,
		createStopAllocationEvent(allocationID, deploymentID), stopTime)

	// Calculate utilization
	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateTimeUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateTimeUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 1, "expected 1 allocation")

	alloc := result.Allocations[0]
	expectedDuration := 120 * time.Second
	assert.Equal(t, expectedDuration, alloc.Duration, "duration should match")
	assert.False(t, alloc.EndTime.IsZero(), "EndTime should be set for stopped service allocation")
	assert.Equal(t, stopTime.Unix(), alloc.EndTime.Unix(), "end time should match")
	assert.Equal(t, 120.0, result.TotalUtilizationSec, "total utilization should be 120.0 seconds")
}

func TestCalculateTimeUtilizationByContract_RunningAllocation(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocationID := "allocation-running-1"

	// Set baseTime in the past so that when we query, time.Now() will definitely be after startTime
	baseTime := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	startTime := baseTime.Add(10 * time.Second) // Allocation started 10 seconds after baseTime (in the past)
	// No end event - allocation is still running

	// Add Start event only (with timestamp in the past)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(allocationID, deploymentID), startTime)

	// Capture current time before calling the function
	// (the function will use time.Now() internally for running allocations)
	beforeCallTime := time.Now()

	// Calculate utilization - query period starts at baseTime
	start := baseTime
	end := baseTime.Add(3 * time.Hour) // 3 hours later - used for query period only
	results, err := store.CalculateTimeUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateTimeUtilizationByContract failed")

	// Capture time after the call to get bounds for time.Now() used inside
	afterCallTime := time.Now()

	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 1, "expected 1 allocation")

	alloc := result.Allocations[0]

	// Since the function now uses time.Now() instead of end, the duration will be
	// from startTime to whenever time.Now() was called inside the function
	// We verify it's within reasonable bounds (between beforeCallTime and afterCallTime)
	minExpectedDuration := beforeCallTime.Sub(startTime)
	maxExpectedDuration := afterCallTime.Sub(startTime)

	assert.True(t, alloc.Duration >= minExpectedDuration,
		"duration should be at least from startTime to beforeCallTime (got %v, expected at least %v)",
		alloc.Duration, minExpectedDuration)
	assert.True(t, alloc.Duration <= maxExpectedDuration,
		"duration should be at most from startTime to afterCallTime (got %v, expected at most %v)",
		alloc.Duration, maxExpectedDuration)
	assert.True(t, alloc.EndTime.IsZero(), "EndTime should be zero for running allocation")
	assert.Equal(t, startTime, alloc.StartTime, "StartTime should match the event timestamp")

	// Total utilization should match the allocation duration
	assert.InDelta(t, alloc.Duration.Seconds(), result.TotalUtilizationSec, 1.0,
		"total utilization should match allocation duration")
}

func TestCalculateTimeUtilizationByContract_MultipleDeployments(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deployment1ID := "deployment-1"
	deployment2ID := "deployment-2" //nolint:goconst
	allocation1ID := "allocation-1" //nolint:goconst
	allocation2ID := "allocation-2" //nolint:goconst

	baseTime := time.Now().Truncate(time.Second)

	// Deployment 1: task allocation (60 seconds)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(allocation1ID, deployment1ID), baseTime.Add(10*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocation1ID, deployment1ID), baseTime.Add(70*time.Second))

	// Deployment 2: service allocation (120 seconds)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(allocation2ID, deployment2ID), baseTime.Add(20*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.StopAllocationEvent,
		createStopAllocationEvent(allocation2ID, deployment2ID), baseTime.Add(140*time.Second))

	// Calculate utilization
	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateTimeUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateTimeUtilizationByContract failed")
	require.Len(t, results, 2, "expected 2 deployments")

	// Find deployments by ID
	var dep1, dep2 *contracts.DeploymentTimeUtilization
	for i := range results {
		if results[i].DeploymentID == deployment1ID { //nolint:staticcheck
			dep1 = &results[i]
		} else if results[i].DeploymentID == deployment2ID {
			dep2 = &results[i]
		}
	}

	require.NotNil(t, dep1, "deployment 1 not found")
	require.NotNil(t, dep2, "deployment 2 not found")

	assert.Len(t, dep1.Allocations, 1, "deployment 1: expected 1 allocation")
	assert.Len(t, dep2.Allocations, 1, "deployment 2: expected 1 allocation")
	assert.Equal(t, 60.0, dep1.TotalUtilizationSec, "deployment 1: expected 60.0 seconds")
	assert.Equal(t, 120.0, dep2.TotalUtilizationSec, "deployment 2: expected 120.0 seconds")
}

func TestCalculateTimeUtilizationByContract_MultipleAllocationsPerDeployment(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	taskAllocID := "allocation-task"
	serviceAllocID := "allocation-service" //nolint:goconst

	baseTime := time.Now().Truncate(time.Second)

	// Task allocation: 60 seconds
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(taskAllocID, deploymentID), baseTime.Add(10*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(taskAllocID, deploymentID), baseTime.Add(70*time.Second))

	// Service allocation: 120 seconds
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(serviceAllocID, deploymentID), baseTime.Add(20*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.StopAllocationEvent,
		createStopAllocationEvent(serviceAllocID, deploymentID), baseTime.Add(140*time.Second))

	// Calculate utilization
	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateTimeUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateTimeUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 2, "expected 2 allocations")

	expectedTotalSec := 180.0 // 60 + 120
	assert.Equal(t, expectedTotalSec, result.TotalUtilizationSec, "expected total utilization 180.0 seconds")
}

func TestCalculateTimeUtilizationByContract_EmptyResult(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"

	// No events added

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(1 * time.Hour)
	results, err := store.CalculateTimeUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateTimeUtilizationByContract failed")
	assert.Len(t, results, 0, "expected 0 deployments")
}

func TestCalculateTimeUtilizationByContract_TimeRangeFiltering(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deployment1ID := "deployment-1"
	deployment2ID := "deployment-2"
	allocationID := "allocation-1"

	baseTime := time.Now().Truncate(time.Second)

	// Event before the start time - should not be included (different deployment to avoid confusion)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(allocationID, deployment1ID), baseTime.Add(-2*time.Hour))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocationID, deployment1ID), baseTime.Add(-1*time.Hour))

	// Event within the range - should be included
	newAllocID := "allocation-2"
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(newAllocID, deployment2ID), baseTime.Add(10*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(newAllocID, deployment2ID), baseTime.Add(70*time.Second))

	// Calculate utilization with time range that excludes first event
	// The first event starts at baseTime-2h, which is before the query start time (baseTime)
	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateTimeUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateTimeUtilizationByContract failed")

	// We should only get deployment2, since deployment1's start event is before the query start time
	// However, if deployment1's complete event is queried, we might still get it
	// So we check that deployment2 is present and has the correct allocation
	foundDeployment2 := false
	for _, result := range results {
		if result.DeploymentID == deployment2ID {
			foundDeployment2 = true
			assert.Len(t, result.Allocations, 1, "deployment 2: expected 1 allocation")
			assert.Equal(t, newAllocID, result.Allocations[0].AllocationID, "deployment 2: allocation ID should match")
		}
	}

	assert.True(t, foundDeployment2, "expected to find deployment %s in results", deployment2ID)
}

func TestCalculateTimeUtilizationByContract_ContractIsolation(t *testing.T) {
	store := setupTestDB(t)
	contractDID1 := "contract-123"
	contractDID2 := "contract-456"
	deployment1ID := "deployment-1"
	deployment2ID := "deployment-2"
	allocation1ID := "allocation-1"
	allocation2ID := "allocation-2"

	baseTime := time.Now().Truncate(time.Second)

	// Add events for contract 1
	addEventWithTimestamp(t, store, contractDID1, events.StartAllocationEvent,
		createStartAllocationEvent(allocation1ID, deployment1ID), baseTime.Add(10*time.Second))
	addEventWithTimestamp(t, store, contractDID1, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocation1ID, deployment1ID), baseTime.Add(70*time.Second))

	// Add events for contract 2
	addEventWithTimestamp(t, store, contractDID2, events.StartAllocationEvent,
		createStartAllocationEvent(allocation2ID, deployment2ID), baseTime.Add(10*time.Second))
	addEventWithTimestamp(t, store, contractDID2, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocation2ID, deployment2ID), baseTime.Add(70*time.Second))

	// Query for contract 1 - should only get contract 1's deployment
	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results1, err := store.CalculateTimeUtilizationByContract(contractDID1, start, end)
	require.NoError(t, err, "CalculateTimeUtilizationByContract failed for contract 1")

	// Verify contract 1 results - should only have its own deployment
	require.Len(t, results1, 1, "contract 1: expected 1 deployment")
	assert.Equal(t, deployment1ID, results1[0].DeploymentID, "contract 1: deployment ID should match")
	require.Len(t, results1[0].Allocations, 1, "contract 1: expected 1 allocation")
	assert.Equal(t, allocation1ID, results1[0].Allocations[0].AllocationID, "contract 1: allocation ID should match")

	// Query for contract 2 - should only get contract 2's deployment
	results2, err := store.CalculateTimeUtilizationByContract(contractDID2, start, end)
	require.NoError(t, err, "CalculateTimeUtilizationByContract failed for contract 2")

	// Verify contract 2 results - should only have its own deployment
	require.Len(t, results2, 1, "contract 2: expected 1 deployment")
	assert.Equal(t, deployment2ID, results2[0].DeploymentID, "contract 2: deployment ID should match")
	require.Len(t, results2[0].Allocations, 1, "contract 2: expected 1 allocation")
	assert.Equal(t, allocation2ID, results2[0].Allocations[0].AllocationID, "contract 2: allocation ID should match")

	// Verify isolation - contract 1 results shouldn't contain contract 2's deployment
	for _, result := range results1 {
		assert.NotEqual(t, deployment2ID, result.DeploymentID, "contract 1 results should not contain contract 2's deployment")
	}
	// Verify isolation - contract 2 results shouldn't contain contract 1's deployment
	for _, result := range results2 {
		assert.NotEqual(t, deployment1ID, result.DeploymentID, "contract 2 results should not contain contract 1's deployment")
	}
}

func TestCalculateTimeUtilizationByContract_MixedTaskAndService(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	taskAllocID := "task-allocation"
	serviceAllocID := "service-allocation"

	baseTime := time.Now().Truncate(time.Second)

	// Task allocation with CompleteAllocationEvent
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(taskAllocID, deploymentID), baseTime.Add(10*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(taskAllocID, deploymentID), baseTime.Add(70*time.Second))

	// Service allocation with StopAllocationEvent
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(serviceAllocID, deploymentID), baseTime.Add(20*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.StopAllocationEvent,
		createStopAllocationEvent(serviceAllocID, deploymentID), baseTime.Add(140*time.Second))

	// Calculate utilization
	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateTimeUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateTimeUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 2, "expected 2 allocations")

	// Verify both allocations are included
	allocIDs := make(map[string]bool)
	for _, alloc := range result.Allocations {
		allocIDs[alloc.AllocationID] = true
	}

	assert.True(t, allocIDs[taskAllocID], "task allocation should be found in results")
	assert.True(t, allocIDs[serviceAllocID], "service allocation should be found in results")

	// Total should be 60 + 120 = 180 seconds
	expectedTotalSec := 180.0
	assert.Equal(t, expectedTotalSec, result.TotalUtilizationSec, "expected total utilization 180.0 seconds")
}

func TestCalculateTimeUtilizationByContract_ContinuingFromPreviousPeriod(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocationID := "allocation-service"

	baseTime := time.Now().Truncate(time.Second)

	// Add StartAllocationEvent BEFORE the query period (simulating a running allocation from previous invoice)
	startTime := baseTime.Add(-10 * time.Minute) // Started 10 minutes before query period
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(allocationID, deploymentID), startTime)

	// Query for time period starting later (simulating second invoice)
	// This allocation should be included because it's still running
	queryStart := baseTime                    // Query starts now
	queryEnd := baseTime.Add(2 * time.Minute) // Query ends 2 minutes later

	// Capture current time before calling the function
	beforeCallTime := time.Now()

	results, err := store.CalculateTimeUtilizationByContract(contractDID, queryStart, queryEnd)
	require.NoError(t, err, "CalculateTimeUtilizationByContract failed")

	// Capture time after the call to get bounds for time.Now() used inside
	afterCallTime := time.Now()

	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 1, "expected 1 allocation")

	alloc := result.Allocations[0]
	assert.Equal(t, allocationID, alloc.AllocationID, "allocation ID should match")

	// The allocation started 10 minutes before queryStart, but we should only count time from queryStart
	// Since the function uses time.Now(), duration will be from queryStart to time.Now()
	minExpectedDuration := beforeCallTime.Sub(queryStart)
	maxExpectedDuration := afterCallTime.Sub(queryStart)

	assert.True(t, alloc.Duration >= minExpectedDuration,
		"duration should be at least from queryStart to beforeCallTime")
	assert.True(t, alloc.Duration <= maxExpectedDuration,
		"duration should be at most from queryStart to afterCallTime")
	assert.True(t, alloc.EndTime.IsZero(), "EndTime should be zero for running allocation")

	// Total utilization should match the allocation duration
	assert.InDelta(t, alloc.Duration.Seconds(), result.TotalUtilizationSec, 1.0,
		"total utilization should match allocation duration")
}

func TestCalculateTimeUtilizationByContract_StoppedInQueryPeriod(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocationID := "allocation-service"

	baseTime := time.Now().Truncate(time.Second)

	// Add StartAllocationEvent BEFORE the query period
	startTime := baseTime.Add(-10 * time.Minute)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(allocationID, deploymentID), startTime)

	// Add StopAllocationEvent WITHIN the query period
	stopTime := baseTime.Add(1 * time.Minute)
	addEventWithTimestamp(t, store, contractDID, events.StopAllocationEvent,
		createStopAllocationEvent(allocationID, deploymentID), stopTime)

	// Query for time period
	queryStart := baseTime
	queryEnd := baseTime.Add(2 * time.Minute)

	results, err := store.CalculateTimeUtilizationByContract(contractDID, queryStart, queryEnd)
	require.NoError(t, err, "CalculateTimeUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 1, "expected 1 allocation")

	alloc := result.Allocations[0]

	// Duration should be from queryStart to stopTime (1 minute), not from startTime
	expectedDuration := stopTime.Sub(queryStart)
	assert.Equal(t, expectedDuration, alloc.Duration, "duration should be from queryStart to stopTime")
	assert.False(t, alloc.EndTime.IsZero(), "EndTime should be set")
	assert.Equal(t, stopTime.Unix(), alloc.EndTime.Unix(), "end time should match")

	// Total utilization should be 1 minute (60 seconds)
	assert.Equal(t, 60.0, result.TotalUtilizationSec, "total utilization should be 60.0 seconds")
}

func TestCalculateTimeUtilizationByContract_ExcludedIfEndedBeforeQueryPeriod(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocationID := "allocation-service"

	baseTime := time.Now().Truncate(time.Second)

	// Add StartAllocationEvent BEFORE the query period
	startTime := baseTime.Add(-10 * time.Minute)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEvent(allocationID, deploymentID), startTime)

	// Add StopAllocationEvent BEFORE the query period (allocation ended before query period)
	stopTime := baseTime.Add(-5 * time.Minute)
	addEventWithTimestamp(t, store, contractDID, events.StopAllocationEvent,
		createStopAllocationEvent(allocationID, deploymentID), stopTime)

	// Query for time period that starts AFTER the allocation ended
	queryStart := baseTime
	queryEnd := baseTime.Add(2 * time.Minute)

	results, err := store.CalculateTimeUtilizationByContract(contractDID, queryStart, queryEnd)
	require.NoError(t, err, "CalculateTimeUtilizationByContract failed")

	// Should have no deployments because the allocation ended before the query period
	assert.Len(t, results, 0, "expected 0 deployments (allocation ended before query period)")
}

// ============================================================================
// CalculateResourceUtilizationByContract Unit Tests
// ============================================================================

func TestCalculateResourceUtilizationByContract_TaskAllocation(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocationID := "allocation-task-1"

	baseTime := time.Now().Truncate(time.Second)
	startTime := baseTime.Add(10 * time.Second)
	completeTime := baseTime.Add(70 * time.Second) // 60 seconds duration

	// Add Start event with resources (2 cores, 4GB RAM, 10GB Disk)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocationID, deploymentID, 2, 4, 10, 0), startTime)

	// Add Complete event
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocationID, deploymentID), completeTime)

	// Calculate utilization
	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateResourceUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	assert.Equal(t, deploymentID, result.DeploymentID, "deployment ID should match")
	require.Len(t, result.Allocations, 1, "expected 1 allocation")

	alloc := result.Allocations[0]
	assert.Equal(t, allocationID, alloc.AllocationID, "allocation ID should match")
	assert.Equal(t, float32(2), alloc.Resources.CPU.Cores, "CPU cores should match")
	assert.Equal(t, uint64(4*1e9), alloc.Resources.RAM.Size, "RAM size should match")
	assert.Equal(t, uint64(10*1e9), alloc.Resources.Disk.Size, "Disk size should match")

	expectedDuration := 60 * time.Second
	assert.Equal(t, expectedDuration, alloc.Duration, "duration should match")
	assert.Equal(t, startTime.Unix(), alloc.StartTime.Unix(), "start time should match")
	assert.False(t, alloc.EndTime.IsZero(), "EndTime should be set for completed allocation")
	assert.Equal(t, completeTime.Unix(), alloc.EndTime.Unix(), "end time should match")
	assert.Equal(t, 60.0, result.TotalUtilizationSec, "total utilization should be 60.0 seconds")
}

func TestCalculateResourceUtilizationByContract_ServiceAllocation(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocationID := "allocation-service-1"

	baseTime := time.Now().Truncate(time.Second)
	startTime := baseTime.Add(10 * time.Second)
	stopTime := baseTime.Add(130 * time.Second) // 120 seconds duration

	// Add Start event with resources (4 cores, 8GB RAM, 20GB Disk)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocationID, deploymentID, 4, 8, 20, 0), startTime)

	// Add Stop event
	addEventWithTimestamp(t, store, contractDID, events.StopAllocationEvent,
		createStopAllocationEvent(allocationID, deploymentID), stopTime)

	// Calculate utilization
	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateResourceUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 1, "expected 1 allocation")

	alloc := result.Allocations[0]
	assert.Equal(t, float32(4), alloc.Resources.CPU.Cores, "CPU cores should match")
	assert.Equal(t, uint64(8*1e9), alloc.Resources.RAM.Size, "RAM size should match")
	assert.Equal(t, uint64(20*1e9), alloc.Resources.Disk.Size, "Disk size should match")

	expectedDuration := 120 * time.Second
	assert.Equal(t, expectedDuration, alloc.Duration, "duration should match")
	assert.False(t, alloc.EndTime.IsZero(), "EndTime should be set for stopped service allocation")
	assert.Equal(t, stopTime.Unix(), alloc.EndTime.Unix(), "end time should match")
	assert.Equal(t, 120.0, result.TotalUtilizationSec, "total utilization should be 120.0 seconds")
}

func TestCalculateResourceUtilizationByContract_RunningAllocation(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocationID := "allocation-running-1"

	// Set baseTime in the past so that when we query, time.Now() will definitely be after startTime
	baseTime := time.Now().Add(-2 * time.Hour).Truncate(time.Second)
	startTime := baseTime.Add(10 * time.Second) // Allocation started 10 seconds after baseTime (in the past)
	// No end event - allocation is still running

	// Add Start event with resources (1 core, 2GB RAM, 5GB Disk)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocationID, deploymentID, 1, 2, 5, 0), startTime)

	// Capture current time before calling the function
	beforeCallTime := time.Now()

	// Calculate utilization - query period starts at baseTime
	start := baseTime
	end := baseTime.Add(3 * time.Hour) // 3 hours later - used for query period only
	results, err := store.CalculateResourceUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed")

	// Capture time after the call to get bounds for time.Now() used inside
	afterCallTime := time.Now()

	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 1, "expected 1 allocation")

	alloc := result.Allocations[0]
	assert.Equal(t, float32(1), alloc.Resources.CPU.Cores, "CPU cores should match")
	assert.Equal(t, uint64(2*1e9), alloc.Resources.RAM.Size, "RAM size should match")
	assert.Equal(t, uint64(5*1e9), alloc.Resources.Disk.Size, "Disk size should match")

	// Since the function now uses time.Now() instead of end, the duration will be
	// from startTime to whenever time.Now() was called inside the function
	// We verify it's within reasonable bounds (between beforeCallTime and afterCallTime)
	minExpectedDuration := beforeCallTime.Sub(startTime)
	maxExpectedDuration := afterCallTime.Sub(startTime)

	assert.True(t, alloc.Duration >= minExpectedDuration,
		"duration should be at least from startTime to beforeCallTime (got %v, expected at least %v)",
		alloc.Duration, minExpectedDuration)
	assert.True(t, alloc.Duration <= maxExpectedDuration,
		"duration should be at most from startTime to afterCallTime (got %v, expected at most %v)",
		alloc.Duration, maxExpectedDuration)
	assert.True(t, alloc.EndTime.IsZero(), "EndTime should be zero for running allocation")
	assert.Equal(t, startTime, alloc.StartTime, "StartTime should match the event timestamp")
}

func TestCalculateResourceUtilizationByContract_MultipleDeployments(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deployment1ID := "deployment-1"
	deployment2ID := "deployment-2"
	allocation1ID := "allocation-1"
	allocation2ID := "allocation-2"

	baseTime := time.Now().Truncate(time.Second)

	// Deployment 1: task allocation (60 seconds)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocation1ID, deployment1ID, 2, 4, 10, 0), baseTime.Add(10*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocation1ID, deployment1ID), baseTime.Add(70*time.Second))

	// Deployment 2: service allocation (120 seconds)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocation2ID, deployment2ID, 4, 8, 20, 0), baseTime.Add(20*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.StopAllocationEvent,
		createStopAllocationEvent(allocation2ID, deployment2ID), baseTime.Add(140*time.Second))

	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateResourceUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed")
	require.Len(t, results, 2, "expected 2 deployments")

	// Find deployments
	var dep1, dep2 contracts.DeploymentResourceUtilization
	for _, r := range results {
		if r.DeploymentID == deployment1ID { //nolint:staticcheck
			dep1 = r
		} else if r.DeploymentID == deployment2ID {
			dep2 = r
		}
	}

	require.NotEmpty(t, dep1.DeploymentID, "should find deployment 1")
	require.NotEmpty(t, dep2.DeploymentID, "should find deployment 2")

	require.Len(t, dep1.Allocations, 1, "deployment 1 should have 1 allocation")
	require.Len(t, dep2.Allocations, 1, "deployment 2 should have 1 allocation")

	assert.Equal(t, 60.0, dep1.TotalUtilizationSec, "deployment 1 should have 60 seconds")
	assert.Equal(t, 120.0, dep2.TotalUtilizationSec, "deployment 2 should have 120 seconds")
}

func TestCalculateResourceUtilizationByContract_MultipleAllocationsPerDeployment(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocation1ID := "allocation-1"
	allocation2ID := "allocation-2"

	baseTime := time.Now().Truncate(time.Second)

	// First allocation: task (60 seconds)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocation1ID, deploymentID, 2, 4, 10, 0), baseTime.Add(10*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocation1ID, deploymentID), baseTime.Add(70*time.Second))

	// Second allocation: service (120 seconds)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocation2ID, deploymentID, 4, 8, 20, 0), baseTime.Add(20*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.StopAllocationEvent,
		createStopAllocationEvent(allocation2ID, deploymentID), baseTime.Add(140*time.Second))

	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateResourceUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 2, "expected 2 allocations")

	expectedTotal := 60.0 + 120.0
	assert.InDelta(t, expectedTotal, result.TotalUtilizationSec, 1.0, "total utilization should be 180.0 seconds")
}

func TestCalculateResourceUtilizationByContract_EmptyResult(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"

	baseTime := time.Now().Truncate(time.Second)
	start := baseTime
	end := baseTime.Add(2 * time.Hour)

	results, err := store.CalculateResourceUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed")
	assert.Len(t, results, 0, "expected empty result")
}

func TestCalculateResourceUtilizationByContract_TimeRangeFiltering(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocation1ID := "allocation-1" // Within range
	allocation2ID := "allocation-2" // Before range
	allocation3ID := "allocation-3" // After range

	baseTime := time.Now().Truncate(time.Second)

	// Allocation 1: starts and ends within query range
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocation1ID, deploymentID, 2, 4, 10, 0), baseTime.Add(10*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocation1ID, deploymentID), baseTime.Add(70*time.Second))

	// Allocation 2: ends before query range (should be excluded)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocation2ID, deploymentID, 1, 2, 5, 0), baseTime.Add(-10*time.Minute))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocation2ID, deploymentID), baseTime.Add(-5*time.Minute))

	// Allocation 3: starts after query range (should be excluded)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocation3ID, deploymentID, 4, 8, 20, 0), baseTime.Add(3*time.Hour))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocation3ID, deploymentID), baseTime.Add(3*time.Hour+60*time.Second))

	queryStart := baseTime
	queryEnd := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateResourceUtilizationByContract(contractDID, queryStart, queryEnd)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 1, "expected 1 allocation (only allocation-1 should be included)")
	assert.Equal(t, allocation1ID, result.Allocations[0].AllocationID, "should only include allocation-1")
}

func TestCalculateResourceUtilizationByContract_ContractIsolation(t *testing.T) {
	store := setupTestDB(t)
	contract1DID := "contract-1"
	contract2DID := "contract-2"
	deployment1ID := "deployment-1"
	deployment2ID := "deployment-2"
	allocation1ID := "allocation-1"
	allocation2ID := "allocation-2"

	baseTime := time.Now().Truncate(time.Second)

	// Contract 1 allocation
	addEventWithTimestamp(t, store, contract1DID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocation1ID, deployment1ID, 2, 4, 10, 0), baseTime.Add(10*time.Second))
	addEventWithTimestamp(t, store, contract1DID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocation1ID, deployment1ID), baseTime.Add(70*time.Second))

	// Contract 2 allocation
	addEventWithTimestamp(t, store, contract2DID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocation2ID, deployment2ID, 4, 8, 20, 0), baseTime.Add(10*time.Second))
	addEventWithTimestamp(t, store, contract2DID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocation2ID, deployment2ID), baseTime.Add(70*time.Second))

	start := baseTime
	end := baseTime.Add(2 * time.Hour)

	// Query contract 1
	results1, err := store.CalculateResourceUtilizationByContract(contract1DID, start, end)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed for contract 1")
	require.Len(t, results1, 1, "contract 1: expected 1 deployment")
	require.Len(t, results1[0].Allocations, 1, "contract 1: expected 1 allocation")
	assert.Equal(t, allocation1ID, results1[0].Allocations[0].AllocationID, "contract 1 results contain contract 2's allocation - contracts are not isolated")

	// Query contract 2
	results2, err := store.CalculateResourceUtilizationByContract(contract2DID, start, end)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed for contract 2")
	require.Len(t, results2, 1, "contract 2: expected 1 deployment")
	require.Len(t, results2[0].Allocations, 1, "contract 2: expected 1 allocation")
	assert.Equal(t, allocation2ID, results2[0].Allocations[0].AllocationID, "contract 2 results contain contract 1's allocation - contracts are not isolated")
}

func TestCalculateResourceUtilizationByContract_MixedTaskAndService(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	taskAllocationID := "allocation-task"
	serviceAllocationID := "allocation-service"

	baseTime := time.Now().Truncate(time.Second)

	// Task allocation (complete)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(taskAllocationID, deploymentID, 2, 4, 10, 0), baseTime.Add(10*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(taskAllocationID, deploymentID), baseTime.Add(70*time.Second))

	// Service allocation (stop)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(serviceAllocationID, deploymentID, 4, 8, 20, 0), baseTime.Add(20*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.StopAllocationEvent,
		createStopAllocationEvent(serviceAllocationID, deploymentID), baseTime.Add(140*time.Second))

	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateResourceUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 2, "expected 2 allocations (task and service)")

	// Verify both allocations have resources
	for _, alloc := range result.Allocations {
		assert.Greater(t, alloc.Resources.CPU.Cores, float32(0), "allocation should have CPU cores")
		assert.Greater(t, alloc.Resources.RAM.Size, uint64(0), "allocation should have RAM")
		assert.Greater(t, alloc.Resources.Disk.Size, uint64(0), "allocation should have Disk")
	}
}

func TestCalculateResourceUtilizationByContract_ContinuingFromPreviousPeriod(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocationID := "allocation-running"

	baseTime := time.Now().Truncate(time.Second)

	// Allocation started before query period but is still running
	startTime := baseTime.Add(-10 * time.Minute)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocationID, deploymentID, 2, 4, 10, 0), startTime)

	// Query for period starting now
	queryStart := baseTime
	queryEnd := baseTime.Add(2 * time.Minute)

	beforeCallTime := time.Now()
	results, err := store.CalculateResourceUtilizationByContract(contractDID, queryStart, queryEnd)
	afterCallTime := time.Now()
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 1, "expected 1 allocation")

	alloc := result.Allocations[0]
	assert.Equal(t, float32(2), alloc.Resources.CPU.Cores, "CPU cores should match")
	assert.Equal(t, uint64(4*1e9), alloc.Resources.RAM.Size, "RAM size should match")

	// Duration should only count from queryStart onwards
	minExpectedDuration := beforeCallTime.Sub(queryStart)
	maxExpectedDuration := afterCallTime.Sub(queryStart)

	assert.True(t, alloc.Duration >= minExpectedDuration,
		"duration should be at least from queryStart to beforeCallTime")
	assert.True(t, alloc.Duration <= maxExpectedDuration,
		"duration should be at most from queryStart to afterCallTime")
	assert.True(t, alloc.EndTime.IsZero(), "EndTime should be zero for running allocation")
}

func TestCalculateResourceUtilizationByContract_StoppedInQueryPeriod(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocationID := "allocation-service"

	baseTime := time.Now().Truncate(time.Second)

	// Add StartAllocationEvent BEFORE the query period
	startTime := baseTime.Add(-10 * time.Minute)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocationID, deploymentID, 2, 4, 10, 0), startTime)

	// Add StopAllocationEvent WITHIN the query period
	stopTime := baseTime.Add(1 * time.Minute)
	addEventWithTimestamp(t, store, contractDID, events.StopAllocationEvent,
		createStopAllocationEvent(allocationID, deploymentID), stopTime)

	// Query for time period
	queryStart := baseTime
	queryEnd := baseTime.Add(2 * time.Minute)

	results, err := store.CalculateResourceUtilizationByContract(contractDID, queryStart, queryEnd)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 1, "expected 1 allocation")

	alloc := result.Allocations[0]
	assert.Equal(t, float32(2), alloc.Resources.CPU.Cores, "CPU cores should match")

	// Duration should be from queryStart to stopTime (1 minute), not from startTime
	expectedDuration := stopTime.Sub(queryStart)
	assert.Equal(t, expectedDuration, alloc.Duration, "duration should be from queryStart to stopTime")
	assert.False(t, alloc.EndTime.IsZero(), "EndTime should be set")
	assert.Equal(t, stopTime.Unix(), alloc.EndTime.Unix(), "end time should match")

	// Total utilization should be 1 minute (60 seconds)
	assert.Equal(t, 60.0, result.TotalUtilizationSec, "total utilization should be 60.0 seconds")
}

func TestCalculateResourceUtilizationByContract_ExcludedIfEndedBeforeQueryPeriod(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocationID := "allocation-service"

	baseTime := time.Now().Truncate(time.Second)

	// Add StartAllocationEvent BEFORE the query period
	startTime := baseTime.Add(-10 * time.Minute)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocationID, deploymentID, 2, 4, 10, 0), startTime)

	// Add StopAllocationEvent BEFORE the query period (allocation ended before query period)
	stopTime := baseTime.Add(-5 * time.Minute)
	addEventWithTimestamp(t, store, contractDID, events.StopAllocationEvent,
		createStopAllocationEvent(allocationID, deploymentID), stopTime)

	// Query for time period that starts AFTER the allocation ended
	queryStart := baseTime
	queryEnd := baseTime.Add(2 * time.Minute)

	results, err := store.CalculateResourceUtilizationByContract(contractDID, queryStart, queryEnd)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed")

	// Should have no deployments because the allocation ended before the query period
	assert.Len(t, results, 0, "expected 0 deployments (allocation ended before query period)")
}

func TestCalculateResourceUtilizationByContract_ResourceFallback(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocationID := "allocation-1"

	baseTime := time.Now().Truncate(time.Second)
	startTime := baseTime.Add(10 * time.Second)
	completeTime := baseTime.Add(70 * time.Second)

	// Add CreateAllocationEvent with resources (should be used as fallback)
	addEventWithTimestamp(t, store, contractDID, events.CreateAllocationEvent,
		createCreateAllocationEventWithResources(allocationID, deploymentID, 2, 4, 10, 0), baseTime)

	// Add StartAllocationEvent WITHOUT resources (should fallback to CreateAllocationEvent)
	startEventWithoutResources := map[string]interface{}{
		"type":                 string(events.StartAllocationEvent),
		"allocation_id":        allocationID,
		"deployment_id":        deploymentID,
		"compute_provider_did": "provider-did",
		// No resources field
	}
	startEventData, _ := json.Marshal(startEventWithoutResources)
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent, startEventData, startTime)

	// Add Complete event
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocationID, deploymentID), completeTime)

	// Calculate utilization
	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateResourceUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 1, "expected 1 allocation")

	alloc := result.Allocations[0]
	// Should have resources from CreateAllocationEvent (fallback)
	assert.Equal(t, float32(2), alloc.Resources.CPU.Cores, "CPU cores should come from CreateAllocationEvent")
	assert.Equal(t, uint64(4*1e9), alloc.Resources.RAM.Size, "RAM size should come from CreateAllocationEvent")
	assert.Equal(t, uint64(10*1e9), alloc.Resources.Disk.Size, "Disk size should come from CreateAllocationEvent")
}

func TestCalculateResourceUtilizationByContract_DifferentResourceConfigurations(t *testing.T) {
	store := setupTestDB(t)
	contractDID := "contract-123"
	deploymentID := "deployment-1"
	allocation1ID := "allocation-1" // 1 core, 2GB RAM, 5GB Disk, 0 GPU
	allocation2ID := "allocation-2" // 4 cores, 16GB RAM, 50GB Disk, 1 GPU
	allocation3ID := "allocation-3" // 8 cores, 32GB RAM, 100GB Disk, 2 GPU

	baseTime := time.Now().Truncate(time.Second)

	// Allocation 1: minimal resources
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocation1ID, deploymentID, 1, 2, 5, 0), baseTime.Add(10*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocation1ID, deploymentID), baseTime.Add(70*time.Second))

	// Allocation 2: medium resources with 1 GPU
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocation2ID, deploymentID, 4, 16, 50, 1), baseTime.Add(20*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocation2ID, deploymentID), baseTime.Add(80*time.Second))

	// Allocation 3: large resources with 2 GPU
	addEventWithTimestamp(t, store, contractDID, events.StartAllocationEvent,
		createStartAllocationEventWithResources(allocation3ID, deploymentID, 8, 32, 100, 2), baseTime.Add(30*time.Second))
	addEventWithTimestamp(t, store, contractDID, events.CompleteAllocationEvent,
		createCompleteAllocationEvent(allocation3ID, deploymentID), baseTime.Add(90*time.Second))

	start := baseTime
	end := baseTime.Add(2 * time.Hour)
	results, err := store.CalculateResourceUtilizationByContract(contractDID, start, end)
	require.NoError(t, err, "CalculateResourceUtilizationByContract failed")
	require.Len(t, results, 1, "expected 1 deployment")

	result := results[0]
	require.Len(t, result.Allocations, 3, "expected 3 allocations")

	// Verify each allocation has correct resources
	for _, alloc := range result.Allocations {
		switch alloc.AllocationID {
		case allocation1ID:
			assert.Equal(t, float32(1), alloc.Resources.CPU.Cores)
			assert.Equal(t, uint64(2*1e9), alloc.Resources.RAM.Size)
			assert.Equal(t, uint64(5*1e9), alloc.Resources.Disk.Size)
			assert.Len(t, alloc.Resources.GPUs, 0, "allocation 1 should have no GPUs")
		case allocation2ID:
			assert.Equal(t, float32(4), alloc.Resources.CPU.Cores)
			assert.Equal(t, uint64(16*1e9), alloc.Resources.RAM.Size)
			assert.Equal(t, uint64(50*1e9), alloc.Resources.Disk.Size)
			assert.Len(t, alloc.Resources.GPUs, 1, "allocation 2 should have 1 GPU")
		case allocation3ID:
			assert.Equal(t, float32(8), alloc.Resources.CPU.Cores)
			assert.Equal(t, uint64(32*1e9), alloc.Resources.RAM.Size)
			assert.Equal(t, uint64(100*1e9), alloc.Resources.Disk.Size)
			assert.Len(t, alloc.Resources.GPUs, 2, "allocation 3 should have 2 GPUs")
		}
	}
}

// Helper to create event JSON data
func createStartAllocationEvent(allocationID, deploymentID string) []byte {
	event := map[string]interface{}{
		"type":                 string(events.StartAllocationEvent),
		"allocation_id":        allocationID,
		"deployment_id":        deploymentID,
		"compute_provider_did": "provider-did",
	}
	data, _ := json.Marshal(event)
	return data
}

func createStartAllocationEventWithResources(allocationID, deploymentID string, cpuCores float32, ramGB uint64, diskGB uint64, gpuCount int) []byte {
	// Convert GB to bytes (1 GB = 1e9 bytes)
	ramBytes := ramGB * 1e9
	diskBytes := diskGB * 1e9

	resources := map[string]interface{}{
		"cpu": map[string]interface{}{
			"cores":       cpuCores,
			"clock_speed": 0,
		},
		"ram": map[string]interface{}{
			"size": ramBytes,
		},
		"disk": map[string]interface{}{
			"size": diskBytes,
		},
	}

	if gpuCount > 0 {
		gpus := make([]map[string]interface{}, gpuCount)
		for i := 0; i < gpuCount; i++ {
			gpus[i] = map[string]interface{}{
				"index": i,
				"vram":  8192,
				"model": "test-gpu",
			}
		}
		resources["gpus"] = gpus
	}

	event := map[string]interface{}{
		"type":                 string(events.StartAllocationEvent),
		"allocation_id":        allocationID,
		"deployment_id":        deploymentID,
		"compute_provider_did": "provider-did",
		"resources":            resources,
	}
	data, _ := json.Marshal(event)
	return data
}

func createCreateAllocationEventWithResources(allocationID, deploymentID string, cpuCores float32, ramGB uint64, diskGB uint64, gpuCount int) []byte {
	// Convert GB to bytes (1 GB = 1e9 bytes)
	ramBytes := ramGB * 1e9
	diskBytes := diskGB * 1e9

	resources := map[string]interface{}{
		"cpu": map[string]interface{}{
			"cores":       cpuCores,
			"clock_speed": 0,
		},
		"ram": map[string]interface{}{
			"size": ramBytes,
		},
		"disk": map[string]interface{}{
			"size": diskBytes,
		},
	}

	if gpuCount > 0 {
		gpus := make([]map[string]interface{}, gpuCount)
		for i := 0; i < gpuCount; i++ {
			gpus[i] = map[string]interface{}{
				"index": i,
				"vram":  8192,
				"model": "test-gpu",
			}
		}
		resources["gpus"] = gpus
	}

	event := map[string]interface{}{
		"type":                 string(events.CreateAllocationEvent),
		"allocation_id":        allocationID,
		"deployment_id":        deploymentID,
		"compute_provider_did": "provider-did",
		"resources":            resources,
	}
	data, _ := json.Marshal(event)
	return data
}

func createCompleteAllocationEvent(allocationID, deploymentID string) []byte {
	event := map[string]interface{}{
		"type":                 string(events.CompleteAllocationEvent),
		"allocation_id":        allocationID,
		"deployment_id":        deploymentID,
		"compute_provider_did": "provider-did",
	}
	data, _ := json.Marshal(event)
	return data
}

func createStopAllocationEvent(allocationID, deploymentID string) []byte {
	event := map[string]interface{}{
		"type":                 string(events.StopAllocationEvent),
		"allocation_id":        allocationID,
		"deployment_id":        deploymentID,
		"compute_provider_did": "provider-did",
	}
	data, _ := json.Marshal(event)
	return data
}

func setupTestDB(t *testing.T) *Store {
	t.Helper()

	tempDir := t.TempDir()

	db, err := clover.Open(tempDir)
	require.NoError(t, err, "failed to open CloverDB")

	err = db.CreateCollection(contractsUsageCollection)
	require.NoError(t, err, "failed to create collection %s", contractsUsageCollection)

	store, err := New(db)
	require.NoError(t, err, "failed to create store")

	return store
}

// Helper function to add an event with a specific timestamp
func addEventWithTimestamp(t *testing.T, store *Store, contractDID string, eventType events.EventType, data []byte, timestamp time.Time) {
	t.Helper()
	doc := document.NewDocument()
	doc.Set("contract_did", contractDID)
	doc.Set("created_at", timestamp.UnixNano())
	doc.Set("usage_data", data)
	if eventType != "" {
		doc.Set("event_type", string(eventType))
	}

	_, err := store.db.InsertOne(contractsUsageCollection, doc)
	require.NoError(t, err, "failed to insert event with timestamp")
}
