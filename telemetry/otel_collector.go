package telemetry

import (
	"context"

	"gitlab.com/nunet/device-management-service/types"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.uber.org/zap"
)

type OpenTelemetryCollector struct {
	config         *types.TelemetryConfig
	logger         *zap.Logger
	tracerProvider *sdktrace.TracerProvider
}

func NewOpenTelemetryCollector(config *types.TelemetryConfig, logger *zap.Logger) *OpenTelemetryCollector {
	return &OpenTelemetryCollector{
		config: config,
		logger: logger,
	}
}

func (c *OpenTelemetryCollector) Initialize() error {
	c.logger.Info("Initializing OpenTelemetry HTTP trace exporter",
		zap.String("endpoint", c.config.GlobalEndpoint),
	)

	exp, err := otlptracehttp.New(context.Background(),
		otlptracehttp.WithEndpoint(c.config.GlobalEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		c.logger.Error("Failed to create HTTP trace exporter", zap.Error(err))
		return err
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(c.config.ServiceName),
		),
	)
	if err != nil {
		c.logger.Error("Failed to create resource", zap.Error(err))
		return err
	}

	c.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(c.tracerProvider)

	c.logger.Info("OpenTelemetryCollector initialized.")
	return nil
}

func (c *OpenTelemetryCollector) HandleEvent(event types.Event) error {
	fields := []attribute.KeyValue{
		attribute.String("message", event.Message),
		attribute.String("level", event.Level.String()),
	}

	for key, value := range event.Payload {
		fields = append(fields, attribute.String(key, value.(string)))
	}

	c.logger.Info("Handling event",
		zap.String("message", event.Message),
		zap.String("level", event.Level.String()),
		zap.Any("context", event.Context),
		zap.Any("payload", event.Payload),
	)

	// Fetch tracer name from context, or default to "otel-tracer"
	tracerName, ok := event.Context.Value(tracerNameKey).(string)
	if !ok {
		tracerName = "otel-tracer"
	}

	tracer := c.tracerProvider.Tracer(tracerName)
	ctx := context.Background()
	_, span := tracer.Start(ctx, event.Message)
	span.SetAttributes(fields...)
	span.End()

	c.logger.Info("Event sent to OpenTelemetry",
		zap.String("message", event.Message),
		zap.String("level", event.Level.String()),
		zap.Any("context", event.Context),
		zap.Any("payload", event.Payload),
	)

	return nil
}

func (c *OpenTelemetryCollector) Flush() error {
	if c.tracerProvider == nil {
		c.logger.Warn("TracerProvider is nil, skipping flush")
		return nil
	}
	c.logger.Info("Flushing tracer provider")
	if err := c.tracerProvider.ForceFlush(context.Background()); err != nil {
		c.logger.Error("Error flushing tracer provider", zap.Error(err))
		return err
	}
	c.logger.Info("Collector flushed successfully")
	return nil
}

func (c *OpenTelemetryCollector) Shutdown() error {
	if c.tracerProvider == nil {
		c.logger.Warn("TracerProvider is nil, skipping shutdown")
		return nil
	}
	c.logger.Info("Shutting down tracer provider")
	if err := c.tracerProvider.Shutdown(context.Background()); err != nil {
		c.logger.Error("Error shutting down tracer provider", zap.Error(err))
		return err
	}
	c.logger.Info("Collector shutdown successfully")
	return nil
}

func (c *OpenTelemetryCollector) GetName() string {
	return "opentelemetry"
}

func (c *OpenTelemetryCollector) SpanContext(ctx context.Context, span string) (context.Context, context.CancelFunc) {
	tracerName, ok := ctx.Value(tracerNameKey).(string)
	if !ok {
		tracerName = c.GetName()
	}
	tracer := c.tracerProvider.Tracer(tracerName)
	ctx, s := tracer.Start(ctx, span)
	cancel := func() {
		s.End()
	}
	return ctx, cancel
}

// Compile-time check to ensure OpenTelemetryCollector implements the Collector interface
var _ Collector = (*OpenTelemetryCollector)(nil)
