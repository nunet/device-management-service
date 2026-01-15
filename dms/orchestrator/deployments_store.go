// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/ostafen/clover/v2"
	"github.com/ostafen/clover/v2/document"
	"github.com/ostafen/clover/v2/query"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

const (
	deploymentsCollection = "deployments"
	sortFieldCreatedAt    = "created_at"
	sortFieldUpdatedAt    = "updated_at"
	sortFieldStatus       = "status"
)

// DeploymentQuery defines query parameters for advanced deployment filtering
type DeploymentQuery struct {
	StatusFilter  []jtypes.DeploymentStatus
	CreatedAfter  *time.Time
	CreatedBefore *time.Time
	UpdatedAfter  *time.Time
	UpdatedBefore *time.Time
	Limit         int
	Offset        int
	SortBy        string // e.g., "created_at", "-created_at"
}

// DeploymentStore defines the interface for persisting orchestrator deployments
type DeploymentStore interface {
	// Upsert saves or updates a deployment
	Upsert(deployment *jtypes.OrchestratorView) error

	// Get retrieves a deployment by ID
	Get(orchestratorID string) (*jtypes.OrchestratorView, error)

	// GetAll retrieves all deployments, optionally filtered by status
	GetAll(statusFilter *jtypes.DeploymentStatus) ([]*jtypes.OrchestratorView, error)

	// Query retrieves deployments with advanced filtering, pagination, and sorting
	Query(query DeploymentQuery) ([]*jtypes.OrchestratorView, int, error)
	// Returns: deployments, total count, error

	// Delete removes a deployment by ID
	Delete(orchestratorID string) error

	// Prune removes deployments older than the specified time
	Prune(olderThan time.Time) error

	// Clear removes all deployments
	Clear() error
}

// cloverDeploymentStore implements DeploymentStore using CloverDB with bytes serialization
type cloverDeploymentStore struct {
	db *clover.DB
}

// NewCloverDeploymentStore creates a new cloverDB-backed deployment store
func NewCloverDeploymentStore(db *clover.DB) (DeploymentStore, error) {
	if db == nil {
		return nil, errors.New("db is nil")
	}

	return &cloverDeploymentStore{db: db}, nil
}

// Upsert inserts or updates a deployment in CloverDB (Upsert behavior)
func (s *cloverDeploymentStore) Upsert(deployment *jtypes.OrchestratorView) error {
	if deployment == nil {
		return errors.New("deployment is nil")
	}

	// Marshal the deployment to bytes
	bts, err := json.Marshal(deployment)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment: %w", err)
	}

	// Check if deployment exists
	q := query.NewQuery(deploymentsCollection).Where(query.Field("orchestrator_id").Eq(deployment.OrchestratorID))
	existingDoc, err := s.db.FindFirst(q)
	if err != nil {
		return fmt.Errorf("failed to check existing deployment: %w", err)
	}

	now := time.Now()

	if existingDoc != nil {
		// Update the existing document
		update := document.NewDocument()
		update.Set("orchestrator_id", deployment.OrchestratorID)
		update.Set("status", int(deployment.Status))
		update.Set("updated_at", now.Unix())
		update.Set("deployment_data", bts)

		// Set completion time if status changed to terminal
		if deployment.Status == jtypes.DeploymentStatusCompleted ||
			deployment.Status == jtypes.DeploymentStatusFailed {
			update.Set("completed_at", now.Unix())
		}

		return s.db.Update(q, update.AsMap())
	}

	// Insert a new document
	doc := document.NewDocument()
	doc.Set("orchestrator_id", deployment.OrchestratorID)
	doc.Set("status", int(deployment.Status))
	doc.Set("created_at", now.Unix())
	doc.Set("updated_at", now.Unix())
	doc.Set("deployment_data", bts)

	// Set completion time if it's already in a terminal state
	if deployment.Status == jtypes.DeploymentStatusCompleted ||
		deployment.Status == jtypes.DeploymentStatusFailed {
		doc.Set("completed_at", now.Unix())
	}

	return s.db.Insert(deploymentsCollection, doc)
}

