// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package resources

import (
	"fmt"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/types"
)

func TestNewStore(t *testing.T) {
	t.Parallel()

	s := newStore()
	require.NotNil(t, s)
}

func TestStore_WithDemandLocks(t *testing.T) {
	t.Parallel()

	s := newStore()
	mockDemand := types.ResourceAllocation{
		AllocationID: "alloc1",
		Resources: types.Resources{
			CPU: types.CPU{
				Cores:      4,
				ClockSpeed: 3000,
			},
			RAM:  types.RAM{Size: 1024},
			Disk: types.Disk{Size: 1024},
		},
	}

	s.withAllocationsLock(func() {
		s.allocations[mockDemand.AllocationID] = mockDemand
	})

	var (
		demand types.ResourceAllocation
		ok     bool
	)
	s.withAllocationsRLock(func() {
		demand, ok = s.allocations[mockDemand.AllocationID]
	})
	require.Truef(t, ok, "expected allocations to be present in the store")
	require.Equalf(t, mockDemand, demand, "expected allocations to be %v, got %v", mockDemand, demand)
}

func TestStore_WithOnboardedLocks(t *testing.T) {
	t.Parallel()

	s := newStore()
	mockOnboarded := types.OnboardedResources{
		Resources: types.Resources{
			CPU: types.CPU{
				Cores:      4,
				ClockSpeed: 3000,
			},
			RAM:  types.RAM{Size: 1024},
			Disk: types.Disk{Size: 1024},
		},
	}

	err := s.withOnboardedLock(func() error {
		s.onboardedResources = &mockOnboarded
		return nil
	})
	require.NoError(t, err)

	var onboarded types.OnboardedResources
	s.withOnboardedRLock(func() {
		onboarded = *s.onboardedResources
	})
	require.Equal(t, mockOnboarded, onboarded)
}

func TestStore_WithCommittedLocks(t *testing.T) {
	t.Parallel()

	s := newStore()
	mockCommitted := types.CommittedResources{
		Resources: types.Resources{
			CPU: types.CPU{
				Cores:      4,
				ClockSpeed: 3000,
			},
			RAM:  types.RAM{Size: 1024},
			Disk: types.Disk{Size: 1024},
		},
	}

	s.withCommittedLock(func() {
		s.committedResources["alloc1"] = &mockCommitted
	})

	var committed types.CommittedResources
	s.withCommittedLock(func() {
		committed = *s.committedResources["alloc1"]
	})

	require.Equal(t, mockCommitted, committed)
}

func TestStore_Concurrency(t *testing.T) {
	t.Parallel()
	const numGoroutines = 50

	t.Run("WithDemandLocks", func(t *testing.T) {
		t.Parallel()

		s := newStore()
		var wg sync.WaitGroup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			index := i
			go func() {
				defer wg.Done()

				demand := types.ResourceAllocation{
					AllocationID: fmt.Sprintf("allocation%d", index),
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      4,
							ClockSpeed: 3000,
						},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 1024},
					},
				}
				s.withAllocationsLock(func() {
					s.allocations[demand.AllocationID] = demand
				})

				var (
					d  types.ResourceAllocation
					ok bool
				)
				s.withAllocationsRLock(func() {
					d, ok = s.allocations[demand.AllocationID]
				})
				require.Truef(t, ok, "expected allocations to be present in the store")
				require.Equalf(t, demand, d, "expected allocations to be %v, got %v", demand, d)

				// Remove the allocations from the store
				s.withAllocationsLock(func() {
					delete(s.allocations, demand.AllocationID)
				})

				s.withAllocationsRLock(func() {
					_, ok = s.allocations[demand.AllocationID]
				})
				require.Falsef(t, ok, "expected allocations to be removed from the store")
			}()
		}

		wg.Wait()
	})

	t.Run("WithOnboardedLocks", func(t *testing.T) {
		t.Parallel()
		s := newStore()
		var wg sync.WaitGroup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				onboarded := types.OnboardedResources{
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      4,
							ClockSpeed: 3000,
						},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 1024},
					},
				}

				err := s.withOnboardedLock(func() error {
					s.onboardedResources = &onboarded
					return nil
				})
				require.NoError(t, err)

				var o types.OnboardedResources
				s.withOnboardedRLock(func() {
					o = *s.onboardedResources
				})
				require.Equal(t, onboarded, o)
			}()
		}

		wg.Wait()
	})

	t.Run("WithCommittedLocks", func(t *testing.T) {
		t.Parallel()
		s := newStore()
		var wg sync.WaitGroup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				committed := types.CommittedResources{
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      4,
							ClockSpeed: 3000,
						},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 1024},
					},
				}

				s.withCommittedLock(func() {
					s.committedResources["alloc1"] = &committed
				})

				var c types.CommittedResources
				s.withCommittedRLock(func() {
					c = *s.committedResources["alloc1"]
				})
				require.Equal(t, committed, c)
			}()
		}

		wg.Wait()
	})
}
