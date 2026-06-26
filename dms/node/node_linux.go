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
	containerdexecutor "gitlab.com/nunet/device-management-service/executor/containerd"
	"gitlab.com/nunet/device-management-service/executor/docker"
)

func (n *Node) initSupportedExecutors(ctx context.Context) error {
	dockerExec, err := docker.NewExecutor(ctx, n.fs, "root")
	if err == nil {
		n.executors[string(job_types.ExecutorDocker)] = executorMetadata{
			executor:      dockerExec,
			executionType: job_types.ExecutorDocker,
		}
	}

	containerdExec, err := containerdexecutor.NewExecutor(ctx, "root", n.dmsConfig.Job.Containerd)
	if err == nil {
		n.executors[string(job_types.ExecutorContainerd)] = executorMetadata{
			executor:      containerdExec,
			executionType: job_types.ExecutorContainerd,
		}
	}

	return nil
}
