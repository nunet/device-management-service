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
	"errors"
	"fmt"
	"time"

	"github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/document"
	"github.com/ostafen/clover/v2/query"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/events"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	contractsUsageCollection  = "contracts_usage"
	lastProcessedAtCollection = "usage_metadata"
	lastProcessedAtKeyPrefix  = "last_processed_at"
)

type Usage struct {
	ContractDID string           `json:"contract_did"`
	EventType   events.EventType `json:"event_type,omitempty"` // For indexing - extracted from JSON if not provided
	Data        []byte           `json:"data"`                 // Raw JSON bytes
	Timestamp   time.Time        `json:"timestamp,omitempty"`  // Event timestamp
}

type Store struct {
	db *clover.DB
}

// EventFilters defines filters for querying events
type EventFilters struct {
	ContractDID string
	EventTypes  []events.EventType
	StartTime   time.Time
	EndTime     time.Time
}

func New(db *clover.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}

	return &Store{
		db: db,
	}, nil
}

func (s *Store) AddUsageEvent(u Usage) error {
	if u.ContractDID == "" {
		return errors.New("contractDID is empty")
	}

	doc := document.NewDocument()
	doc.Set("contract_did", u.ContractDID)
	doc.Set("created_at", time.Now().UnixNano())
	doc.Set("usage_data", u.Data)

	// Extract event_type from JSON for indexing (if not already provided)
	eventType := u.EventType
	if eventType == "" && len(u.Data) > 0 {
		var base events.EventBase
		if err := json.Unmarshal(u.Data, &base); err == nil {
			eventType = base.Type
		}
	}

	// Store event_type as indexed field for efficient querying
	if eventType != "" {
		doc.Set("event_type", string(eventType))
	}

	_, err := s.db.InsertOne(contractsUsageCollection, doc)
	if err != nil {
		return fmt.Errorf("failed to insert usage event: %w", err)
	}

	return nil
}

func (s *Store) GetEventsByContract(contractDID string) ([]*Usage, error) {
	q := query.NewQuery(contractsUsageCollection).Where(query.Field("contract_did").Eq(contractDID))

	docs, err := s.db.FindAll(q)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve usages for contract %s: %w", contractDID, err)
	}

	usages := make([]*Usage, 0, len(docs))
	for _, doc := range docs {
		var u Usage
		if cdid, ok := doc.Get("contract_did").(string); ok {
			u.ContractDID = cdid
		}
		if data, ok := doc.Get("usage_data").([]byte); ok {
			u.Data = data
		}
		if eventTypeStr, ok := doc.Get("event_type").(string); ok {
			u.EventType = events.EventType(eventTypeStr)
		}
		usages = append(usages, &u)
	}

	return usages, nil
}

// GetAllEvents retrieves all events from DB.
func (s *Store) GetAllEvents() ([]*Usage, error) {
	q := query.NewQuery(contractsUsageCollection)

	docs, err := s.db.FindAll(q)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve all usages: %w", err)
	}

	allUsages := make([]*Usage, 0)
	for _, doc := range docs {
		var currentUsage Usage
		data := doc.Get("usage_data")
		currentUsage.Data = data.([]byte)
		currentUsage.ContractDID = doc.Get("contract_did").(string)
		if eventTypeStr, ok := doc.Get("event_type").(string); ok {
			currentUsage.EventType = events.EventType(eventTypeStr)
		}
		allUsages = append(allUsages, &currentUsage)
	}

	return allUsages, nil
}

// GetEventsByDateRange retrieves all events created within the given date range.
func (s *Store) GetEventsByDateRange(start, end time.Time) ([]*Usage, error) {
	q := query.NewQuery(contractsUsageCollection).Where(
		query.Field("created_at").GtEq(start.UnixNano()).And(query.Field("created_at").LtEq(end.UnixNano())),
	)

	docs, err := s.db.FindAll(q)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve usages by date range: %w", err)
	}

	usages := make([]*Usage, 0, len(docs))
	for _, doc := range docs {
		var u Usage
		if contractDID, ok := doc.Get("contract_did").(string); ok {
			u.ContractDID = contractDID
		}
		if data, ok := doc.Get("usage_data").([]byte); ok {
			u.Data = data
		}
		if eventTypeStr, ok := doc.Get("event_type").(string); ok {
			u.EventType = events.EventType(eventTypeStr)
		}
		usages = append(usages, &u)
	}

	return usages, nil
}

