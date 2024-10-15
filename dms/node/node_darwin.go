//go:build darwin

package node

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/jobs"

	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/executor/null"
)

func (n *Node) initSupportedExecutors(ctx context.Context) error {
	executor, err := null.NewExecutor(ctx, "root")
	if err != nil {
		return fmt.Errorf("failed to setup null executor: %w", err)
	}

	n.executors[string(jobs.ExecutorNull)] = executorMetadata{
		executor:      executor,
		executionType: jobs.ExecutorNull,
	}

	dockerExec, err := docker.NewExecutor(ctx, "root")
	if err == nil {
		n.executors[string(jobs.ExecutorDocker)] = executorMetadata{
			executor:      dockerExec,
			executionType: jobs.ExecutorDocker,
		}
	}

	return nil
}
