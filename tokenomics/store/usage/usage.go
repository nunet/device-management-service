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
	contractsUsageCollection = "contracts_usage"
	usageMetadataCollection  = "usage_metadata"
	metadataDocID            = "last_processed_at"
)

type Usage struct {
	ContractDID string           `json:"contract_did"`
	EventType   events.EventType `json:"event_type,omitempty"` // For indexing - extracted from JSON if not provided
	Data        []byte           `json:"data"`                 // Raw JSON bytes
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
		var base struct {
			Type events.EventType `json:"type"`
		}
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

	if filters.ContractDID != "" {
		q = q.Where(query.Field("contract_did").Eq(filters.ContractDID))
	}

	if len(filters.EventTypes) > 0 {
		typeStrs := make([]interface{}, len(filters.EventTypes))
		for i, et := range filters.EventTypes {
			typeStrs[i] = string(et)
		}
		q = q.Where(query.Field("event_type").In(typeStrs...))
	}

	if !filters.StartTime.IsZero() {
		q = q.Where(query.Field("created_at").GtEq(filters.StartTime.UnixNano()))
	}
	if !filters.EndTime.IsZero() {
		q = q.Where(query.Field("created_at").LtEq(filters.EndTime.UnixNano()))
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
		usages = append(usages, &u)
	}

	return usages, nil
}

// CountAllocationsByContract retrieves all events within a given time range
// and returns a map of contractDID -> allocation count (based on START_ALLOCATION_EVENT).
// This is the backward-compatible version that returns counts for all contracts.
func (s *Store) CountAllocationsByContract(start, end time.Time) (map[string]int, error) {
	// First filter by event_type at DB level, then unmarshal to count unique allocation_ids
	events, err := s.QueryEvents(EventFilters{
		EventTypes: []events.EventType{events.StartAllocationEvent},
		StartTime:  start,
		EndTime:    end,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}

	// Group by contract and count unique allocations
	contractAllocations := make(map[string]map[string]bool)
	for _, evt := range events {
		var evtData struct {
			AllocationID string `json:"allocation_id"`
		}
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
	events, err := s.QueryEvents(EventFilters{
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
	for _, evt := range events {
		var evtData struct {
			AllocationID string `json:"allocation_id"`
		}
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
	events, err := s.QueryEvents(EventFilters{
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
	for _, evt := range events {
		var evtData struct {
			DeploymentID string `json:"deployment_id"`
		}
		if err := json.Unmarshal(evt.Data, &evtData); err != nil {
			continue
		}
		if evtData.DeploymentID != "" {
			deploymentSet[evtData.DeploymentID] = true
		}
	}

	return len(deploymentSet), nil
}

// SaveLastProcessedAt stores the last processed timestamp (Unix seconds).
func (s *Store) SaveLastProcessedAt(t time.Time) error {
	ok, err := s.db.HasCollection(usageMetadataCollection)
	if err != nil {
		return fmt.Errorf("failed to check collection: %w", err)
	}
	if !ok {
		if err := s.db.CreateCollection(usageMetadataCollection); err != nil {
			return fmt.Errorf("failed to create metadata collection: %w", err)
		}
	}

	q := query.NewQuery(usageMetadataCollection).Where(query.Field("key").Eq("last_processed_at"))
	docs, err := s.db.FindAll(q)
	if err != nil {
		return fmt.Errorf("failed to query metadata: %w", err)
	}

	if len(docs) > 0 {
		doc := docs[0]
		doc.Set("last_processed_at", t.Unix())
		if err := s.db.ReplaceById(usageMetadataCollection, doc.ObjectId(), doc); err != nil {
			return fmt.Errorf("failed to update last processed at: %w", err)
		}
	} else {
		doc := document.NewDocument()
		doc.Set("key", "last_processed_at")
		doc.Set("last_processed_at", t.Unix())
		if _, err := s.db.InsertOne(usageMetadataCollection, doc); err != nil {
			return fmt.Errorf("failed to insert last processed at: %w", err)
		}
	}

	return nil
}

// GetLastProcessedAt retrieves the last processed timestamp.
// If no record exists, it returns Unix(0).
func (s *Store) GetLastProcessedAt() (time.Time, error) {
	ok, err := s.db.HasCollection(usageMetadataCollection)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to check metadata collection: %w", err)
	}
	if !ok {
		return time.Unix(0, 0), nil
	}

	q := query.NewQuery(usageMetadataCollection).Where(query.Field("key").Eq("last_processed_at"))
	docs, err := s.db.FindAll(q)
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to query metadata: %w", err)
	}

	if len(docs) == 0 {
		return time.Unix(0, 0), nil
	}

	doc := docs[0]
	if ts, ok := doc.Get("last_processed_at").(int64); ok {
		return time.Unix(ts, 0), nil
	}

	return time.Unix(0, 0), nil
}