// QueryEvents - Generic query with filters (event_type-based filtering)
func (s *Store) QueryEvents(filters EventFilters) ([]*Usage, error) {
	q := query.NewQuery(contractsUsageCollection)

	// Build up conditions incrementally using And() to ensure proper AND logic
	// This approach ensures all conditions are properly combined as AND conditions
	var combinedCondition query.Criteria
	hasCondition := false

	if filters.ContractDID != "" {
		combinedCondition = query.Field("contract_did").Eq(filters.ContractDID)
		hasCondition = true
	}

	if len(filters.EventTypes) > 0 {
		typeStrs := make([]interface{}, len(filters.EventTypes))
		for i, et := range filters.EventTypes {
			typeStrs[i] = string(et)
		}
		eventTypeCondition := query.Field("event_type").In(typeStrs...)
		if !hasCondition {
			combinedCondition = eventTypeCondition
		} else {
			combinedCondition = combinedCondition.And(eventTypeCondition)
		}
		hasCondition = true
	}

	if !filters.StartTime.IsZero() {
		startTimeCondition := query.Field("created_at").GtEq(filters.StartTime.UnixNano())
		if !hasCondition {
			combinedCondition = startTimeCondition
		} else {
			combinedCondition = combinedCondition.And(startTimeCondition)
		}
		hasCondition = true
	}
	if !filters.EndTime.IsZero() {
		endTimeCondition := query.Field("created_at").LtEq(filters.EndTime.UnixNano())
		if !hasCondition {
			combinedCondition = endTimeCondition
		} else {
			combinedCondition = combinedCondition.And(endTimeCondition)
		}
		hasCondition = true
	}

	// Apply combined condition if we have any
	if hasCondition {
		q = q.Where(combinedCondition)
	}

	docs, err := s.db.FindAll(q)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	usages := make([]*Usage, 0, len(docs))
	for _, doc := range docs {
		var u Usage
		if cdid, ok := doc.Get("contract_did").(string); ok {
			u.ContractDID = cdid
		}
		if data, ok := doc.Get("usage_data").([]byte); ok {
			u.Data = data
		}
		if eventTypeStr, ok := doc.Get("event_type").(string); ok {
			u.EventType = events.EventType(eventTypeStr)
		}
		// Extract timestamp from created_at
		if timestampNano, ok := doc.Get("created_at").(int64); ok {
			u.Timestamp = time.Unix(0, timestampNano)
		}
		usages = append(usages, &u)
	}

	return usages, nil
}

