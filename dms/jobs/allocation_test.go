package jobs

import (
	"context"
	"errors"
	"io"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/dms/actor"
	"gitlab.com/nunet/device-management-service/types"
)

func TestNewAllocation(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		actor                *actor.BasicActor
		alloc                AllocationDetails
		resourceManagerMocks func(ctrl *gomock.Controller) types.ResourceManager
		expErr               string
	}{
		"no resource manager": {
			expErr: "resource manager is nil",
			resourceManagerMocks: func(_ *gomock.Controller) types.ResourceManager {
				return nil
			},
		},
		"success": {
			actor: &actor.BasicActor{},
			alloc: AllocationDetails{},
			resourceManagerMocks: func(ctrl *gomock.Controller) types.ResourceManager {
				return NewMockResourceManager(ctrl)
			},
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resourceManager := tt.resourceManagerMocks(ctrl)
			allocation, err := NewAllocation(tt.actor, tt.alloc, resourceManager)
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
		allocationBuilder    func(resourceManager *MockResourceManager) *Allocation
		status               AllocationStatus
		expErr               string
		resourceManagerMocks func(ctrl *gomock.Controller) *MockResourceManager
	}{
		"no free resources - allocation fails": {
			resourceManagerMocks: func(ctrl *gomock.Controller) *MockResourceManager {
				resourceManager := NewMockResourceManager(ctrl)
				resourceManager.EXPECT().AllocateResources(gomock.Any(), gomock.Any()).Return(errors.New("no available resources for job 123"))
				return resourceManager
			},
			allocationBuilder: func(resourceManager *MockResourceManager) *Allocation {
				// job resources
				wantJob := Job{
					ID:   "123",
					Name: "Job name",
					Resources: types.Resources{
						CPU: types.CPU{
							Cores: 3,
						},
						Disk: types.Disk{Size: 2000000000},
					},
				}

				alloc, err := NewAllocation(&actor.BasicActor{}, AllocationDetails{Job: wantJob}, resourceManager)
				assert.NoError(t, err)
				return alloc
			},
			status: pending,
			expErr: "failed to allocate resources: no available resources for job 123",
		},
		"execution failed": {
			resourceManagerMocks: func(ctrl *gomock.Controller) *MockResourceManager {
				resourceManager := NewMockResourceManager(ctrl)
				resourceManager.EXPECT().AllocateResources(gomock.Any(), gomock.Any()).Return(nil)
				resourceManager.EXPECT().DeallocateResources(gomock.Any(), gomock.Any()).Return(nil)
				return resourceManager
			},
			allocationBuilder: func(resourceManager *MockResourceManager) *Allocation {
				// job resources
				wantJob := Job{
					ID:   "123",
					Name: "Job name",
					Resources: types.Resources{
						CPU: types.CPU{
							Cores: 1,
						},
						Disk: types.Disk{Size: 10},
					},
				}

				alloc, err := NewAllocation(&actor.BasicActor{}, AllocationDetails{Job: wantJob}, resourceManager)
				assert.NoError(t, err)

				// mocket executor
				alloc.executor = &mockExecutor{err: errors.New("internal error")}

				return alloc
			},
			status: pending,
			expErr: "failed to start executor: internal error",
		},
		"successful": {
			resourceManagerMocks: func(ctrl *gomock.Controller) *MockResourceManager {
				resourceManager := NewMockResourceManager(ctrl)
				resourceManager.EXPECT().AllocateResources(gomock.Any(), gomock.Any()).Return(nil)
				return resourceManager
			},
			allocationBuilder: func(resourceManager *MockResourceManager) *Allocation {
				// job resources
				wantJob := Job{
					ID:   "123",
					Name: "Job name",
					Resources: types.Resources{
						CPU: types.CPU{
							Cores: 1,
						},
						Disk: types.Disk{Size: 10},
					},
				}

				alloc, err := NewAllocation(&actor.BasicActor{}, AllocationDetails{Job: wantJob}, resourceManager)
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resourceManager := tt.resourceManagerMocks(ctrl)
			alloc := tt.allocationBuilder(resourceManager)
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
		allocationBuilder    func(resourceManager *MockResourceManager) *Allocation
		resourceManagerMocks func(ctrl *gomock.Controller) *MockResourceManager
		status               AllocationStatus
		expErr               string
	}{
		"execution failed to cancel": {
			allocationBuilder: func(resourceManager *MockResourceManager) *Allocation {
				alloc, err := NewAllocation(&actor.BasicActor{}, AllocationDetails{}, resourceManager)
				assert.NoError(t, err)

				// mocket executor
				alloc.executor = &mockExecutor{err: errors.New("cancel failed")}
				alloc.status = running

				return alloc
			},
			//nolint:gocritic //we need to return the resource manager or change the test structure
			resourceManagerMocks: func(ctrl *gomock.Controller) *MockResourceManager {
				return NewMockResourceManager(ctrl)
			},
			status: running,
			expErr: "failed to stop execution: cancel failed",
		},
		"failed to deallocate resources after stoping allocation": {
			allocationBuilder: func(resourceManager *MockResourceManager) *Allocation {
				alloc, err := NewAllocation(&actor.BasicActor{}, AllocationDetails{}, resourceManager)
				assert.NoError(t, err)

				// mocket executor
				alloc.executor = &mockExecutor{}
				alloc.status = running

				return alloc
			},
			resourceManagerMocks: func(ctrl *gomock.Controller) *MockResourceManager {
				resourceManager := NewMockResourceManager(ctrl)
				resourceManager.EXPECT().DeallocateResources(gomock.Any(), gomock.Any()).Return(errors.New("failed to deallocate resources"))
				return resourceManager
			},
			status: running,
			expErr: "failed to deallocate resources: failed to deallocate resources",
		},
		"success": {
			resourceManagerMocks: func(ctrl *gomock.Controller) *MockResourceManager {
				resourceManager := NewMockResourceManager(ctrl)
				resourceManager.EXPECT().DeallocateResources(gomock.Any(), gomock.Any()).Return(nil)
				return resourceManager
			},
			allocationBuilder: func(resourceManager *MockResourceManager) *Allocation {
				alloc, err := NewAllocation(&actor.BasicActor{}, AllocationDetails{}, resourceManager)
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
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			resourceManager := tt.resourceManagerMocks(ctrl)
			alloc := tt.allocationBuilder(resourceManager)
			err := alloc.Stop(context.Background())
			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
			}
			sts := alloc.Status(context.TODO())
			assert.Equal(t, sts.Status, alloc.status)
		})
	}
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
