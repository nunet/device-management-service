//go:build darwin
// +build darwin

package node

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/executor"
	"gitlab.com/nunet/device-management-service/executor/null"
)

func NewExecutor(ctx context.Context) (executor.Executor, error) {
	executor, err := null.NewExecutor(ctx, "root")
	if err != nil {
		return nil, fmt.Errorf("failed to setup null executor: %w", err)
	}

	return executor, nil
}
