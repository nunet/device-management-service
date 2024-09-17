package executor

import (
	"gitlab.com/nunet/device-management-service/executor/docker"
)

// Assert that Docker Executor implements the Executor interface.
var _ Executor = (*docker.Executor)(nil)
