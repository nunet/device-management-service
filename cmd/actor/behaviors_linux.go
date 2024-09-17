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
