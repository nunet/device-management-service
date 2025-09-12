package payment

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/document"
	"github.com/ostafen/clover/v2/query"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

const (
	paymentsCollection = "contracts_payments"
)

type Payment struct {
	UniqueID string             `json:"unique_id"`
	Contract contracts.Contract `json:"contract"`
	Usages   int                `json:"usages"`
	Amount   string             `json:"amount"`
	Paid     bool               `json:"paid"`
}

type Store struct {
	db *clover.DB
}

// New payment store
func New(db *clover.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}

	return &Store{
		db: db,
	}, nil
}

func (s *Store) Insert(p Payment) error {
	bts, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal payment: %w", err)
	}

	q := query.NewQuery(paymentsCollection).Where(query.Field("unique_id").Eq(p.UniqueID))
	existingDoc, err := s.db.FindFirst(q)
	if err != nil {
		return fmt.Errorf("failed to check existing payment: %w", err)
	}

	if existingDoc != nil {
		// payment with this unique_id already exists
		return nil
	}

	doc := document.NewDocumentOf(p)
	doc.Set("unique_id", p.UniqueID)
	doc.Set("contract_did", p.Contract.ContractDID)
	doc.Set("created_at", time.Now().Unix())
	doc.Set("payment_data", bts)

	return s.db.Insert(paymentsCollection, doc)
}

// AllPayments retrieves all payments from the database
func (s *Store) AllPayments() ([]*Payment, error) {
	q := query.NewQuery(paymentsCollection)

	docs, err := s.db.FindAll(q)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve all payments: %w", err)
	}

	allPayments := make([]*Payment, 0)
	for _, doc := range docs {
		var payment Payment
		data := doc.Get("payment_data")
		err = json.Unmarshal(data.([]byte), &payment)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal single payment: %w", err)
		}

		allPayments = append(allPayments, &payment)
	}

	return allPayments, nil
}

// GetByUniqueID retrieves a payment by its unique_id
func (s *Store) GetByUniqueID(uniqueID string) (*Payment, error) {
	q := query.NewQuery(paymentsCollection).Where(query.Field("unique_id").Eq(uniqueID))

	doc, err := s.db.FindFirst(q)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve payment with unique_id %s: %w", uniqueID, err)
	}

	if doc == nil {
		return nil, fmt.Errorf("payment with unique_id %s not found", uniqueID)
	}

	data, ok := doc.Get("payment_data").([]byte)
	if !ok {
		return nil, fmt.Errorf("invalid data format for payment with unique_id %s", uniqueID)
	}

	var payment Payment
	if err := json.Unmarshal(data, &payment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payment with unique_id %s: %w", uniqueID, err)
	}

	return &payment, nil
}

// Update updates an existing payment by its unique_id.
func (s *Store) Update(p *Payment) error {
	if p.UniqueID == "" {
		return errors.New("unique_id is required for update")
	}

	q := query.NewQuery(paymentsCollection).Where(query.Field("unique_id").Eq(p.UniqueID))
	doc, err := s.db.FindFirst(q)
	if err != nil {
		return fmt.Errorf("failed to query existing payment: %w", err)
	}
	if doc == nil {
		return fmt.Errorf("payment with unique_id %s not found", p.UniqueID)
	}

	bts, err := json.Marshal(p)
	if err != nil {
		return fmt.Errorf("failed to marshal payment: %w", err)
	}

	err = s.db.UpdateById(paymentsCollection, doc.ObjectId(), func(d *document.Document) *document.Document {
		d.Set("payment_data", bts)
		d.Set("updated_at", time.Now().Unix())
		return d
	})
	if err != nil {
		return fmt.Errorf("failed to update payment with unique_id %s: %w", p.UniqueID, err)
	}

	return nil
}
