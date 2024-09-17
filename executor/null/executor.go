package null

import (
	"context"
	"io"

	"gitlab.com/nunet/device-management-service/executor"
	"gitlab.com/nunet/device-management-service/types"
)

// Executor is a no-op implementation of the Executor interface.
type Executor struct{}

// NewExecutor creates a new Executor.
func NewExecutor(_ context.Context, _ string) (executor.Executor, error) {
	return &Executor{}, nil
}

var _ executor.Executor = (*Executor)(nil)

// Start does nothing and returns nil.
func (e *Executor) Start(_ context.Context, _ *types.ExecutionRequest) error {
	return nil
}

// Run returns a nil result and nil error.
func (e *Executor) Run(_ context.Context, _ *types.ExecutionRequest) (*types.ExecutionResult, error) {
	return nil, nil
}

// Wait returns channels that immediately close.
func (e *Executor) Wait(_ context.Context, _ string) (<-chan *types.ExecutionResult, <-chan error) {
	resultCh := make(chan *types.ExecutionResult)
	errCh := make(chan error)
	close(resultCh)
	close(errCh)
	return resultCh, errCh
}

// Cancel does nothing and returns nil.
func (e *Executor) Cancel(_ context.Context, _ string) error {
	return nil
}

// GetLogStream returns a closed io.ReadCloser and nil error.
func (e *Executor) GetLogStream(_ context.Context, _ types.LogStreamRequest) (io.ReadCloser, error) {
	return io.NopCloser(nil), nil
}

// List returns an empty slice of ExecutionListItem.
func (e *Executor) List() []types.ExecutionListItem {
	return []types.ExecutionListItem{}
}

// Cleanup does nothing and returns nil.
func (e *Executor) Cleanup(_ context.Context) error {
	return nil
}
