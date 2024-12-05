// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux && (arm || arm64)

//nolint:all // This is a stub file for arm linux
package gpu

import "gitlab.com/nunet/device-management-service/types"

// GetGPUs returns the GPUs in the system
func GetGPUs() ([]types.GPU, error) {
	// TODO: Implement this function
	return nil, nil
}

// GetGPUUsage returns the GPU usage for the GPUs with the specified UUIDs. If no UUIDs are provided, it returns the usage of all the GPUs
func GetGPUUsage(uuid ...string) ([]types.GPU, error) {
	// TODO: Implement this function
	return nil, nil
}
