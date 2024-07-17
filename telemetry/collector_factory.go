package telemetry

import (
	"fmt"

	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
)

var logCollectorFactory = logger.OtelZapLogger("collector_factory")

type CollectorType string

const (
	Log           CollectorType = "log"
	OpenTelemetry CollectorType = "opentelemetry"
)

type CollectorFactory struct {
	config     *models.TelemetryConfig
	collectors map[CollectorType]func(config *models.TelemetryConfig) (Collector, error)
}

func NewCollectorFactory(config *models.TelemetryConfig) *CollectorFactory {
	return &CollectorFactory{
		config:     config,
		collectors: make(map[CollectorType]func(config *models.TelemetryConfig) (Collector, error)),
	}
}

func (f *CollectorFactory) RegisterCollector(t CollectorType, creator func(config *models.TelemetryConfig) (Collector, error)) {
	f.collectors[t] = creator
}

func (f *CollectorFactory) CreateCollector(t CollectorType) (Collector, error) {
	if creator, ok := f.collectors[t]; ok {
		collector, err := creator(f.config)
		if err != nil {
			logCollectorFactory.Sugar().Errorw("Error creating collector", "type", t, "error", err)
			return nil, err
		}
		return collector, nil
	}
	return nil, fmt.Errorf("unsupported collector type: %s", t)
}
