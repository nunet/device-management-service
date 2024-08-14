package otel

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/telemetry"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

var log = logger.OtelZapLogger("otel")

type OpenTelemetryCollector struct {
	TracerProvider *sdktrace.TracerProvider
	OtEndpoint     string
}

type CollectorImpl struct {
	OpenTelemetryCollector
	config *types.TelemetryConfig
}

func NewOpenTelemetryCollector(config *types.TelemetryConfig) (telemetry.Collector, error) {
	// Determine the endpoint for this collector
	endpoint := config.GlobalEndpoint
	if collectorConfig, exists := config.CollectorConfigs["OPENTELEMETRY"]; exists && collectorConfig.CollectorEndpoint != "" {
		endpoint = collectorConfig.CollectorEndpoint
	}

	return &CollectorImpl{
		OpenTelemetryCollector: OpenTelemetryCollector{
			OtEndpoint: endpoint,
		},
		config: config,
	}, nil
}

func (c *CollectorImpl) Initialize() error {
	ctx := context.Background()

	log.Sugar().Infow("Initializing OTLP HTTP exporter", "endpoint", c.OtEndpoint)
	exp, err := otlptracehttp.New(ctx, otlptracehttp.WithEndpoint(c.OtEndpoint), otlptracehttp.WithInsecure())
	if err != nil {
		log.Sugar().Errorw("Failed to create HTTP trace exporter", "error", err)
		return err
	}
	log.Sugar().Infow("HTTP trace exporter created successfully")

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(c.config.ServiceName),
		),
	)
	if err != nil {
		log.Sugar().Errorw("Failed to create resource", "error", err)
		return err
	}
	log.Sugar().Infow("Resource created successfully")

	c.TracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(c.TracerProvider)
	log.Sugar().Infow("TracerProvider set successfully")

	return nil
}

func (c *CollectorImpl) HandleEvent(event telemetry.Event) error {
	if c.TracerProvider == nil {
		log.Sugar().Errorw("TracerProvider is nil in HandleEvent")
		return fmt.Errorf("TracerProvider is nil")
	}

	ctx := context.Background()
	tr := otel.Tracer("http-tracer")
	ctx, span := tr.Start(ctx, "HandleEvent")
	defer span.End()

	span.SetAttributes(
		attribute.String("event.type", fmt.Sprintf("%d", event.Type)),
		attribute.String("event.payload", fmt.Sprintf("%v", event.Payload)),
		attribute.String("event.index", event.Index),
	)

	log.Sugar().Infow("Handling event", "payload", event.Payload)
	return nil
}

func (c *CollectorImpl) Shutdown() error {
	if c.TracerProvider == nil {
		log.Sugar().Warnw("TracerProvider is nil, skipping shutdown")
		return nil
	}
	ctx := context.Background()
	if err := c.TracerProvider.Shutdown(ctx); err != nil {
		log.Sugar().Errorw("Error shutting down tracer provider", "error", err)
		return err
	}
	log.Sugar().Infow("Collector shutdown successfully")
	return nil
}

func (c *CollectorImpl) Flush() error {
	if c.TracerProvider == nil {
		log.Sugar().Warnw("TracerProvider is nil, skipping flush")
		return nil
	}
	ctx := context.Background()
	if err := c.TracerProvider.ForceFlush(ctx); err != nil {
		log.Sugar().Errorw("Error flushing tracer provider", "error", err)
		return err
	}
	log.Sugar().Infow("Collector flushed successfully")
	return nil
}

func (c *CollectorImpl) GetObservedLevel() types.ObservabilityLevel {
	return types.ObservabilityLevel(c.config.ObservabilityLevel)
}

func (c *CollectorImpl) GetEndpoint() string {
	return c.OtEndpoint
}
