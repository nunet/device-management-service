//go:build linux
// +build linux

package node

import (
	"context"

	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/executor/firecracker"
)

func (n *Node) initSupportedExecutors(ctx context.Context) error {
	executor, err := firecracker.NewExecutor(ctx, "root")
	if err == nil {
		n.executors[string(jobs.ExecutorFirecracker)] = executorMetadata{
			executor:      executor,
			executionType: jobs.ExecutorFirecracker,
		}
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
