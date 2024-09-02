# telemetry 

- [Project README](https://gitlab.com/nunet/device-management-service/-/blob/main/README.md)
- [Release/Build Status](https://gitlab.com/nunet/device-management-service/-/releases)
- [Changelog](https://gitlab.com/nunet/device-management-service/-/blob/main/CHANGELOG.md)
- [License](https://www.apache.org/licenses/LICENSE-2.0.txt)
- [Contribution Guidelines](https://gitlab.com/nunet/device-management-service/-/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://gitlab.com/nunet/device-management-service/-/blob/main/CODE_OF_CONDUCT.md)
- [Secure Coding Guidelines](https://gitlab.com/nunet/team-processes-and-guidelines/-/blob/main/secure_coding_guidelines/README.md)

## Table of Contents

1. [Description](#1-description)
2. [Structure and Organisation](#2-structure-and-organisation)
3. [Class Diagram](#3-class-diagram)
4. [Functionality](#4-functionality)
5. [Data Types](#5-data-types)
6. [Testing](#6-testing)
7. [Proposed Functionality/Requirements](#7-proposed-functionality--requirements)
8. [References](#8-references)

## Specification

### 1. Description

The Telemetry package is designed to handle and manage telemetry data collection within the Device Management Service (DMS). It supports a variety of observables and collectors to provide a flexible and extensible telemetry system. This package is built to cater to different requirements and separate indices for various packages within DMS.

### 2. Structure and Organisation

Here is quick overview of the contents of this pacakge:

* [README](https://gitlab.com/nunet/device-management-service/-/blob/main/telemetry/README.md): Current file which is aimed towards developers who wish to use and modify the functionality.

* [collector.go](https://gitlab.com/nunet/device-management-service/-/blob/main/telemetry/collector.go): `TBD`

* [collector_factory.go](https://gitlab.com/nunet/device-management-service/-/blob/main/telemetry/collector_factory.go): `TBD`

* [observable.go](https://gitlab.com/nunet/device-management-service/-/blob/main/telemetry/observable.go): `TBD`

* [observable_factory.go](https://gitlab.com/nunet/device-management-service/-/blob/main/telemetry/observable_factory.go): `TBD`

* [event.go](https://gitlab.com/nunet/device-management-service/-/blob/main/telemetry/event.go): `TBD`

* [telemetry.go](https://gitlab.com/nunet/device-management-service/-/blob/main/telemetry/telemetry.go): `TBD`

* [logger](https://gitlab.com/nunet/device-management-service/-/blob/main/telemetry/logger): `TBD`

* [otel](https://gitlab.com/nunet/device-management-service/-/blob/main/telemetry/otel): `TBD`

* [specs](https://gitlab.com/nunet/device-management-service/-/blob/main/telemetry/specs): `TBD`

### 3. Class Diagram

The class diagram for the `telemetry` sub-package is shown below.

#### Source file

[telemetry Class Diagram](https://gitlab.com/nunet/device-management-service/-/blob/main/telemetry/specs/class_diagram.puml)

#### Rendered from source file

```plantuml
!$rootUrlGitlab = "https://gitlab.com/nunet/device-management-service/-/raw/main"
!$packageRelativePath = "/telemetry"
!$packageUrlGitlab = $rootUrlGitlab + $packageRelativePath
 
!include $packageUrlGitlab/specs/class_diagram.puml
```

### 4. Functionality

#### Features

- **Modular Collectors**: Easily configure and extend collectors.
- **Dynamic Observables**: Create and manage observables dynamically.
- **Separate Indices**: Support separate indices for different types of metrics and traces.
- **Configuration Management**: Handle multiple configurations for different parts of the application.

#### Installation

To use the Telemetry package, import it as follows:

```go
import "gitlab.com/nunet/device-management-service/telemetry"
```

#### Configuration

##### TelemetryConfig

The `TelemetryConfig` struct holds the configuration for the telemetry system. Configuration can be loaded from environment variables:

```go
package models

type TelemetryConfig struct {
    ServiceName        string
    GlobalEndpoint     string
    ObservabilityLevel int
    CollectorConfigs   map[string]CollectorConfig
}
```

Example of loading configuration from environment variables:

```go
config, err := models.LoadConfigFromEnv()
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}
```

#### Usage

##### Initializing Telemetry

Initialize the telemetry system with the loaded configuration:

```go
telemetryInstance := telemetry.NewTelemetry(config)
```

#### Creating Collectors

Use the `CollectorFactory` to create and register collectors:

```go
collectorFactory := telemetry.NewCollectorFactory(config)

collectorFactory.RegisterCollector(telemetry.Log, telemetry.NewLogCollector)
collectorFactory.RegisterCollector(telemetry.OpenTelemetry, otel.NewOpenTelemetryCollector)
```

Create and initialize collectors:

```go
logCollector, err := collectorFactory.CreateCollector(telemetry.Log)
if err != nil {
    log.Fatalf("Failed to create Log collector: %v", err)
}
if err := logCollector.Initialize(); err != nil {
    log.Fatalf("Failed to initialize Log collector: %v", err)
}
telemetryInstance.AddCollector("log", logCollector)

otelCollector, err := collectorFactory.CreateCollector(telemetry.OpenTelemetry)
if err != nil {
    log.Fatalf("Failed to create OpenTelemetry collector: %v", err)
}
if err := otelCollector.Initialize(); err != nil {
    log.Fatalf("Failed to initialize OpenTelemetry collector: %v", err)
}
telemetryInstance.AddCollector("otel", otelCollector)
```

#### Creating Observables

Use the `ObservableFactory` to create observables and add them to the telemetry instance:

```go
observableFactory := telemetry.NewObservableFactory()

heartbeatObservable := observableFactory.CreateObservable()
telemetryInstance.AddObservable(telemetry.Heartbeat, "heartbeat_index", heartbeatObservable, []string{"log"})

metricObservable := observableFactory.CreateObservable()
telemetryInstance.AddObservable(telemetry.Metric, "metrics_index", metricObservable, []string{"otel"})
```

#### Handling Events

Create an event and handle it through the telemetry instance:

```go
event := telemetry.Event{
    Type:    telemetry.Heartbeat,
    Payload: map[string]interface{}{"status": "alive"},
    Index:   "heartbeat_index",
}
telemetryInstance.HandleEvent(event)

metricEvent := telemetry.Event{
    Type:    telemetry.Metric,
    Payload: map[string]interface{}{"cpu_usage": 70.5, "memory_usage": 512},
    Index:   "metrics_index",
}
telemetryInstance.HandleEvent(metricEvent)
```

#### Manually Flushing Collectors

You can manually flush the collectors if needed:

```go
if err := telemetryInstance.Flush(); err != nil {
    log.Fatalf("Failed to flush telemetry: %v", err)
}
```

#### Shutting Down

Ensure proper shutdown of the telemetry system to flush and close all collectors:

```go
defer func() {
    if err := telemetryInstance.Shutdown(); err != nil {
        log.Fatalf("Failed to shutdown telemetry: %v", err)
    }
}()
```

#### Extending the Telemetry Package

##### Adding New Collectors

To add a new collector, implement the `Collector` interface:

```go
type Collector interface {
    Initialize() error
    HandleEvent(event Event) error
    Shutdown() error
    Flush() error
    GetObservedLevel() models.ObservabilityLevel
    GetEndpoint() string
}
```

Register the new collector with the `CollectorFactory`:

```go
collectorFactory.RegisterCollector("newCollectorType", NewNewCollector)
```

##### Adding New Observables

To add a new observable, implement the `Observable` interface:

```go
type Observable interface {
    Observe(event Event)
    AddCollector(c Collector)
    GetCollectors() []Collector
}
```

Create the observable using the `ObservableFactory`:

```go
observableFactory := telemetry.NewObservableFactory()
observable := observableFactory.CreateObservable()
```

#### Different Collector Sets and Endpoints

You can configure different sets of collectors for different observables. For example, you may want heartbeat events to be handled by a log collector and metric events to be handled by an OpenTelemetry collector.

To specify different endpoints and collector sets, configure your `TelemetryConfig` with specific collector endpoints:

```go
config, err := models.LoadConfigFromEnv()
if err != nil {
    log.Fatalf("Failed to load configuration: %v", err)
}

// Example configuration:
config.CollectorConfigs = map[string]models.CollectorConfig{
    "OPENTELEMETRY": {
        CollectorType:     "OPENTELEMETRY",
        CollectorEndpoint: "otel-collector.telemetry.nunet.io:4318",
    },
    "LOG": {
        CollectorType:     "LOG",
        CollectorEndpoint: "log-collector.endpoint:1234",
    },
}
```

When adding observables, specify which collectors should handle each type of event:

```go
heartbeatObservable := observableFactory.CreateObservable()
telemetryInstance.AddObservable(telemetry.Heartbeat, "heartbeat_index", heartbeatObservable, []string{"log"})

metricObservable := observableFactory.CreateObservable()
telemetryInstance.AddObservable(telemetry.Metric, "metrics_index", metricObservable, []string{"otel"})
```

#### Why This Design?

This design ensures a modular and flexible approach to telemetry data collection, catering to various requirements and indices across different packages. It provides a robust configuration management system and supports dynamic creation and management of observables and collectors, making it easy to extend and integrate into different parts of the application.

#### Conclusion

The Telemetry package is a powerful tool for managing telemetry data in a device management service. By following the detailed guide provided, developers can easily integrate and extend this package to meet their specific needs, ensuring efficient and effective telemetry data collection and management.

#### Demo Implementation in Main

Below is a demo implementation in `main.go`. This is just for showcasing and will be removed later on:

```go
package main

import (
	"time"

	"github.com/joho/godotenv"
	"gitlab.com/nunet/device-management-service/models"
	"gitlab.com/nunet/device-management-service/telemetry"
	"gitlab.com/nunet/device-management-service/telemetry/logger"
	"gitlab.com/nunet/device-management-service/telemetry/otel"
	"go.uber.org/zap"
)

var log *zap.SugaredLogger

func init() {
	l := logger.New("main")
	log = l.Sugar()
}

func main() {
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	// Load configuration
	config, err := models.LoadConfigFromEnv()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Debug: Log loaded configuration
	log.Infof("Loaded Configuration: %+v", config)

	// Initialize telemetry instance for global endpoint
	globalTelemetry := telemetry.NewTelemetry(config)

	// Create the collector factory for global telemetry
	globalCollectorFactory := telemetry.NewCollectorFactory(config)

	// Register the Log and OpenTelemetry collectors
	globalCollectorFactory.RegisterCollector(telemetry.Log, telemetry.NewLogCollector)
	globalCollectorFactory.RegisterCollector(telemetry.OpenTelemetry, otel.NewOpenTelemetryCollector)

	// Create and add log collector for global telemetry
	globalLogCollector, err := globalCollectorFactory.CreateCollector(telemetry.Log)
	if err != nil {
		log.Fatalf("Failed to create Log collector: %v", err)
	}
	if err := globalLogCollector.Initialize(); err != nil {
		log.Fatalf("Failed to initialize Log collector: %v", err)
	}
	globalTelemetry.AddCollector("log", globalLogCollector)

	// Create and add OpenTelemetry collector for global telemetry
	globalOtelCollector, err := globalCollectorFactory.CreateCollector(telemetry.OpenTelemetry)
	if err != nil {
		log.Fatalf("Failed to create OpenTelemetry collector: %v", err)
	}
	if err := globalOtelCollector.Initialize(); err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry collector: %v", err)
	}
	globalTelemetry.AddCollector("otel", globalOtelCollector)

	// Create a separate config for specific telemetry
	specificConfig := *config
	specificConfig.GlobalEndpoint = config.CollectorConfigs["OPENTELEMETRY"].CollectorEndpoint

	// Initialize telemetry instance for specific endpoint
	specificTelemetry := telemetry.NewTelemetry(&specificConfig)

	// Create the collector factory for specific telemetry
	specificCollectorFactory := telemetry.NewCollectorFactory(&specificConfig)
	specificCollectorFactory.RegisterCollector(telemetry.Log, telemetry.NewLogCollector)
	specificCollectorFactory.RegisterCollector(telemetry.OpenTelemetry, otel.NewOpenTelemetryCollector)

	// Create and add log collector for specific telemetry
	specificLogCollector, err := specificCollectorFactory.CreateCollector(telemetry.Log)
	if err != nil {
		log.Fatalf("Failed to create Log collector: %v", err)
	}
	if err := specificLogCollector.Initialize(); err != nil {
		log.Fatalf("Failed to initialize Log collector: %v", err)
	}
	specificTelemetry.AddCollector("log", specificLogCollector)

	// Create and add OpenTelemetry collector for specific telemetry
	specificOtelCollector, err := specificCollectorFactory.CreateCollector(telemetry.OpenTelemetry)
	if err != nil {
		log.Fatalf("Failed to create OpenTelemetry collector: %v", err)
	}
	if err := specificOtelCollector.Initialize(); err != nil {
		log.Fatalf("Failed to initialize OpenTelemetry collector: %v", err)
	}
	specificTelemetry.AddCollector("otel", specificOtelCollector)

	// Create an observable factory
	observableFactory := telemetry.NewObservableFactory()

	// Create observables and add them to global telemetry with specific collectors
	heartbeatObservableGlobal := observableFactory.CreateObservable()
	globalTelemetry.AddObservable(telemetry.Heartbeat, "heartbeat_global", heartbeatObservableGlobal, []string{"log"})

	metricObservableGlobal := observableFactory.CreateObservable()
	globalTelemetry.AddObservable(telemetry.Metric, "metrics_global", metricObservableGlobal, []string{"otel"})

	// Create observables and add them to specific telemetry with specific collectors
	heartbeatObservableSpecific := observableFactory.CreateObservable()
	specificTelemetry.AddObservable(telemetry.Heartbeat, "heartbeat_specific", heartbeatObservableSpecific, []string{"log"})

	metricObservableSpecific := observableFactory.CreateObservable()
	specificTelemetry.AddObservable(telemetry.Metric, "metrics_specific", metricObservableSpecific, []string{"otel"})

	// Send events to global telemetry
	heartbeatEventGlobal := telemetry.Event{
		Type:    telemetry.Heartbeat,
		Payload: map[string]interface{}{"status": "global"},
		Index:   "heartbeat_global",
	}
	globalTelemetry.HandleEvent(heartbeatEventGlobal)

	metricEventGlobal := telemetry.Event{
		Type:    telemetry.Metric,
		Payload: map[string]interface{}{"cpu_usage": 55.5, "memory_usage": 1024},
		Index:   "metrics_global",
	}
	globalTelemetry.HandleEvent(metricEventGlobal)

	// Send events to specific telemetry
	heartbeatEventSpecific := telemetry.Event{
		Type:    telemetry.Heartbeat,
		Payload: map[string]interface{}{"status": "specific"},
		Index:   "heartbeat_specific",
	}
	specificTelemetry.HandleEvent(heartbeatEventSpecific)

	metricEventSpecific := telemetry.Event{
		Type:    telemetry.Metric,
		Payload: map[string]interface{}{"cpu_usage": 75.3, "memory_usage": 2048},
		Index:   "metrics_specific",
	}
	specificTelemetry.HandleEvent(metricEventSpecific)

	// Manually flush telemetry instances
	if err := globalTelemetry.Flush(); err != nil {
		log.Fatalf("Failed to flush global telemetry: %v", err)
	}
	if err := specificTelemetry.Flush(); err != nil {
		log.Fatalf("Failed to flush specific telemetry: %v", err)
	}

	// Allow some time for events to be sent before shutting down
	time.Sleep(5 * time.Second)

	// Handle shutdown
	defer func() {
		if err := globalTelemetry.Shutdown(); err != nil {
			log.Fatalf("Failed to shutdown global telemetry: %v", err)
		}
		if err := specificTelemetry.Shutdown(); err != nil {
			log.Fatalf("Failed to shutdown specific telemetry: %v", err)
		}
	}()
}
```

### 5. Data Types

`TBD`

### 6. Testing
`TBD`

### 7. Proposed Functionality / Requirements 

#### List of issues

All issues that are related to the implementation of `telemetry` package can be found below. These include any proposals for modifications to the package or new data structures needed to cover the requirements of other packages.

- [telemetry package implementation]() `TBD`


### 8. References