// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux
// +build linux

package jobs

// import (
// 	"context"
// 	"fmt"

// 	"gitlab.com/nunet/device-management-service/executor/docker"
// 	"gitlab.com/nunet/device-management-service/executor/firecracker"
// 	"gitlab.com/nunet/device-management-service/types"
// )

// func (a *Allocation) createExecutor(ctx context.Context, conf types.SpecConfig) error {
// 	if conf.Type == string(types.ExecutorTypeFirecracker) {
// 		executor, err := firecracker.NewExecutor(ctx, a.executionID)
// 		if err != nil {
// 			return fmt.Errorf("firecracker executor: %w", err)
// 		}
// 		a.executor = executor
// 	} else if conf.Type == string(types.ExecutorTypeDocker) {
// 		executor, err := docker.NewExecutor(ctx, a.executionID)
// 		if err != nil {
// 			return fmt.Errorf("docker executor: %w", err)
// 		}
// 		a.executor = executor
// 	}

// 	return nil
// }
