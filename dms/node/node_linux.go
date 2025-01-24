// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux

package node

import (
	"context"

	job_types "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/executor/firecracker"
)

func (n *Node) initSupportedExecutors(ctx context.Context) error {
	executor, err := firecracker.NewExecutor(ctx, "root")
	if err == nil {
		n.executors[string(job_types.ExecutorFirecracker)] = executorMetadata{
			executor:      executor,
			executionType: job_types.ExecutorFirecracker,
		}
	}

	dockerExec, err := docker.NewExecutor(ctx, n.fs, "root")
	if err == nil {
		n.executors[string(job_types.ExecutorDocker)] = executorMetadata{
			executor:      dockerExec,
			executionType: job_types.ExecutorDocker,
		}
	}

	return nil
}
