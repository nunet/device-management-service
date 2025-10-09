// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package usage

import (
	"testing"
	"time"

	"github.com/ostafen/clover/v2"
)

func setupTestDB(t *testing.T) *Store {
	t.Helper()

	tempDir := t.TempDir()

	db, err := clover.Open(tempDir)
	if err != nil {
		t.Fatalf("failed to open CloverDB: %v", err)
	}

	err = db.CreateCollection(contractsUsageCollection)
	if err != nil {
		t.Fatalf("failed to create collection %s: %v", contractsUsageCollection, err)
	}

	store, err := New(db)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	return store
}

func TestAddUsageEventAndGetAllEvents(t *testing.T) {
	store := setupTestDB(t)

	usage := Usage{
		ContractDID: "contract-123",
		Data:        []byte("sample data"),
	}

	if err := store.AddUsageEvent(usage); err != nil {
		t.Fatalf("failed to add usage: %v", err)
	}

	events, err := store.GetAllEvents()
	if err != nil {
		t.Fatalf("failed to get all events: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].ContractDID != usage.ContractDID {
		t.Errorf("expected ContractDID %s, got %s", usage.ContractDID, events[0].ContractDID)
	}
	if string(events[0].Data) != string(usage.Data) {
		t.Errorf("expected Data %s, got %s", usage.Data, events[0].Data)
	}
}

func TestGetEventsByContract(t *testing.T) {
	store := setupTestDB(t)

	contract1 := "contract-111"
	contract2 := "contract-222"

	_ = store.AddUsageEvent(Usage{ContractDID: contract1, Data: []byte("event-1")})
	_ = store.AddUsageEvent(Usage{ContractDID: contract1, Data: []byte("event-2")})
	_ = store.AddUsageEvent(Usage{ContractDID: contract2, Data: []byte("event-3")})

	events1, err := store.GetEventsByContract(contract1)
	if err != nil {
		t.Fatalf("failed to get events by contract: %v", err)
	}

	if len(events1) != 2 {
		t.Fatalf("expected 2 events for %s, got %d", contract1, len(events1))
	}

	events2, err := store.GetEventsByContract(contract2)
	if err != nil {
		t.Fatalf("failed to get events by contract: %v", err)
	}

	if len(events2) != 1 {
		t.Fatalf("expected 1 event for %s, got %d", contract2, len(events2))
	}
}

func TestGetEventsByDateRange(t *testing.T) {
	store := setupTestDB(t)

	firstUsage := Usage{ContractDID: "contract-1", Data: []byte("data-1")}
	secondUsage := Usage{ContractDID: "contract-2", Data: []byte("data-2")}

	if err := store.AddUsageEvent(firstUsage); err != nil {
		t.Fatalf("failed to insert first usage: %v", err)
	}

	time.Sleep(1 * time.Second)

	if err := store.AddUsageEvent(secondUsage); err != nil {
		t.Fatalf("failed to insert second usage: %v", err)
	}

	start := time.Now().Add(-500 * time.Millisecond)
	end := time.Now().Add(1 * time.Second)

	events, err := store.GetEventsByDateRange(start, end)
	if err != nil {
		t.Fatalf("failed to get events by date range: %v", err)
	}

	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}

	if events[0].ContractDID != secondUsage.ContractDID {
		t.Errorf("expected ContractDID %s, got %s", secondUsage.ContractDID, events[0].ContractDID)
	}
}

func TestCountAllocationsByContract(t *testing.T) {
	store := setupTestDB(t)

	createEvent := []byte(`{"type":"CREATE_ALLOCATION_EVENT","allocation_id":"a1"}`)
	startEvent := []byte(`{"type":"START_ALLOCATION_EVENT","allocation_id":"a1"}`)

	usages := []Usage{
		{ContractDID: "contract-123", Data: createEvent}, // should count
		{ContractDID: "contract-123", Data: startEvent},  // should NOT count
		{ContractDID: "contract-123", Data: createEvent}, // should count
		{ContractDID: "contract-456", Data: createEvent}, // should count
	}

	for _, u := range usages {
		if err := store.AddUsageEvent(u); err != nil {
			t.Fatalf("failed to insert usage: %v", err)
		}
	}

	start := time.Now().Add(-1 * time.Hour)
	end := time.Now().Add(1 * time.Hour)

	counts, err := store.CountAllocationsByContract(start, end)
	if err != nil {
		t.Fatalf("CountAllocationsByContract failed: %v", err)
	}

	if counts["contract-123"] != 2 {
		t.Errorf("expected 2 allocations for contract-123, got %d", counts["contract-123"])
	}
	if counts["contract-456"] != 1 {
		t.Errorf("expected 1 allocation for contract-456, got %d", counts["contract-456"])
	}
	if _, ok := counts["contract-789"]; ok {
		t.Errorf("did not expect any count for contract-789")
	}
}

func TestSaveAndGetLastProcessedAt(t *testing.T) {
	store := setupTestDB(t)

	ts, err := store.GetLastProcessedAt()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ts.Equal(time.Unix(0, 0)) {
		t.Errorf("expected initial timestamp to be Unix(0), got %v", ts)
	}

	now := time.Now().Truncate(time.Second) // truncate for equality
	if err := store.SaveLastProcessedAt(now); err != nil {
		t.Fatalf("failed to save last processed at: %v", err)
	}

	ts, err = store.GetLastProcessedAt()
	if err != nil {
		t.Fatalf("failed to get last processed at: %v", err)
	}
	if !ts.Equal(now) {
		t.Errorf("expected timestamp %v, got %v", now, ts)
	}

	newer := now.Add(15 * time.Minute).Truncate(time.Second)
	if err := store.SaveLastProcessedAt(newer); err != nil {
		t.Fatalf("failed to update last processed at: %v", err)
	}

	ts, err = store.GetLastProcessedAt()
	if err != nil {
		t.Fatalf("failed to get updated last processed at: %v", err)
	}
	if !ts.Equal(newer) {
		t.Errorf("expected updated timestamp %v, got %v", newer, ts)
	}
}
