package executor

import (
	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/executor/firecracker"
)

// Assert that Docker Executor implements the Executor interface.
var _ Executor = (*docker.Executor)(nil)

// Assert that Firecracker Executor implements the Executor interface.
var _ Executor = (*firecracker.Executor)(nil)
