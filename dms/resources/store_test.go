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

func Test_WithDemandLocks(t *testing.T) {
	t.Parallel()

	s := newStore()
	mockDemand := types.ResourceAllocation{
		JobID: "job1",
		Resources: types.Resources{
			CPU: types.CPU{
				Cores:      4,
				ClockSpeed: 3000,
				Compute:    12000,
			},
			RAM:  types.RAM{Size: 1024},
			Disk: types.Disk{Size: 1024},
		},
	}

	s.withAllocationsLock(func() {
		s.allocations[mockDemand.JobID] = mockDemand
	})

	var (
		demand types.ResourceAllocation
		ok     bool
	)
	s.withAllocationsRLock(func() {
		demand, ok = s.allocations[mockDemand.JobID]
	})
	require.Truef(t, ok, "expected allocations to be present in the store")
	require.Equalf(t, mockDemand, demand, "expected allocations to be %v, got %v", mockDemand, demand)
}

func Test_WithOnboardedLocks(t *testing.T) {
	t.Parallel()

	s := newStore()
	mockOnboarded := types.OnboardedResources{
		Resources: types.Resources{
			CPU: types.CPU{
				Cores:      4,
				ClockSpeed: 3000,
				Compute:    12000,
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

func Test_WithFreeLocks(t *testing.T) {
	t.Parallel()

	s := newStore()
	mockFree := types.FreeResources{
		Resources: types.Resources{
			CPU: types.CPU{
				Cores:      4,
				ClockSpeed: 3000,
				Compute:    12000,
			},
			RAM:  types.RAM{Size: 1024},
			Disk: types.Disk{Size: 1024},
		},
	}

	s.withFreeLock(func() {
		s.freeResources = &mockFree
	})

	var free types.FreeResources
	s.withFreeRLock(func() {
		free = *s.freeResources
	})
	require.Equal(t, mockFree, free)
}

func Test_Concurrency(t *testing.T) {
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
					JobID: fmt.Sprintf("job%d", index),
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      4,
							ClockSpeed: 3000,
							Compute:    12000,
						},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 1024},
					},
				}
				s.withAllocationsLock(func() {
					s.allocations[demand.JobID] = demand
				})

				var (
					d  types.ResourceAllocation
					ok bool
				)
				s.withAllocationsRLock(func() {
					d, ok = s.allocations[demand.JobID]
				})
				require.Truef(t, ok, "expected allocations to be present in the store")
				require.Equalf(t, demand, d, "expected allocations to be %v, got %v", demand, d)

				// Remove the allocations from the store
				s.withAllocationsLock(func() {
					delete(s.allocations, demand.JobID)
				})

				s.withAllocationsRLock(func() {
					_, ok = s.allocations[demand.JobID]
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
							Compute:    12000,
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

	t.Run("WithFreeLocks", func(t *testing.T) {
		t.Parallel()
		s := newStore()
		var wg sync.WaitGroup
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()

				free := types.FreeResources{
					Resources: types.Resources{
						CPU: types.CPU{
							Cores:      4,
							ClockSpeed: 3000,
							Compute:    12000,
						},
						RAM:  types.RAM{Size: 1024},
						Disk: types.Disk{Size: 1024},
					},
				}

				s.withFreeLock(func() {
					s.freeResources = &free
				})

				var f types.FreeResources
				s.withFreeRLock(func() {
					f = *s.freeResources
				})
				require.Equal(t, free, f)
			}()
		}

		wg.Wait()
	})
}
