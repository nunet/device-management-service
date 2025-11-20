// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package store

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/document"
	"github.com/ostafen/clover/v2/query"
	"gitlab.com/nunet/device-management-service/gateway/provider"
)

const provisionedResourcesCollection = "provisioned_resources"

type ProvisionedResources struct {
	ProviderName        string
	Orchestrator        string
	ProvisionedVMPeerID string
	Resource            provider.Server
	CreatedAt           time.Time
}

type Store struct {
	db *clover.DB
}

// New store
func New(db *clover.DB) (*Store, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}

	return &Store{
		db: db,
	}, nil
}

// Insert
func (s *Store) Insert(r *ProvisionedResources) error {
	if r == nil {
		return errors.New("resources data is nil")
	}

	bts, err := json.Marshal(r)
	if err != nil {
		return fmt.Errorf("failed to marshal contract: %w", err)
	}

	// Insert a new document
	doc := document.NewDocumentOf(r)
	doc.Set("id", r.Resource.ID)
	doc.Set("created_at", time.Now().UnixNano())
	doc.Set("data", bts)

	return s.db.Insert(provisionedResourcesCollection, doc)
}

// All retrieves all provisioned resources from the database
func (s *Store) All() ([]*ProvisionedResources, error) {
	q := query.NewQuery(provisionedResourcesCollection)

	docs, err := s.db.FindAll(q)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve all provisioned resources: %w", err)
	}

	all := make([]*ProvisionedResources, 0)
	for _, doc := range docs {
		var res ProvisionedResources
		data := doc.Get("data")
		err = json.Unmarshal(data.([]byte), &res)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal single resource: %w", err)
		}

		all = append(all, &res)
	}

	return all, nil
}

// Delete removes a provisioned resource by its Resource.ID
func (s *Store) Delete(resourceID string) error {
	if resourceID == "" {
		return errors.New("resourceID is empty")
	}

	q := query.NewQuery(provisionedResourcesCollection).
		Where(query.Field("id").Eq(resourceID))

	err := s.db.Delete(q)
	if err != nil {
		return fmt.Errorf("failed to delete resource with id %s: %w", resourceID, err)
	}

	return nil
}
