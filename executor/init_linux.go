//go:build linux
// +build linux

package executor

import (
	"gitlab.com/nunet/device-management-service/executor/firecracker"
)

// Assert that Docker Executor implements the Executor interface.
var _ Executor = (*firecracker.Executor)(nil)
