// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
