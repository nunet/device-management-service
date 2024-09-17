//go:build linux
// +build linux

package node

import (
	"context"
	"errors"
	"fmt"

	"gitlab.com/nunet/device-management-service/executor"
	"gitlab.com/nunet/device-management-service/executor/firecracker"
	"gitlab.com/nunet/device-management-service/executor/null"
)

func NewExecutor(ctx context.Context) (executor.Executor, error) {
	executor, err := firecracker.NewExecutor(ctx, "root")
	if err != nil {
		if errors.Is(err, firecracker.ErrNotInstalled) {
			executor, err := null.NewExecutor(ctx, "root")
			if err != nil {
				return nil, fmt.Errorf("failed to setup null executor: %w", err)
			}
			return executor, nil
		}

		return nil, fmt.Errorf("failed to create executor: %w", err)
	}

	return executor, nil
}
