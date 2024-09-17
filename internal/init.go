package internal

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
)

var (
	zlog         *otelzap.Logger
	ShutdownChan chan os.Signal
)

func init() {
	zlog = logger.OtelZapLogger("internal")

	ShutdownChan = make(chan os.Signal, 1)
	signal.Notify(ShutdownChan, syscall.SIGINT, syscall.SIGTERM)
}
