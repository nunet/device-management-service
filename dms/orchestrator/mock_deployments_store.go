// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

// mockDeploymentStore implements DeploymentStore for testing
type mockDeploymentStore struct {
	mu          sync.RWMutex
	deployments map[string]*jtypes.OrchestratorView
}

// NewMockDeploymentStore creates a new mock deployment store for testing
func NewMockDeploymentStore() DeploymentStore {
	return &mockDeploymentStore{
		deployments: make(map[string]*jtypes.OrchestratorView),
	}
}

func (m *mockDeploymentStore) Upsert(deployment *jtypes.OrchestratorView) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	if existing, exists := m.deployments[deployment.OrchestratorID]; exists {
		deployment.CreatedAt = existing.CreatedAt
		deployment.UpdatedAt = now
		if (deployment.Status == jtypes.DeploymentStatusCompleted ||
			deployment.Status == jtypes.DeploymentStatusFailed) &&
			existing.Status != deployment.Status {
			deployment.CompletedAt = &now
		} else {
			deployment.CompletedAt = existing.CompletedAt
		}
	} else {
		deployment.CreatedAt = now
		deployment.UpdatedAt = now
	}

	m.deployments[deployment.OrchestratorID] = deployment
	return nil
}

func (m *mockDeploymentStore) Get(orchestratorID string) (*jtypes.OrchestratorView, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	deployment, exists := m.deployments[orchestratorID]
	if !exists {
		return nil, fmt.Errorf("deployment not found")
	}

	return deployment, nil
}

func (m *mockDeploymentStore) GetAll(statusFilter *jtypes.DeploymentStatus) ([]*jtypes.OrchestratorView, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var result []*jtypes.OrchestratorView
	for _, deployment := range m.deployments {
		if statusFilter == nil || deployment.Status == *statusFilter {
			result = append(result, deployment)
		}
	}

	return result, nil
}

func (m *mockDeploymentStore) Delete(orchestratorID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.deployments[orchestratorID]; !exists {
		return fmt.Errorf("deployment not found")
	}

	delete(m.deployments, orchestratorID)
	return nil
}

func (m *mockDeploymentStore) Prune(olderThan time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, deployment := range m.deployments {
		if deployment.CreatedAt.Before(olderThan) {
			delete(m.deployments, id)
		}
	}

	return nil
}

func (m *mockDeploymentStore) Clear() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.deployments = make(map[string]*jtypes.OrchestratorView)
	return nil
}

func (m *mockDeploymentStore) Query(q DeploymentQuery) ([]*jtypes.OrchestratorView, int, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Collect all deployments
	allDeployments := make([]*jtypes.OrchestratorView, 0, len(m.deployments))
	for _, deployment := range m.deployments {
		allDeployments = append(allDeployments, deployment)
	}

	// Apply filters
	filtered := make([]*jtypes.OrchestratorView, 0)
	for _, deployment := range allDeployments {
		// Status filter
		if len(q.StatusFilter) > 0 {
			found := false
			for _, status := range q.StatusFilter {
				if deployment.Status == status {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Date filters
		if q.CreatedAfter != nil && deployment.CreatedAt.Before(*q.CreatedAfter) {
			continue
		}
		if q.CreatedBefore != nil && deployment.CreatedAt.After(*q.CreatedBefore) {
			continue
		}
		if q.UpdatedAfter != nil && deployment.UpdatedAt.Before(*q.UpdatedAfter) {
			continue
		}
		if q.UpdatedBefore != nil && deployment.UpdatedAt.After(*q.UpdatedBefore) {
			continue
		}

		filtered = append(filtered, deployment)
	}

	// Count total before sorting/pagination
	total := len(filtered)

	// Apply sorting
	if q.SortBy != "" {
		sortField := q.SortBy
		descending := strings.HasPrefix(sortField, "-")
		if descending {
			sortField = sortField[1:]
		}

		sort.Slice(filtered, func(i, j int) bool {
			var less bool
			switch mapSortField(sortField) {
			case "created_at":
				less = filtered[i].CreatedAt.Before(filtered[j].CreatedAt)
			case "updated_at":
				less = filtered[i].UpdatedAt.Before(filtered[j].UpdatedAt)
			case "status":
				less = int(filtered[i].Status) < int(filtered[j].Status)
			default:
				less = false
			}
			if descending {
				return !less
			}
			return less
		})
	} else {
		// Default sort: newest first
		sort.Slice(filtered, func(i, j int) bool {
			return filtered[i].CreatedAt.After(filtered[j].CreatedAt)
		})
	}

	// Apply pagination
	start := q.Offset
	if start < 0 {
		start = 0
	}
	if start > len(filtered) {
		return []*jtypes.OrchestratorView{}, total, nil
	}

	end := len(filtered)
	if q.Limit > 0 && start+q.Limit < end {
		end = start + q.Limit
	}

	if start >= len(filtered) {
		return []*jtypes.OrchestratorView{}, total, nil
	}

	return filtered[start:end], total, nil
}
