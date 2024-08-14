package telemetry

import (
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
)

var logCollector = logger.OtelZapLogger("collector")

type Collector interface {
	Initialize() error
	HandleEvent(event Event) error
	Shutdown() error
	GetObservedLevel() types.ObservabilityLevel
	GetEndpoint() string
}

type LogCollector struct{}

func NewLogCollector(config *types.TelemetryConfig) (Collector, error) {
	// No specific configuration needed for LogCollector currently.
	return &LogCollector{}, nil
}

func (c *LogCollector) Initialize() error {
	logCollector.Sugar().Infow("LogCollector initialized.")
	return nil
}

func (c *LogCollector) HandleEvent(event Event) error {
	logCollector.Sugar().Infow("LogCollector received event", "event", event)
	return nil
}

func (c *LogCollector) Shutdown() error {
	logCollector.Sugar().Infow("LogCollector shutdown.")
	return nil
}

func (c *LogCollector) Flush() error {
	logCollector.Sugar().Infow("LogCollector flushed.")
	return nil
}

func (c *LogCollector) GetObservedLevel() types.ObservabilityLevel {
	return types.INFO
}

func (c *LogCollector) GetEndpoint() string {
	return "localhost"
}
