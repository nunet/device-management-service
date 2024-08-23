package telemetry

import (
	"gitlab.com/nunet/device-management-service/telemetry/logger"
	"gitlab.com/nunet/device-management-service/types"
)

var logTelemetry = logger.OtelZapLogger("telemetry")

type Flusher interface {
	Flush() error
}

type Telemetry struct {
	config      *types.TelemetryConfig
	observables map[EventType]map[string][]Observable
	collectors  map[string]Collector
}

func NewTelemetry(config *types.TelemetryConfig) *Telemetry {
	return &Telemetry{
		config:      config,
		observables: make(map[EventType]map[string][]Observable),
		collectors:  make(map[string]Collector),
	}
}

func (t *Telemetry) AddCollector(name string, c Collector) {
	t.collectors[name] = c
}

func (t *Telemetry) AddObservable(eventType EventType, index string, o Observable, collectorNames []string) {
	if _, exists := t.observables[eventType]; !exists {
		t.observables[eventType] = make(map[string][]Observable)
	}
	t.observables[eventType][index] = append(t.observables[eventType][index], o)
	for _, name := range collectorNames {
		if collector, exists := t.collectors[name]; exists {
			o.AddCollector(collector)
		} else {
			logTelemetry.Sugar().Warnw("Collector not found, skipping", "name", name)
		}
	}
}

func (t *Telemetry) HandleEvent(e Event) {
	if observablesByIndex, ok := t.observables[e.Type]; ok {
		if observables, exists := observablesByIndex[e.Index]; exists {
			for _, o := range observables {
				o.Observe(e)
			}
		}
	}
}

func (t *Telemetry) Flush() error {
	for name, collector := range t.collectors {
		if flusher, ok := collector.(Flusher); ok {
			if err := flusher.Flush(); err != nil {
				logTelemetry.Sugar().Errorw("Error flushing collector", "name", name, "error", err)
				return err
			}
		}
	}
	return nil
}

func (t *Telemetry) Shutdown() error {
	if err := t.Flush(); err != nil {
		return err
	}
	for name, collector := range t.collectors {
		if err := collector.Shutdown(); err != nil {
			logTelemetry.Sugar().Errorw("Error shutting down collector", "name", name, "error", err)
			return err
		}
	}
	return nil
}
