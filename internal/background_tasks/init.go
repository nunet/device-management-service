package backgroundtasks

import (
	"github.com/uptrace/opentelemetry-go-extra/otelzap"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
)

var zlog *otelzap.Logger

func init() {
	zlog = logger.OtelZapLogger("background_tasks")
}
