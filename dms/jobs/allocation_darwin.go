//go:build darwin
// +build darwin

package jobs

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/types"
)

func (a *Allocation) createExecutor(ctx context.Context, conf types.SpecConfig) error {
	if conf.Type == string(types.ExecutorTypeDocker) {
		executor, err := docker.NewExecutor(ctx, a.executionID)
		if err != nil {
			return fmt.Errorf("docker executor: %w", err)
		}
		a.executor = executor
		return nil
	}

	return fmt.Errorf("unsupported executor type: %s", conf.Type)
}
