//go:build linux
// +build linux

package firecracker

import (
	"gitlab.com/nunet/device-management-service/telemetry/logger"
)

var zlog *logger.Logger

func init() {
	zlog = logger.New("executor.firecracker")
}
