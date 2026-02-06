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
	"gitlab.com/nunet/device-management-service/tokenomics/events"
)

const (
	contractsUsageCollection  = "contracts_usage"
	lastProcessedAtCollection = "usage_metadata"
	lastProcessedAtKeyPrefix  = "last_processed_at"
)

type Usage struct {
	ContractDID     string           `json:"contract_did"`                // Tail Contract DID (existing)
	HeadContractDID string           `json:"head_contract_did,omitempty"` // Head Contract DID (new)
	ProviderDID     string           `json:"provider_did,omitempty"`      // Provider DID for per-node billing
	EventType       events.EventType `json:"event_type,omitempty"`        // For indexing - extracted from JSON if not provided
	Data            []byte           `json:"data"`                        // Raw JSON bytes
	Timestamp       time.Time        `json:"timestamp,omitempty"`         // Event timestamp
}

type Store struct {
	db *clover.DB
}

// EventFilters defines filters for querying events
type EventFilters struct {
	ContractDID     string // Tail Contract DID
	HeadContractDID string // Head Contract DID (new)
	EventTypes      []events.EventType
	StartTime       time.Time
	EndTime         time.Time
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
	doc.Set("head_contract_did", u.HeadContractDID) // Store Head Contract DID (empty for non-chain contracts)
	doc.Set("provider_did", u.ProviderDID)          // Store provider DID for filtering
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
		if hcDid, ok := doc.Get("head_contract_did").(string); ok {
			u.HeadContractDID = hcDid
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
		if hcDid, ok := doc.Get("head_contract_did").(string); ok {
			currentUsage.HeadContractDID = hcDid
		}
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
		if hcDid, ok := doc.Get("head_contract_did").(string); ok {
			u.HeadContractDID = hcDid
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

	if filters.HeadContractDID != "" {
		headContractCondition := query.Field("head_contract_did").Eq(filters.HeadContractDID)
		if !hasCondition {
			combinedCondition = headContractCondition
		} else {
			combinedCondition = combinedCondition.And(headContractCondition)
		}
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
		if hcDid, ok := doc.Get("head_contract_did").(string); ok {
			u.HeadContractDID = hcDid
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

// QueryEventsByProvider queries events filtered by contract and provider
func (s *Store) QueryEventsByProvider(contractDID, providerDID string, filters EventFilters) ([]*Usage, error) {
	filters.ContractDID = contractDID

	q := query.NewQuery(contractsUsageCollection)

	// Build conditions
	var conditions []query.Criteria
	conditions = append(conditions, query.Field("contract_did").Eq(contractDID))
	if providerDID != "" {
		conditions = append(conditions, query.Field("provider_did").Eq(providerDID))
	}

	// Apply other filters
	if len(filters.EventTypes) > 0 {
		typeStrs := make([]interface{}, len(filters.EventTypes))
		for i, et := range filters.EventTypes {
			typeStrs[i] = string(et)
		}
		conditions = append(conditions, query.Field("event_type").In(typeStrs...))
	}

	if !filters.StartTime.IsZero() {
		conditions = append(conditions, query.Field("created_at").GtEq(filters.StartTime.UnixNano()))
	}

	if !filters.EndTime.IsZero() {
		conditions = append(conditions, query.Field("created_at").LtEq(filters.EndTime.UnixNano()))
	}

	// Combine all conditions
	var combinedCondition query.Criteria
	for i, cond := range conditions {
		if i == 0 {
			combinedCondition = cond
		} else {
			combinedCondition = combinedCondition.And(cond)
		}
	}

	q = q.Where(combinedCondition)

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
		if hcDid, ok := doc.Get("head_contract_did").(string); ok {
			u.HeadContractDID = hcDid
		}
		if pdid, ok := doc.Get("provider_did").(string); ok {
			u.ProviderDID = pdid
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

// GetEventsByHeadContract retrieves events for Head Contract (head contract in chain)
func (s *Store) GetEventsByHeadContract(headContractDID string) ([]*Usage, error) {
	q := query.NewQuery(contractsUsageCollection).
		Where(query.Field("head_contract_did").Eq(headContractDID))

	docs, err := s.db.FindAll(q)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve usages for head contract %s: %w", headContractDID, err)
	}

	usages := make([]*Usage, 0, len(docs))
	for _, doc := range docs {
		var u Usage
		if cdid, ok := doc.Get("contract_did").(string); ok {
			u.ContractDID = cdid
		}
		if hcDid, ok := doc.Get("head_contract_did").(string); ok {
			u.HeadContractDID = hcDid
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

// QueryAllocationEvents queries allocation start/end events with smart time bounds.
// This helper eliminates duplication across calculation methods.
// It looks back 1 year to catch allocations that started before query period but are still running.
// If headContractDID is provided, queries by Head Contract DID; otherwise queries by contractDID.
func (s *Store) QueryAllocationEvents(
	contractDID string,
	queryStart, queryEnd time.Time,
	headContractDID string, // Optional: if provided, queries by Head Contract DID
) ([]*Usage, []*Usage, error) {
	// Look back 1 year to catch allocations that started before query period
	queryStartBound := queryStart.AddDate(-1, 0, 0)

	filters := EventFilters{
		EventTypes: []events.EventType{events.StartAllocationEvent},
		StartTime:  queryStartBound,
		EndTime:    queryEnd,
	}
	if headContractDID != "" {
		filters.HeadContractDID = headContractDID
	} else {
		filters.ContractDID = contractDID
	}

	startEvents, err := s.QueryEvents(filters)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query start events: %w", err)
	}

	endFilters := EventFilters{
		EventTypes: []events.EventType{events.CompleteAllocationEvent, events.StopAllocationEvent},
		StartTime:  queryStartBound,
		EndTime:    queryEnd,
	}
	if headContractDID != "" {
		endFilters.HeadContractDID = headContractDID
	} else {
		endFilters.ContractDID = contractDID
	}

	endEvents, err := s.QueryEvents(endFilters)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query end events: %w", err)
	}

	return startEvents, endEvents, nil
}

// QueryDeploymentEvents queries deployment start/stop events with smart time bounds.
// This helper eliminates duplication for deployment-based calculations.
// It looks back 1 year to catch deployments that started before query period but are still running.
// If headContractDID is provided, queries by Head Contract DID; otherwise queries by contractDID.
func (s *Store) QueryDeploymentEvents(
	contractDID string,
	queryStart, queryEnd time.Time,
	headContractDID string, // Optional: if provided, queries by Head Contract DID
) ([]*Usage, []*Usage, error) {
	queryStartBound := queryStart.AddDate(-1, 0, 0)

	filters := EventFilters{
		EventTypes: []events.EventType{events.DeploymentStartEvent},
		StartTime:  queryStartBound,
		EndTime:    queryEnd,
	}
	if headContractDID != "" {
		filters.HeadContractDID = headContractDID
	} else {
		filters.ContractDID = contractDID
	}

	startEvents, err := s.QueryEvents(filters)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query deployment start events: %w", err)
	}

	stopFilters := EventFilters{
		EventTypes: []events.EventType{events.DeploymentStopEvent},
		StartTime:  queryStartBound,
		EndTime:    queryEnd,
	}
	if headContractDID != "" {
		stopFilters.HeadContractDID = headContractDID
	} else {
		stopFilters.ContractDID = contractDID
	}

	stopEvents, err := s.QueryEvents(stopFilters)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query deployment stop events: %w", err)
	}

	return startEvents, stopEvents, nil
}

// QueryCreateAllocationEvents queries create allocation events (for resource fallback).
// This is used when StartAllocationEvent doesn't contain resources.
// No time restriction - need all for resource fallback lookup.
func (s *Store) QueryCreateAllocationEvents(
	contractDID string,
) ([]*Usage, error) {
	return s.QueryEvents(EventFilters{
		ContractDID: contractDID,
		EventTypes:  []events.EventType{events.CreateAllocationEvent},
		// No time restriction - need all for resource fallback
	})
}

// CalculateEffectiveTime calculates effective start/end time for a window within query period.
// This is a simple utility function, not an abstraction.
// It handles the common pattern of:
// - If window started before query period, use query start as effective start
// - If window ended, use window end time (if after query start)
// - If window still running, use query end as effective end
// Returns effectiveStart, effectiveEnd, and valid flag (false if window should be excluded).
func CalculateEffectiveTime(
	windowStart, windowEnd time.Time,
	isComplete bool,
	queryStart, queryEnd time.Time,
) (effectiveStart, effectiveEnd time.Time, valid bool) {
	// If window started before query period, use query start
	if windowStart.Before(queryStart) {
		effectiveStart = queryStart
	} else {
		effectiveStart = windowStart
	}

	// Determine effective end time
	if isComplete {
		// Window ended - check if it ended after query start
		if windowEnd.After(queryStart) {
			effectiveEnd = windowEnd
		} else {
			// Window ended before query period, exclude it
			return time.Time{}, time.Time{}, false
		}
	} else {
		// Window still running - use query end
		effectiveEnd = queryEnd
	}

	// Validate
	if !effectiveStart.Before(effectiveEnd) {
		return time.Time{}, time.Time{}, false
	}

	return effectiveStart, effectiveEnd, true
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
