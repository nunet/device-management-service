package telemetry

import (
	"context"

	"gitlab.com/nunet/device-management-service/types"
	"go.uber.org/zap"
)

type LogCollector struct {
	config *types.TelemetryConfig
	logger *zap.Logger
}

func NewLogCollector(config *types.TelemetryConfig, logger *zap.Logger) *LogCollector {
	return &LogCollector{
		config: config,
		logger: logger,
	}
}

func (c *LogCollector) Initialize() error {
	c.logger.Info("LogCollector initialized.")
	return nil
}

func (c *LogCollector) HandleEvent(event types.Event) error {
	fields := []zap.Field{
		zap.Any("context", event.Context),
		zap.String("message", event.Message),
		zap.String("level", event.Level.String()),
		zap.Any("payload", event.Payload),
	}

	switch event.Level {
	case types.TRACE:
		c.logger.Debug(event.Message, fields...)
	case types.DEBUG:
		c.logger.Debug(event.Message, fields...)
	case types.INFO:
		c.logger.Info(event.Message, fields...)
	case types.WARN:
		c.logger.Warn(event.Message, fields...)
	case types.ERROR:
		c.logger.Error(event.Message, fields...)
	case types.FATAL:
		c.logger.Fatal(event.Message, fields...)
	default:
		c.logger.Info(event.Message, fields...)
	}

	return nil
}

func (c *LogCollector) Flush() error {
	if err := c.logger.Sync(); err != nil { // Check for error in Sync
		return err
	}
	return nil
}

func (c *LogCollector) Shutdown() error {
	return c.Flush()
}

func (c *LogCollector) GetName() string {
	return "log"
}

func (c *LogCollector) SpanContext(ctx context.Context, _ string) (context.Context, context.CancelFunc) {
	// LogCollector does not support tracing, so just return the original context and a no-op cancel function
	return ctx, func() {}
}

// Compile-time check to ensure LogCollector implements the Collector interface
var _ Collector = (*LogCollector)(nil)
