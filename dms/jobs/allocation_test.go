package jobs

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/types"
)

func TestNewAllocation(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		actor           *dms.BasicActor
		alloc           AllocationDetails
		resourceManager resources.Manager
		expErr          string
	}{
		"no resource manager": {
			expErr: "resource manager is nil",
		},
		"success": {
			actor:           &dms.BasicActor{},
			alloc:           AllocationDetails{},
			resourceManager: &resourceManagerMock{},
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			allocation, err := NewAllocation(tt.actor, tt.alloc, tt.resourceManager)
			if tt.expErr != "" {
				assert.Nil(t, allocation)
				assert.EqualError(t, err, tt.expErr)
			} else {
				assert.NotNil(t, allocation)
				assert.Equal(t, pending, allocation.status)
			}
		})
	}
}

func TestAllocationRunStatus(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		allocationBuilder func() *Allocation
		status            AllocationStatus
		expErr            string
	}{
		"issue getting resource manager": {
			allocationBuilder: func() *Allocation {
				alloc, err := NewAllocation(&dms.BasicActor{}, AllocationDetails{}, &resourceManagerMock{
					err: errors.New("internal error"),
				})
				assert.NoError(t, err)
				return alloc
			},
			status: pending,
			expErr: "failed to get free resources: internal error",
		},
		"no free resources": {
			allocationBuilder: func() *Allocation {
				// job resources
				wantJob := Job{
					ID:   "123",
					Name: "Job name",
					Resources: types.ExecutionResources{
						CPU: types.CPU{
							Cores: 3,
						},
						Disk: types.Disk{Size: 2000000000},
					},
				}

				alloc, err := NewAllocation(&dms.BasicActor{}, AllocationDetails{Job: wantJob}, &resourceManagerMock{freeRes: types.FreeResources{
					Resources: types.Resources{
						CPU:      1,
						NumCores: 1,
						RAM:      10,
						Disk:     10,
					},
				}})
				assert.NoError(t, err)
				return alloc
			},
			status: pending,
			expErr: "no available resources for job 123",
		},
		"execution failed": {
			allocationBuilder: func() *Allocation {
				// job resources
				wantJob := Job{
					ID:   "123",
					Name: "Job name",
					Resources: types.ExecutionResources{
						CPU: types.CPU{
							Cores: 1,
						},
						Disk: types.Disk{Size: 10},
					},
				}

				alloc, err := NewAllocation(&dms.BasicActor{}, AllocationDetails{Job: wantJob}, &resourceManagerMock{freeRes: types.FreeResources{
					Resources: types.Resources{
						CPU:      100,
						NumCores: 6,
						RAM:      10000000000,
						Disk:     100000000000,
					},
				}})
				assert.NoError(t, err)

				// mocket executor
				alloc.executor = &mockExecutor{err: errors.New("internal error")}

				return alloc
			},
			status: pending,
			expErr: "failed to start executor: internal error",
		},
		"successful": {
			allocationBuilder: func() *Allocation {
				// job resources
				wantJob := Job{
					ID:   "123",
					Name: "Job name",
					Resources: types.ExecutionResources{
						CPU: types.CPU{
							Cores: 1,
						},
						Disk: types.Disk{Size: 10},
					},
				}

				alloc, err := NewAllocation(&dms.BasicActor{}, AllocationDetails{Job: wantJob}, &resourceManagerMock{freeRes: types.FreeResources{
					Resources: types.Resources{
						CPU:      100,
						NumCores: 6,
						RAM:      10000000000,
						Disk:     100000000000,
					},
				}})
				assert.NoError(t, err)

				// mocket executor
				alloc.executor = &mockExecutor{}

				return alloc
			},
			status: running,
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			alloc := tt.allocationBuilder()
			err := alloc.Run(context.Background())
			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
			}
			sts := alloc.Status(context.TODO())
			assert.Equal(t, sts.Status, alloc.status)
		})
	}
}

