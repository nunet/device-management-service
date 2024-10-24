// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux
// +build linux

package actor

import (
	"github.com/google/uuid"

	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/executor/firecracker"
	"gitlab.com/nunet/device-management-service/types"
)

func newCustomVMStartRequest(opts *vmStartOpts) (node.CustomVMStartRequest, error) {
	engine := firecracker.NewFirecrackerEngineBuilder(opts.Engine.RootFileSystem)
	engine = engine.WithKernelImage(opts.Engine.KernelImage)
	engine = engine.WithKernelArgs(opts.Engine.KernelArgs)
	engine = engine.WithInitrd(opts.Engine.Initrd)
	es := engine.Build()
	req := node.CustomVMStartRequest{
		Execution: types.ExecutionRequest{
			ExecutionID: uuid.New().String(),
			EngineSpec:  es,
			Resources:   &opts.Resources,
		},
	}
	return req, nil
}

type vmStartOpts struct {
	Engine    firecracker.EngineSpec
	Resources types.Resources
}
