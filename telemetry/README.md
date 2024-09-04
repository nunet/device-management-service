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
var st = telemetry.GetTelemetry()

func ExampleFunction(ctx context.Context) {
    // Trace level
    st.Trace(ctx, "Trace message", nil)

    // Debug level
    st.Debug(ctx, "Debug message", map[string]interface{}{"key2": "value2"})

    // Info level
    st.Info(ctx, "Info message", map[string]interface{}{"key3": "value3"})

    // Warn level
    st.Warn(ctx, "Warn message", map[string]interface{}{"key4": "value4"})

    // Error level
    st.Error(ctx, "Error message", map[string]interface{}{"key5": "value5"})

    // Fatal level
    st.Fatal(ctx, "Fatal message", map[string]interface{}{"key6": "value6"})
}
```
the payload part at the end can be nil if not needed

### Context Propagation and Tracing

The telemetry system provides robust context propagation and tracing capabilities, ensuring that important contextual information, such as `DMS version`, `file name`, and `function name`, is captured and included in traces and logs throughout the application.

#### Automatic Context Values
The system automatically includes certain key context values:
- `DMS version`: Automatically added to every context, ensuring that the version of the service is tracked.
- `File name` and `Function name`: Using Go's runtime package, the telemetry system captures the file name and function name of the calling code, providing detailed tracing information without requiring manual input.

#### Custom Context Values
In addition to the automatic context values, you can add custom values to the context, such as `libp2p` information, `uuid`, or any other relevant metadata. This allows for more granular tracing and debugging capabilities.

#### Tracing with `SpanContext`
The `SpanContext` function simplifies the process of adding tracing to your functions. It takes a context, tracer name, span name, and optional collectors, and returns a modified context along with a cancel function. This approach enables you to manage multiple collectors and ensure that all tracing spans are properly closed.

Example usage of `SpanContext`:
```go
func SomeFunction(ctx context.Context) {
    // Use SpanContext to start a new trace span and add tracing information
    ctx, cancel := st.SpanContext(ctx, "exampleTracer", "exampleSpan", "opentelemetry")
    defer cancel() // Ensure the span is closed properly

    // Add custom context values
    ctx = context.WithValue(ctx, "libp2p", "some-libp2p-info")
    ctx = context.WithValue(ctx, "uuid", "some-uuid-value")

    // Pass the context to telemetry logging
    st.Info(ctx, "Operation successful", map[string]interface{}{"operation": "example"})
}
```
With this setup, the telemetry system ensures that all relevant tracing information, including automatic context values and any custom values you add, is included in both the logs and traces, providing a comprehensive view of the application's behavior and performance.

### Configuration

The telemetry system is highly configurable, allowing you to control which events are logged and where they are sent. Configuration is loaded from a configuration file, and the following variables are used:

- **SERVICE_NAME**: The name of the service being monitored.
- **GLOBAL_ENDPOINT**: The endpoint to which telemetry data (e.g., traces) is sent.
- **OBSERVABILITY_LEVEL**: The minimum level of events to log (e.g., INFO, DEBUG).
- **TELEMETRY_MODE**: The mode in which the telemetry system operates (e.g., production, test, disabled).

```json
{
  "telemetry": {
    "service_name": "NunetDMS",
    "global_endpoint": "otel-collector.telemetry.nunet.io:4318",
    "observability_level": "INFO",
    "telemetry_mode": "production"
  }
}
```

### Collectors

Collectors are responsible for handling events and traces captured by the telemetry system. The package comes with two built-in collectors, but custom collectors can be easily added.

#### OpenTelemetry Collector

The OpenTelemetry collector is responsible for sending trace data to an OpenTelemetry endpoint. This allows you to capture detailed performance metrics and traces that can be analyzed using OpenTelemetry-compatible tools.


#### Log Collector

The log collector uses a logger (e.g., Zap) to log events locally. This is useful for scenarios where you want to capture events and errors in log files or other logging systems.


#### Custom Collectors

You can create custom collectors by implementing the `Collector` interface. This allows you to extend the telemetry system to support additional use cases, such as sending events to third-party monitoring tools or storing telemetry data in a custom format.

```go
type CustomCollector struct{}

func (c *CustomCollector) Initialize() error {
    // Custom initialization logic
    return nil
}

func (c *CustomCollector) HandleEvent(event models.Event) error {
    // Custom event handling logic
    return nil
}

func (c *CustomCollector) Flush() error {
    // Custom flush logic
    return nil
}

func (c *CustomCollector) Shutdown() error {
    // Custom shutdown logic
    return nil
}

func (c *CustomCollector) GetName() string {
    return "custom_collector"
}

// Register the custom collector
telemetry.GetTelemetry().RegisterCollector(&CustomCollector{})
```

### Periodic Flush

To ensure that all telemetry data is captured and sent before the application shuts down, the telemetry system supports periodic flushing. This is particularly useful in scenarios where you want to minimize data loss during unexpected shutdowns or crashes.

You can start the periodic flush process by calling `StartPeriodicFlush`, passing the desired flush interval as an argument.

Example periodic flush setup placeholder

### Shutdown

When shutting down the application, it's important to ensure that all telemetry data is flushed and collectors are properly shut down. This can be done by calling the `Shutdown` method on the telemetry system, which will flush any remaining data and gracefully terminate all collectors.

```go
// Start periodic flush of telemetry data every 10 seconds
telemetry.StartPeriodicFlush(10 * time.Second)
```
Note that its already initalized by default in `init.go`

### Additional Features

- **Automatic Error Logging**: Errors encountered during telemetry operations (e.g., failing to send traces) are automatically logged using the log collector.
- **Context Management**: The system automatically manages context propagation, allowing you to trace requests and operations across different parts of your application.
- **Extensibility**: The system is designed to be easily extendable with custom collectors, allowing you to adapt it to your specific monitoring and observability needs.

### Selective Collector Tracing with `SpanContext`

The `SpanContext` function provides fine-grained control over which collectors handle a particular trace. This allows you to specify only the collectors you want to use for a specific operation, offering more flexibility in your telemetry setup.

#### How it Works

The `SpanContext` function takes the following parameters:

- **Context**: The context to propagate.
- **Tracer Name**: The name of the tracer (e.g., the name of the component or function).
- **Span Name**: The name of the span (e.g., the operation being performed).
- **Collectors**: A list of collector names (e.g., "opentelemetry", "log") that should handle this trace.

Only the collectors specified in the `SpanContext` call will handle the trace, allowing you to customize which collectors receive specific tracing data.

Example usage:

```go
ctx, cancel := st.SpanContext(context.Background(), "controller", "volume_controller_init", "opentelemetry", "log")
defer cancel()

// Perform operations, and only the "opentelemetry" and "log" collectors will handle the trace
```

Note that you can pass more than 2  or only pass 1 collector 