func TestAllocationStop(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		allocationBuilder func() *Allocation
		status            AllocationStatus
		expErr            string
	}{
		"stop non running allocation": {
			allocationBuilder: func() *Allocation {
				alloc, err := NewAllocation(&dms.BasicActor{}, AllocationDetails{}, &resourceManagerMock{})
				assert.NoError(t, err)

				// mocket executor
				alloc.executor = &mockExecutor{err: errors.New("cancel failed")}

				return alloc
			},
			status: running,
			expErr: "allocation is not running",
		},
		"execution failed to cancel": {
			allocationBuilder: func() *Allocation {
				alloc, err := NewAllocation(&dms.BasicActor{}, AllocationDetails{}, &resourceManagerMock{})
				assert.NoError(t, err)

				// mocket executor
				alloc.executor = &mockExecutor{err: errors.New("cancel failed")}
				alloc.status = running

				return alloc
			},
			status: running,
			expErr: "failed to stop execution: cancel failed",
		},
		"failed to update free resources after stoping allocation": {
			allocationBuilder: func() *Allocation {
				alloc, err := NewAllocation(&dms.BasicActor{}, AllocationDetails{}, &resourceManagerMock{
					err: errors.New("failed to update resources"),
				})
				assert.NoError(t, err)

				// mocket executor
				alloc.executor = &mockExecutor{}
				alloc.status = running

				return alloc
			},
			status: running,
			expErr: "failed to update resources after stoping allocation's executor: failed to update resources",
		},
		"success": {
			allocationBuilder: func() *Allocation {
				alloc, err := NewAllocation(&dms.BasicActor{}, AllocationDetails{}, &resourceManagerMock{})
				assert.NoError(t, err)

				alloc.executor = &mockExecutor{}
				alloc.status = running

				return alloc
			},
			status: stopped,
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			alloc := tt.allocationBuilder()
			err := alloc.Stop(context.Background())
			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
			}
			sts := alloc.Status(context.TODO())
			assert.Equal(t, sts.Status, alloc.status)
		})
	}
}

type resourceManagerMock struct {
	freeRes            types.FreeResources
	onboardedResources types.OnboardedResources

	err error
}

func (m *resourceManagerMock) UpdateFreeResources(context.Context) (types.FreeResources, error) {
	return m.freeRes, m.err
}

func (m *resourceManagerMock) UpdateOnboardedResources(context.Context, types.OnboardedResources) error {
	return m.err
}

func (m *resourceManagerMock) GetOnboardedResources(context.Context) (types.OnboardedResources, error) {
	return m.onboardedResources, m.err
}

func (m *resourceManagerMock) GetRequiredResources(context.Context) (types.Resources, error) {
	return m.freeRes.Resources, m.err
}

func (m *resourceManagerMock) SystemSpecs() resources.SystemSpecs {
	return nil
}

func (m *resourceManagerMock) UsageMonitor() resources.UsageMonitor {
	return nil
}

type mockExecutor struct {
	isInstalled bool
	res         *types.ExecutionResult
	err         error
	errChan     <-chan error
	resChan     <-chan *types.ExecutionResult
	logStream   io.ReadCloser
}

func (m *mockExecutor) IsInstalled(_ context.Context) bool {
	return m.isInstalled
}

func (m *mockExecutor) Start(_ context.Context, _ *types.ExecutionRequest) error {
	return m.err
}

func (m *mockExecutor) Run(_ context.Context, _ *types.ExecutionRequest) (*types.ExecutionResult, error) {
	return m.res, m.err
}

func (m *mockExecutor) Wait(_ context.Context, _ string) (<-chan *types.ExecutionResult, <-chan error) {
	return m.resChan, m.errChan
}

func (m *mockExecutor) Cancel(_ context.Context, _ string) error {
	return m.err
}

func (m *mockExecutor) GetLogStream(_ context.Context, _ types.LogStreamRequest) (io.ReadCloser, error) {
	return m.logStream, m.err
}
