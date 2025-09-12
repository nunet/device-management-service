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
	ContractDID string `json:"contract_did"`
	Data        []byte `json:"data"`
}

type Store struct {
	db *clover.DB
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
	doc.Set("created_at", time.Now().Unix())
	doc.Set("usage_data", u.Data)

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
		allUsages = append(allUsages, &currentUsage)
	}

	return allUsages, nil
}

// GetEventsByDateRange retrieves all events created within the given date range.
func (s *Store) GetEventsByDateRange(start, end time.Time) ([]*Usage, error) {
	q := query.NewQuery(contractsUsageCollection).Where(
		query.Field("created_at").GtEq(start.Unix()).And(query.Field("created_at").LtEq(end.Unix())),
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
		usages = append(usages, &u)
	}

	return usages, nil
}

// CountAllocationsByContract retrieves all events and returns a map
// of contractDID -> allocation count (based on CreateAllocationEvent).
// CountAllocationsByContract retrieves events within a given time range
// and returns a map of contractDID -> allocation count (based on CreateAllocationEvent).
func (s *Store) CountAllocationsByContract(start, end time.Time) (map[string]int, error) {
	usages, err := s.GetEventsByDateRange(start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by date range: %w", err)
	}

	contractCounts := make(map[string]int)

	for _, usage := range usages {
		var base struct {
			Type events.EventType `json:"type"`
		}
		if err := json.Unmarshal(usage.Data, &base); err != nil {
			continue // skip invalid payloads
		}

		if base.Type == events.CreateAllocationEvent {
			contractCounts[usage.ContractDID]++
		}
	}

	return contractCounts, nil
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
