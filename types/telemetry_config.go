package types

import (
	"context"
)

// TelemetryConfig holds the configuration for the telemetry system.
type TelemetryConfig struct {
	ServiceName        string
	GlobalEndpoint     string
	ObservabilityLevel string
	CollectorConfigs   map[string]CollectorConfig
	TelemetryMode      string
}

// CollectorConfig holds the configuration for individual collectors.
type CollectorConfig struct {
	CollectorType     string
	CollectorEndpoint string
}

// Event represents a telemetry event with its details.
type Event struct {
	Context context.Context
	Level   ObservabilityLevel
	Message string
	Payload map[string]interface{}
}

// ObservabilityLevel defines the levels of observability.
type ObservabilityLevel int

const (
	TRACE ObservabilityLevel = 1
	DEBUG ObservabilityLevel = 2
	INFO  ObservabilityLevel = 3
	WARN  ObservabilityLevel = 4
	ERROR ObservabilityLevel = 5
	FATAL ObservabilityLevel = 6
)

// ParseObservabilityLevel converts a string representation of the observability level to an integer.
func ParseObservabilityLevel(levelStr string) (ObservabilityLevel, error) {
	switch levelStr {
	case "TRACE":
		return TRACE, nil
	case "DEBUG":
		return DEBUG, nil
	case "INFO":
		return INFO, nil
	case "WARN":
		return WARN, nil
	case "ERROR":
		return ERROR, nil
	case "FATAL":
		return FATAL, nil
	default:
		return INFO, nil
	}
}

func (level ObservabilityLevel) String() string {
	switch level {
	case TRACE:
		return "TRACE"
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}
