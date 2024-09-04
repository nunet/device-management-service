package telemetry

import (
	"context"
	"runtime"

	"go.uber.org/zap"

	"gitlab.com/nunet/device-management-service/api/docs"

	"gitlab.com/nunet/device-management-service/types"
)

type Telemetry struct {
	config     *types.TelemetryConfig
	collectors map[string]Collector
	testMode   bool
}

// Define a custom type for context keys to avoid conflicts
type contextKey string

const (
	collectorsKey contextKey = "collectors"
	tracerNameKey contextKey = "tracerName"
	versionKey    contextKey = "version"
)

func GetTelemetry() *Telemetry {
	return instance
}

// NewTelemetry initializes a new Telemetry instance.
// If testMode is true, the telemetry operations will be no-ops.
func NewTelemetry(config *types.TelemetryConfig, collectors map[string]Collector, testMode bool) *Telemetry {
	if testMode {
		return &Telemetry{
			testMode: true,
		}
	}
	return &Telemetry{
		config:     config,
		collectors: collectors,
		testMode:   false,
	}
}

func (t *Telemetry) SpanContext(ctx context.Context, tracerName string, span string, collectors ...string) (context.Context, context.CancelFunc) {
	if t.testMode {
		return ctx, func() {}
	}

	var cancelFuncs []context.CancelFunc

	// Fetch caller info
	pc, _, _, ok := runtime.Caller(1)
	functionName := "unknown_function"
	if ok {
		function := runtime.FuncForPC(pc)
		functionName = function.Name()
	}

	// Use caller info as default tracer and span names if not provided
	if tracerName == "" {
		tracerName = functionName
	}
	if span == "" {
		span = functionName
	}

	ctx = context.WithValue(ctx, collectorsKey, collectors)
	ctx = context.WithValue(ctx, tracerNameKey, tracerName)
	var cancelFunc context.CancelFunc
	for _, collector := range collectors {
		if c, ok := t.collectors[collector]; ok {
			ctx, cancelFunc = c.SpanContext(ctx, span)
			cancelFuncs = append(cancelFuncs, cancelFunc)
		}
	}
	cancel := func() {
		for _, cancelFunc := range cancelFuncs {
			cancelFunc()
		}
	}
	return ctx, cancel
}

func (t *Telemetry) Trace(ctx context.Context, message string, payload map[string]interface{}) {
	if t.testMode {
		return
	}
	t.logEvent(ctx, types.TRACE, message, payload)
}

func (t *Telemetry) Debug(ctx context.Context, message string, payload map[string]interface{}) {
	if t.testMode {
		return
	}
	t.logEvent(ctx, types.DEBUG, message, payload)
}

func (t *Telemetry) Info(ctx context.Context, message string, payload map[string]interface{}) {
	if t.testMode {
		return
	}
	t.logEvent(ctx, types.INFO, message, payload)
}

func (t *Telemetry) Warn(ctx context.Context, message string, payload map[string]interface{}) {
	if t.testMode {
		return
	}
	t.logEvent(ctx, types.WARN, message, payload)
}

func (t *Telemetry) Error(ctx context.Context, message string, payload map[string]interface{}) {
	if t.testMode {
		return
	}
	t.logEvent(ctx, types.ERROR, message, payload)
}

func (t *Telemetry) Fatal(ctx context.Context, message string, payload map[string]interface{}) {
	if t.testMode {
		return
	}
	t.logEvent(ctx, types.FATAL, message, payload)
}

func (t *Telemetry) logEvent(ctx context.Context, level types.ObservabilityLevel, message string, payload map[string]interface{}) {
	// Check if telemetry is enabled
	if t.config.TelemetryMode == "disabled" {
		return
	}

	// Only log events that are at or above the configured log level
	if level < logLevel {
		return
	}

	// Add the version to the context
	ctx = context.WithValue(ctx, versionKey, docs.SwaggerInfo.Version)

	event := types.Event{
		Context: ctx,
		Level:   level,
		Message: message,
		Payload: payload,
	}

	// Check for specific collector in context
	collectors, ok := ctx.Value(collectorsKey).([]string)
	if ok {
		for _, collector := range collectors {
			if c, ok := t.collectors[collector]; ok {
				if err := c.HandleEvent(event); err != nil {
					zap.L().Error("Failed to handle event", zap.Error(err))
				}
			}
		}
		return
	}

	// Forward to all collectors by default
	for _, collector := range t.collectors {
		if err := collector.HandleEvent(event); err != nil {
			zap.L().Error("Failed to handle event", zap.Error(err))
		}
	}
}

func (t *Telemetry) Flush() {
	if t.testMode {
		return
	}
	for _, collector := range t.collectors {
		if err := collector.Flush(); err != nil {
			zap.L().Error("Failed to flush collector", zap.Error(err))
		}
	}
}

func (t *Telemetry) Shutdown() {
	if t.testMode {
		return
	}
	t.Flush()
	for _, collector := range t.collectors {
		if err := collector.Shutdown(); err != nil {
			zap.L().Error("Failed to shut down collector", zap.Error(err))
		}
	}
}