// CalculateTimeUtilizationByContract calculates time utilization per deployment for a contract.
//
// Allocation Event Handling Logic:
// - StartAllocationEvent is pushed for BOTH task and service allocations (this is the start event for both)
// - CreateAllocationEvent is also pushed for both types, but we don't use it for time tracking
// - Allocation type is determined by the end event:
//   - CompleteAllocationEvent → task allocation (one-shot task that completes)
//   - StopAllocationEvent → service allocation (long-running service that can be stopped)
//
// Process:
//  1. Query ALL StartAllocationEvents for the contract (no time restriction) to catch
//     allocations that started before 'start' but are still running or ended after 'start'
//  2. Query ALL end events (Complete/Stop) for the contract (no time restriction) to properly
//     match start and end events, even if end happened before 'start'
//  3. First pass: Process StartAllocationEvent to create allocation windows
//  4. Second pass: Process CompleteAllocationEvent/StopAllocationEvent to determine type and set end time
//  5. Filter allocations to only include those that were active during the period:
//     - Allocations that started after 'start' are included (they're within the period)
//     - Allocations that started before 'start' but ended after 'start' (or are still running) are included
//     - Allocations that ended before 'start' are excluded (already counted in previous period)
//  6. For allocations that started before 'start', only count time from 'start' onwards
func (s *Store) CalculateTimeUtilizationByContract(contractDID string, start, _ time.Time) ([]contracts.DeploymentTimeUtilization, error) {
	// Query ALL StartAllocationEvents for this contract (no time restriction)
	// This is necessary to find allocations that started before 'start' but are still running
	startEvents, err := s.QueryEvents(EventFilters{
		ContractDID: contractDID,
		EventTypes:  []events.EventType{events.StartAllocationEvent},
		// No time restriction - we need ALL start events to catch allocations that started before 'start'
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query start events: %w", err)
	}

	// Query ALL end events for this contract (no time restriction)
	// This is necessary to properly match start and end events, even if end happened before 'start'
	endEventTypes := []events.EventType{
		events.CompleteAllocationEvent, // End event for task allocations
		events.StopAllocationEvent,     // End event for service allocations
	}
	endEvents, err := s.QueryEvents(EventFilters{
		ContractDID: contractDID,
		EventTypes:  endEventTypes,
		// No time restriction - we need ALL end events to properly match with start events
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query end events: %w", err)
	}

	// Combine all events for processing
	startEvents = append(startEvents, endEvents...)

	// Track allocation windows by allocation ID
	type allocationWindow struct {
		allocationID   string
		deploymentID   string
		startTime      time.Time
		endTime        time.Time
		isComplete     bool   // true if we have both start and end events
		allocationType string // "task" or "service" - determined by end event
	}

	// Map: allocationID -> window
	allocationWindows := make(map[string]*allocationWindow)

	// Map: deploymentID -> []allocationID
	deploymentAllocations := make(map[string]map[string]bool) // Use map to avoid duplicates

	// First pass: Process StartAllocationEvent to create allocation windows
	// StartAllocationEvent is used for BOTH task and service allocations
	for _, evt := range startEvents {
		if evt.EventType != events.StartAllocationEvent {
			continue
		}

		eventTime := evt.Timestamp
		if eventTime.IsZero() {
			continue // Skip if no timestamp
		}

		var data events.StartAllocation
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			continue
		}

		// Create window for this allocation (type will be determined by end event)
		if allocationWindows[data.AllocationID] == nil {
			allocationWindows[data.AllocationID] = &allocationWindow{
				allocationID:   data.AllocationID,
				deploymentID:   data.DeploymentID,
				startTime:      eventTime,
				allocationType: "", // Will be determined by end event
			}
			if deploymentAllocations[data.DeploymentID] == nil {
				deploymentAllocations[data.DeploymentID] = make(map[string]bool)
			}
			deploymentAllocations[data.DeploymentID][data.AllocationID] = true
		}
	}

	// Second pass: Process end events to determine allocation type and set end time
	for _, evt := range startEvents {
		eventTime := evt.Timestamp
		if eventTime.IsZero() {
			continue // Skip if no timestamp
		}

		// CompleteAllocationEvent indicates a task allocation
		var data events.AllocationBase
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			continue
		}
		window := allocationWindows[data.AllocationID]
		if window == nil {
			continue // No start event for this allocation, skip
		}
		window.allocationType = "service"
		window.endTime = eventTime
		window.isComplete = false

		if evt.EventType == events.CompleteAllocationEvent { //nolint:staticcheck
			window.allocationType = "task"
			window.isComplete = true
		} else if evt.EventType == events.StopAllocationEvent {
			window.isComplete = true
		}
	}

	// Build deployment time utilization structures
	result := make([]contracts.DeploymentTimeUtilization, 0)
	for deploymentID, allocationIDMap := range deploymentAllocations {
		deploymentUtil := contracts.DeploymentTimeUtilization{
			DeploymentID: deploymentID,
			Allocations:  make([]contracts.AllocationTimeUtilization, 0),
		}

		for allocID := range allocationIDMap {
			window := allocationWindows[allocID]
			if window == nil {
				continue
			}

			// Determine if this allocation is relevant for this time period
			// An allocation is relevant if:
			// 1. It started after 'start' (within the period), OR
			// 2. It started before 'start' but ended after 'start' (or is still running)

			// Calculate effective start time: if allocation started before 'start', use 'start' as effective start
			effectiveStartTime := window.startTime
			if window.startTime.Before(start) {
				// Allocation started before the period - only count time from 'start' onwards
				effectiveStartTime = start
			}

			// Determine end time for duration calculation
			var effectiveEndTime time.Time
			if window.isComplete {
				// Allocation ended - check if it ended after 'start'
				if window.endTime.After(start) {
					effectiveEndTime = window.endTime
				} else {
					// Allocation ended before 'start', skip it (already counted in previous period)
					continue
				}
			} else {
				// Allocation is still running - use current time as effective end time
				effectiveEndTime = time.Now()
			}

			// Skip if effective start is after or equal to effective end
			if !effectiveStartTime.Before(effectiveEndTime) {
				continue
			}

			// Calculate duration from effective start to effective end
			duration := effectiveEndTime.Sub(effectiveStartTime)

			allocUtil := contracts.AllocationTimeUtilization{
				AllocationID: window.allocationID,
				Duration:     duration,
				StartTime:    window.startTime, // Always use actual start time for tracking
			}

			if window.isComplete {
				allocUtil.EndTime = window.endTime
			}

			deploymentUtil.Allocations = append(deploymentUtil.Allocations, allocUtil)
			deploymentUtil.TotalUtilizationSec += duration.Seconds()
		}

		if len(deploymentUtil.Allocations) > 0 {
			result = append(result, deploymentUtil)
		}
	}

	return result, nil
}

