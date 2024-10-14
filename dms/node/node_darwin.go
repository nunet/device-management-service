//go:build darwin
// +build darwin

package node

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/executor/null"
)

func (n *Node) initSupportedExecutors(ctx context.Context) error {
	executor, err := null.NewExecutor(ctx, "root")
	if err != nil {
		return fmt.Errorf("failed to setup null executor: %w", err)
	}

	n.executors[string(NullExecutor)] = executorMetadata{
		executor:      executor,
		executionType: NullExecutor,
	}

	dockerExec, err := docker.NewExecutor(ctx, "root")
	if err == nil {
		n.executors[string(DockerExecutor)] = executorMetadata{
			executor:      dockerExec,
			executionType: DockerExecutor,
		}
	}

	return nil
}