// Get retrieves a deployment by OrchestratorID
func (s *cloverDeploymentStore) Get(orchestratorID string) (*jtypes.OrchestratorView, error) {
	if orchestratorID == "" {
		return nil, errors.New("orchestratorID is empty")
	}

	q := query.NewQuery(deploymentsCollection).Where(query.Field("orchestrator_id").Eq(orchestratorID))
	doc, err := s.db.FindFirst(q)
	if err != nil || doc == nil {
		return nil, fmt.Errorf("failed to find deployment by ID: %w", err)
	}

	var deployment jtypes.OrchestratorView
	data := doc.Get("deployment_data")
	deploymentData, ok := data.([]byte)
	if !ok {
		return nil, errors.New("no deployment data available")
	}

	err = json.Unmarshal(deploymentData, &deployment)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal deployment: %w", err)
	}

	return &deployment, nil
}

// GetAll retrieves all deployments, optionally filtered by status
func (s *cloverDeploymentStore) GetAll(statusFilter *jtypes.DeploymentStatus) ([]*jtypes.OrchestratorView, error) {
	q := query.NewQuery(deploymentsCollection)

	if statusFilter != nil {
		q = q.Where(query.Field("status").Eq(int(*statusFilter)))
	}

	docs, err := s.db.FindAll(q)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve all deployments: %w", err)
	}

	allDeployments := make([]*jtypes.OrchestratorView, 0, len(docs))
	for _, doc := range docs {
		var deployment jtypes.OrchestratorView
		data := doc.Get("deployment_data")
		deploymentData, ok := data.([]byte)
		if !ok {
			return nil, fmt.Errorf("no deployment data available for document")
		}

		err = json.Unmarshal(deploymentData, &deployment)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal single deployment: %w", err)
		}

		allDeployments = append(allDeployments, &deployment)
	}

	return allDeployments, nil
}

// Delete removes a deployment by OrchestratorID
func (s *cloverDeploymentStore) Delete(orchestratorID string) error {
	if orchestratorID == "" {
		return errors.New("orchestratorID is empty")
	}

	q := query.NewQuery(deploymentsCollection).Where(query.Field("orchestrator_id").Eq(orchestratorID))
	return s.db.Delete(q)
}

// Prune removes deployments older than the specified time
func (s *cloverDeploymentStore) Prune(olderThan time.Time) error {
	q := query.NewQuery(deploymentsCollection).Where(query.Field("created_at").Lt(olderThan.Unix()))
	return s.db.Delete(q)
}

// Clear removes all deployments
func (s *cloverDeploymentStore) Clear() error {
	q := query.NewQuery(deploymentsCollection)
	return s.db.Delete(q)
}