// CalculateResourceUtilizationByContract calculates resource utilization per deployment for a contract.
// Allocation Event Handling Logic:
// - StartAllocationEvent is pushed for BOTH task and service allocations (this is the start event for both)
// - Resources are embedded in StartAllocationEvent (primary source)
// - CreateAllocationEvent also contains resources (fallback if StartAllocationEvent missing resources)
// - Allocation type is determined by the end event:
//   - CompleteAllocationEvent → task allocation (one-shot task that completes)
//   - StopAllocationEvent → service allocation (long-running service that can be stopped)
//
// Process:
//  1. Query ALL StartAllocationEvents for the contract (no time restriction) to catch
//     allocations that started before 'start' but are still running or ended after 'start'
//  2. Query ALL end events (Complete/Stop) for the contract (no time restriction) to properly
//     match start and end events, even if end happened before 'start'
//  3. Query ALL CreateAllocationEvents for resource fallback lookup
//  4. First pass: Process StartAllocationEvent to create allocation windows (with resources)
//  5. Second pass: Process CompleteAllocationEvent/StopAllocationEvent to determine type and set end time
//  6. Filter allocations to only include those that were active during the period:
//     - Allocations that started after 'start' are included (they're within the period)
//     - Allocations that started before 'start' but ended after 'start' (or are still running) are included
//     - Allocations that ended before 'start' are excluded (already counted in previous period)
//  7. For allocations that started before 'start', only count time from 'start' onwards
func (s *Store) CalculateResourceUtilizationByContract(contractDID string, start, end time.Time) ([]contracts.DeploymentResourceUtilization, error) {
	// Query ALL StartAllocationEvents for this contract (no time restriction)
	// This is necessary to find allocations that started before 'start' but are still running
	startEvents, err := s.QueryEvents(EventFilters{
		ContractDID: contractDID,
		EventTypes:  []events.EventType{events.StartAllocationEvent},
		// No time restriction - we need ALL start events to catch allocations that started before 'start'
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query start events: %w", err)
	}

	// Query ALL end events for this contract (no time restriction)
	// This is necessary to properly match start and end events, even if end happened before 'start'
	endEventTypes := []events.EventType{
		events.CompleteAllocationEvent, // End event for task allocations
		events.StopAllocationEvent,     // End event for service allocations
	}
	endEvents, err := s.QueryEvents(EventFilters{
		ContractDID: contractDID,
		EventTypes:  endEventTypes,
		// No time restriction - we need ALL end events to properly match with start events
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query end events: %w", err)
	}

	// Query ALL CreateAllocationEvents for resource fallback lookup
	createEvents, err := s.QueryEvents(EventFilters{
		ContractDID: contractDID,
		EventTypes:  []events.EventType{events.CreateAllocationEvent},
		// No time restriction - we need ALL create events for resource fallback lookup
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query create events: %w", err)
	}

	// Build resource map from CreateAllocationEvents (fallback if StartAllocationEvent missing resources)
	allocationResources := make(map[string]types.Resources)
	for _, evt := range createEvents {
		var data events.CreateAllocation
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			continue
		}
		allocationResources[data.AllocationID] = data.Resources
	}

	// Combine all events for processing
	startEvents = append(startEvents, endEvents...)

	// Track allocation windows by allocation ID
	type allocationWindow struct {
		allocationID   string
		deploymentID   string
		startTime      time.Time
		endTime        time.Time
		isComplete     bool   // true if we have both start and end events
		allocationType string // "task" or "service" - determined by end event
		resources      types.Resources
	}

	// Map: allocationID -> window
	allocationWindows := make(map[string]*allocationWindow)

	// Map: deploymentID -> []allocationID
	deploymentAllocations := make(map[string]map[string]bool) // Use map to avoid duplicates

	// First pass: Process StartAllocationEvent to create allocation windows
	// StartAllocationEvent is used for BOTH task and service allocations
	for _, evt := range startEvents {
		if evt.EventType != events.StartAllocationEvent {
			continue
		}

		eventTime := evt.Timestamp
		if eventTime.IsZero() {
			continue // Skip if no timestamp
		}

		var data events.StartAllocation
		if err := json.Unmarshal(evt.Data, &data); err != nil {
			continue
		}

		// Get resources from StartAllocationEvent (primary source)
		resources := data.Resources

		// Fallback: If resources not in StartAllocationEvent, use CreateAllocationEvent
		if resources.CPU.Cores == 0 && resources.RAM.Size == 0 {
			if createRes, ok := allocationResources[data.AllocationID]; ok {
				resources = createRes
			}
		}

		// Create window for this allocation (type will be determined by end event)
		if allocationWindows[data.AllocationID] == nil {
			allocationWindows[data.AllocationID] = &allocationWindow{
				allocationID:   data.AllocationID,
				deploymentID:   data.DeploymentID,
				startTime:      eventTime,
				allocationType: "", // Will be determined by end event
				resources:      resources,
			}
			if deploymentAllocations[data.DeploymentID] == nil {
				deploymentAllocations[data.DeploymentID] = make(map[string]bool)
			}
			deploymentAllocations[data.DeploymentID][data.AllocationID] = true
		}
	}

	// Second pass: Process end events to determine allocation type and set end time
	for _, evt := range startEvents {
		eventTime := evt.Timestamp
		if eventTime.IsZero() {
			continue // Skip if no timestamp
		}

		switch evt.EventType {
		case events.CompleteAllocationEvent:
			// CompleteAllocationEvent indicates a task allocation
			var data events.CompleteAllocation
			if err := json.Unmarshal(evt.Data, &data); err != nil {
				continue
			}
			window := allocationWindows[data.AllocationID]
			if window == nil {
				continue // No start event for this allocation, skip
			}
			window.allocationType = "task"
			window.endTime = eventTime
			window.isComplete = true

		case events.StopAllocationEvent:
			// StopAllocationEvent indicates a service allocation
			var data events.StopAllocation
			if err := json.Unmarshal(evt.Data, &data); err != nil {
				continue
			}
			window := allocationWindows[data.AllocationID]
			if window == nil {
				continue // No start event for this allocation, skip
			}
			window.allocationType = "service"
			window.endTime = eventTime
			window.isComplete = true
		}
	}

	// Build deployment resource utilization structures
	result := make([]contracts.DeploymentResourceUtilization, 0)
	for deploymentID, allocationIDMap := range deploymentAllocations {
		deploymentUtil := contracts.DeploymentResourceUtilization{
			DeploymentID: deploymentID,
			Allocations:  make([]contracts.AllocationResourceUtilization, 0),
		}

		for allocID := range allocationIDMap {
			window := allocationWindows[allocID]
			if window == nil {
				continue
			}

			// Determine if this allocation is relevant for this time period
			// An allocation is relevant if:
			// 1. It started after 'start' (within the period), OR
			// 2. It started before 'start' but ended after 'start' (or is still running)

			// Exclude allocations that started after 'end' (query period end)
			if window.startTime.After(end) {
				continue
			}

			// Calculate effective start time: if allocation started before 'start', use 'start' as effective start
			effectiveStartTime := window.startTime
			if window.startTime.Before(start) {
				// Allocation started before the period - only count time from 'start' onwards
				effectiveStartTime = start
			}

			// Determine end time for duration calculation
			var effectiveEndTime time.Time
			if window.isComplete {
				// Allocation ended - check if it ended after 'start'
				if window.endTime.After(start) {
					effectiveEndTime = window.endTime
				} else {
					// Allocation ended before 'start', skip it (already counted in previous period)
					continue
				}
			} else {
				// Allocation is still running - use current time as effective end time
				effectiveEndTime = time.Now()
			}

			// Skip if effective start is after or equal to effective end
			if !effectiveStartTime.Before(effectiveEndTime) {
				continue
			}

			// Calculate duration from effective start to effective end
			duration := effectiveEndTime.Sub(effectiveStartTime)

			allocUtil := contracts.AllocationResourceUtilization{
				AllocationID: window.allocationID,
				Resources:    window.resources,
				Duration:     duration,
				StartTime:    window.startTime, // Always use actual start time for tracking
			}

			if window.isComplete {
				allocUtil.EndTime = window.endTime
			}

			deploymentUtil.Allocations = append(deploymentUtil.Allocations, allocUtil)
			deploymentUtil.TotalUtilizationSec += duration.Seconds()
		}

		if len(deploymentUtil.Allocations) > 0 {
			result = append(result, deploymentUtil)
		}
	}

	return result, nil
}

// CountAllocationsByContract retrieves all events within a given time range
// and returns a map of contractDID -> allocation count (based on START_ALLOCATION_EVENT).
// This is the backward-compatible version that returns counts for all contracts.
func (s *Store) CountAllocationsByContract(start, end time.Time) (map[string]int, error) {
	// First filter by event_type at DB level, then unmarshal to count unique allocation_ids
	usageEvents, err := s.QueryEvents(EventFilters{
		EventTypes: []events.EventType{events.StartAllocationEvent},
		StartTime:  start,
		EndTime:    end,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	// Group by contract and count unique allocations
	contractAllocations := make(map[string]map[string]bool)
	for _, evt := range usageEvents {
		var evtData events.StartAllocation
		if err := json.Unmarshal(evt.Data, &evtData); err != nil {
			continue
		}
		if evtData.AllocationID != "" {
			if contractAllocations[evt.ContractDID] == nil {
				contractAllocations[evt.ContractDID] = make(map[string]bool)
			}
			contractAllocations[evt.ContractDID][evtData.AllocationID] = true
		}
	}

	// Convert to map[string]int
	contractCounts := make(map[string]int)
	for contractDID, allocationSet := range contractAllocations {
		contractCounts[contractDID] = len(allocationSet)
	}

	return contractCounts, nil
}

// CountAllocationsByContractDID retrieves events within a given time range
// for a specific contract DID and returns the count of unique allocations based on START_ALLOCATION_EVENT.
func (s *Store) CountAllocationsByContractDID(contractDID string, start, end time.Time) (int, error) {
	// First filter by event_type at DB level, then unmarshal to count unique allocation_ids
	usageEvents, err := s.QueryEvents(EventFilters{
		ContractDID: contractDID,
		EventTypes:  []events.EventType{events.StartAllocationEvent},
		StartTime:   start,
		EndTime:     end,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to query events: %w", err)
	}

	// Unmarshal JSON to get allocation_id for unique counting
	allocationSet := make(map[string]bool)
	for _, evt := range usageEvents {
		var evtData events.StartAllocation
		if err := json.Unmarshal(evt.Data, &evtData); err != nil {
			continue
		}
		if evtData.AllocationID != "" {
			allocationSet[evtData.AllocationID] = true
		}
	}

	return len(allocationSet), nil
}

// CountDeploymentsByContract retrieves events within a given time range
// and returns the count of unique deployments based on DEPLOYMENT_START_EVENT.
func (s *Store) CountDeploymentsByContract(contractDID string, start, end time.Time) (int, error) {
	usageEvents, err := s.QueryEvents(EventFilters{
		ContractDID: contractDID,
		EventTypes:  []events.EventType{events.DeploymentStartEvent},
		StartTime:   start,
		EndTime:     end,
	})
	if err != nil {
		return 0, fmt.Errorf("failed to query events: %w", err)
	}

	// Unmarshal JSON to get deployment_id for unique counting
	deploymentSet := make(map[string]bool)
	for _, evt := range usageEvents {
		var evtData events.DeploymentStart
		if err := json.Unmarshal(evt.Data, &evtData); err != nil {
			continue
		}
		if evtData.DeploymentID != "" {
			deploymentSet[evtData.DeploymentID] = true
		}
	}

	return len(deploymentSet), nil
}

// SaveLastProcessedAt stores the last processed timestamp (Unix seconds) for a specific contract.
// If contractDID is empty, it stores a global timestamp.
func (s *Store) SaveLastProcessedAt(contractDID string, t time.Time) error {
	ok, err := s.db.HasCollection(lastProcessedAtCollection)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}
	if !ok {
		if err := s.db.CreateCollection(lastProcessedAtCollection); err != nil {
			return fmt.Errorf("failed to create metadata collection: %w", err)
		}
	}

	// Create a unique key for each contract
	key := lastProcessedAtKeyPrefix
	if contractDID != "" {
		key = fmt.Sprintf("%s:%s", lastProcessedAtKeyPrefix, contractDID)
	}

	q := query.NewQuery(lastProcessedAtCollection).Where(query.Field("key").Eq(key))
	docs, err := s.db.FindAll(q)
	if err != nil {
		return fmt.Errorf("failed to query metadata: %w", err)
	}

	if len(docs) > 0 {
		doc := docs[0]
		doc.Set(lastProcessedAtKeyPrefix, t.Unix())
		if err := s.db.ReplaceById(lastProcessedAtCollection, doc.ObjectId(), doc); err != nil {
			return fmt.Errorf("failed to update last processed at: %w", err)
		}
	} else {
		doc := document.NewDocument()
		doc.Set("key", key)
		if contractDID != "" {
			doc.Set("contract_did", contractDID)
		}
		doc.Set(lastProcessedAtKeyPrefix, t.Unix())
		if _, err := s.db.InsertOne(lastProcessedAtCollection, doc); err != nil {
			return fmt.Errorf("failed to insert last processed at: %w", err)
		}
	}

	return nil
}

// GetLastProcessedAt retrieves the last processed timestamp for a specific contract.
// If contractDID is empty, it retrieves the global timestamp.
// If no record exists, it returns Unix(0).
func (s *Store) GetLastProcessedAt(contractDID string) (time.Time, error) {
	ok, err := s.db.HasCollection(lastProcessedAtCollection)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to check metadata collection: %w", err)
	}
	if !ok {
		return time.Unix(0, 0), nil
	}

	// Create a unique key for each contract
	key := lastProcessedAtKeyPrefix
	if contractDID != "" {
		key = fmt.Sprintf("%s:%s", lastProcessedAtKeyPrefix, contractDID)
	}

	q := query.NewQuery(lastProcessedAtCollection).Where(query.Field("key").Eq(key))
	docs, err := s.db.FindAll(q)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to query metadata: %w", err)
	}

	if len(docs) == 0 {
		return time.Unix(0, 0), nil
	}

	doc := docs[0]
	if ts, ok := doc.Get(lastProcessedAtKeyPrefix).(int64); ok {
		return time.Unix(ts, 0), nil
	}

	return time.Unix(0, 0), nil
}

// InitializeContractMetadata initializes the usage metadata for a new contract.
// This creates a metadata entry with the contract-specific key and sets the initial
// last processed timestamp to Unix(0).
func (s *Store) InitializeContractMetadata(contractDID string) error {
	if contractDID == "" {
		return errors.New("contractDID cannot be empty")
	}

	ok, err := s.db.HasCollection(lastProcessedAtCollection)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}
	if !ok {
		if err := s.db.CreateCollection(lastProcessedAtCollection); err != nil {
			return fmt.Errorf("failed to create metadata collection: %w", err)
		}
	}

	// Create contract-specific key
	key := fmt.Sprintf("%s:%s", lastProcessedAtKeyPrefix, contractDID)

	// Check if metadata already exists for this contract
	q := query.NewQuery(lastProcessedAtCollection).Where(query.Field("key").Eq(key))
	docs, err := s.db.FindAll(q)
	if err != nil {
		return fmt.Errorf("failed to query metadata: %w", err)
	}

	// If metadata already exists, don't overwrite it
	if len(docs) > 0 {
		return nil
	}

	// Create new metadata entry for this contract
	doc := document.NewDocument()
	doc.Set("key", key)
	doc.Set("contract_did", contractDID)
	doc.Set(lastProcessedAtKeyPrefix, time.Unix(0, 0).Unix())

	if _, err := s.db.InsertOne(lastProcessedAtCollection, doc); err != nil {
		return fmt.Errorf("failed to initialize contract metadata: %w", err)
	}

	return nil
}
