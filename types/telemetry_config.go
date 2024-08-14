package types

import (
	"log"
	"os"
)

type CollectorConfig struct {
	CollectorType     string
	CollectorEndpoint string
}

type TelemetryConfig struct {
	ServiceName        string
	GlobalEndpoint     string
	ObservabilityLevel int
	CollectorConfigs   map[string]CollectorConfig
}

func LoadConfigFromEnv() (*TelemetryConfig, error) {
	levelStr := os.Getenv("OBSERVABILITY_LEVEL")
	level := parseObservabilityLevel(levelStr)

	// Assume the format for collector-specific configs is like COLLECTOR_<TYPE>_ENDPOINT
	collectorConfigs := make(map[string]CollectorConfig)
	for _, collectorType := range []string{"OPENTELEMETRY", "LOG"} {
		endpoint := os.Getenv("COLLECTOR_" + collectorType + "_ENDPOINT")
		if endpoint != "" {
			collectorConfigs[collectorType] = CollectorConfig{
				CollectorType:     collectorType,
				CollectorEndpoint: endpoint,
			}
		}
	}

	config := &TelemetryConfig{
		ServiceName:        os.Getenv("SERVICE_NAME"),
		GlobalEndpoint:     os.Getenv("COLLECTOR_ENDPOINT"),
		ObservabilityLevel: level,
		CollectorConfigs:   collectorConfigs,
	}

	// Debug: Log loaded environment variables
	log.Printf("Loaded environment variables: SERVICE_NAME=%s, COLLECTOR_ENDPOINT=%s, OBSERVABILITY_LEVEL=%s", config.ServiceName, config.GlobalEndpoint, levelStr)

	return config, nil
}

func parseObservabilityLevel(levelStr string) int {
	switch levelStr {
	case "TRACE":
		return int(TRACE)
	case "DEBUG":
		return int(DEBUG)
	case "INFO":
		return int(INFO)
	case "WARN":
		return int(WARN)
	case "ERROR":
		return int(ERROR)
	case "FATAL":
		return int(FATAL)
	default:
		log.Printf("Invalid OBSERVABILITY_LEVEL: %s, defaulting to INFO", levelStr)
		return int(INFO)
	}
}

// ObservabilityLevel defines levels of observability.
type ObservabilityLevel int

// Constants representing levels of observability.
const (
	TRACE ObservabilityLevel = 1
	DEBUG ObservabilityLevel = 2
	INFO  ObservabilityLevel = 3
	WARN  ObservabilityLevel = 4
	ERROR ObservabilityLevel = 5
	FATAL ObservabilityLevel = 6
)