// Query retrieves deployments with advanced filtering, pagination, and sorting
func (s *cloverDeploymentStore) Query(q DeploymentQuery) ([]*jtypes.OrchestratorView, int, error) {
	// Build combined condition using the same pattern as QueryEvents
	// This ensures all conditions are properly combined as AND conditions
	var combinedCondition query.Criteria
	hasCondition := false

	// Status filter
	if len(q.StatusFilter) > 0 {
		statusInts := make([]interface{}, len(q.StatusFilter))
		for i, status := range q.StatusFilter {
			statusInts[i] = int(status)
		}
		statusCondition := query.Field("status").In(statusInts...)
		if !hasCondition {
			combinedCondition = statusCondition
		} else {
			combinedCondition = combinedCondition.And(statusCondition)
		}
		hasCondition = true
	}

	// Date filters
	if q.CreatedAfter != nil {
		createdAfterCondition := query.Field("created_at").GtEq(q.CreatedAfter.Unix())
		if !hasCondition {
			combinedCondition = createdAfterCondition
		} else {
			combinedCondition = combinedCondition.And(createdAfterCondition)
		}
		hasCondition = true
	}

	if q.CreatedBefore != nil {
		createdBeforeCondition := query.Field("created_at").LtEq(q.CreatedBefore.Unix())
		if !hasCondition {
			combinedCondition = createdBeforeCondition
		} else {
			combinedCondition = combinedCondition.And(createdBeforeCondition)
		}
		hasCondition = true
	}

	if q.UpdatedAfter != nil {
		updatedAfterCondition := query.Field("updated_at").GtEq(q.UpdatedAfter.Unix())
		if !hasCondition {
			combinedCondition = updatedAfterCondition
		} else {
			combinedCondition = combinedCondition.And(updatedAfterCondition)
		}
		hasCondition = true
	}

	if q.UpdatedBefore != nil {
		updatedBeforeCondition := query.Field("updated_at").LtEq(q.UpdatedBefore.Unix())
		if !hasCondition {
			combinedCondition = updatedBeforeCondition
		} else {
			combinedCondition = combinedCondition.And(updatedBeforeCondition)
		}
		hasCondition = true
	}

	// Build base query for counting (without sorting/pagination)
	baseQuery := query.NewQuery(deploymentsCollection)
	if hasCondition {
		baseQuery = baseQuery.Where(combinedCondition)
	}

	// Count total using base query
	totalDocs, err := s.db.Count(baseQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count deployments: %w", err)
	}

	// Build fetch query with sorting and pagination
	fetchQuery := query.NewQuery(deploymentsCollection)
	if hasCondition {
		fetchQuery = fetchQuery.Where(combinedCondition)
	}

	// Apply sorting
	if q.SortBy != "" {
		sortField := q.SortBy
		direction := 1
		if strings.HasPrefix(sortField, "-") {
			direction = -1
			sortField = sortField[1:]
		}
		// Map field names to DB fields
		dbField := mapSortField(sortField)
		fetchQuery = fetchQuery.Sort(query.SortOption{Field: dbField, Direction: direction})
	} else {
		// Default sort: newest first
		fetchQuery = fetchQuery.Sort(query.SortOption{Field: "created_at", Direction: -1})
	}

	// Apply limit to query (if we have offset, we need offset+limit to get enough results)
	// If no limit specified, we still want to limit to prevent huge results
	effectiveLimit := q.Limit
	if effectiveLimit <= 0 {
		effectiveLimit = 1000 // Default max to prevent unbounded queries
	}
	if q.Offset > 0 {
		// Need to fetch offset+limit documents, then skip the first offset
		fetchQuery = fetchQuery.Limit(q.Offset + effectiveLimit)
	} else {
		fetchQuery = fetchQuery.Limit(effectiveLimit)
	}

	// Execute query
	allDocs, err := s.db.FindAll(fetchQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query deployments: %w", err)
	}

	// Handle offset manually (slice the results)
	var docs []*document.Document
	if q.Offset > 0 {
		if q.Offset < len(allDocs) {
			docs = allDocs[q.Offset:]
			// Apply limit after offset
			if q.Limit > 0 && len(docs) > q.Limit {
				docs = docs[:q.Limit]
			}
		}
		// else: offset beyond results, docs remains empty
	} else {
		docs = allDocs
	}

	// Unmarshal results
	deployments := make([]*jtypes.OrchestratorView, 0, len(docs))
	for _, doc := range docs {
		var deployment jtypes.OrchestratorView
		data := doc.Get("deployment_data")
		deploymentData, ok := data.([]byte)
		if !ok {
			continue
		}
		if err := json.Unmarshal(deploymentData, &deployment); err != nil {
			continue
		}
		deployments = append(deployments, &deployment)
	}

	return deployments, totalDocs, nil
}

// mapSortField maps sort field names to DB field names
func mapSortField(field string) string {
	switch field {
	case sortFieldCreatedAt, "createdAt":
		return sortFieldCreatedAt
	case sortFieldUpdatedAt, "updatedAt":
		return sortFieldUpdatedAt
	case sortFieldStatus:
		return sortFieldStatus
	default:
		return field
	}
}
