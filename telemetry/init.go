// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package telemetry

import (
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/types"
)

var (
	once      sync.Once
	instance  *Telemetry
	logLevel  types.ObservabilityLevel
	zapLogger *zap.Logger
)

// InitGlobalTelemetry initializes the global telemetry instance with configuration loaded from the configuration package.
func InitGlobalTelemetry() error {
	var initError error
	once.Do(func() {
		// Initialize Zap logger
		zapLogger, initError = initZapLogger()
		if initError != nil {
			panic(initError)
		}
		zap.ReplaceGlobals(zapLogger)

		cfg := config.GetConfig()
		telemetryConfig := cfg.Telemetry

		logLevel = types.INFO // Default level
		if level, err := types.ParseObservabilityLevel(telemetryConfig.ObservabilityLevel); err == nil {
			logLevel = level
		} else {
			zap.L().Warn("Invalid observability level, defaulting to INFO", zap.Error(err))
		}

		instance = &Telemetry{
			config: &types.TelemetryConfig{
				ServiceName:        telemetryConfig.ServiceName,
				GlobalEndpoint:     telemetryConfig.GlobalEndpoint,
				ObservabilityLevel: telemetryConfig.ObservabilityLevel, // Assign the string value
				TelemetryMode:      telemetryConfig.TelemetryMode,
			},
		}

		opentelemetryCollector := NewOpenTelemetryCollector(instance.config, zap.L())
		logCollector := NewLogCollector(instance.config, zap.L())
		instance.collectors = map[string]Collector{
			logCollector.GetName():           logCollector,
			opentelemetryCollector.GetName(): opentelemetryCollector,
		}
		for _, collector := range instance.collectors {
			if err := collector.Initialize(); err != nil {
				zap.L().Error("Failed to initialize collector", zap.Error(err))
			}
		}

		// Start periodic flush after initializing collectors
		StartPeriodicFlush(5 * time.Minute)
	})
	return initError
}

// initZapLogger initializes the zap logger based on configuration or environment variables.
func initZapLogger() (*zap.Logger, error) {
	var err error
	var logger *zap.Logger

	if _, debug := os.LookupEnv("NUNET_DEBUG"); debug || config.GetConfig().General.Debug {
		zapConfig := zap.NewDevelopmentConfig()
		zapConfig.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		logger, err = zapConfig.Build()
	} else {
		logger, err = zap.NewProduction()
	}

	return logger, err
}

// StartPeriodicFlush starts a goroutine that periodically flushes telemetry data.
func StartPeriodicFlush(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			zap.L().Info("Periodic flush started for telemetry")
			instance.Flush()
		}
	}()
}
