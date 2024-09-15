//go:build darwin
// +build darwin

package actor

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/types"
)

func newCustomVMStartRequest(opts *vmStartOpts) (node.CustomVMStartRequest, error) {
	return node.CustomVMStartRequest{}, fmt.Errorf("VMs not supported in this system")
}

type vmStartOpts struct {
	Engine struct {
		KernelImage, RootFileSystem, Initrd, KernelArgs string
	}
	Resources types.Resources
}
