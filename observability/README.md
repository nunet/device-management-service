# Observability Module

## tracing.go

The `tracing.go` file is part of an observability framework for the DMS, integrating Elastic APM (Application Performance Monitoring) to track and monitor system performance and trace the flow of operations. Here’s an explanation of the key components and functionalities:

* **Elastic APM Setup**: Initialize, manage, and shut down the Elastic APM tracer for monitoring and tracing operations in the system.
* **System Metrics**: The code collects key system metrics (CPU, memory, disk, etc.) and sends them as custom metrics to the APM server.
* **Tracing Operations**: The `StartTrace` function enables the tracing of operations with a name, logging the start and end of operations along with detailed performance metrics (e.g., execution time).
* **Transaction and Span Management**: Transactions and spans are created to trace and measure specific operations in the service, such as HTTP requests or background tasks.
* **No-Op Mode**: If tracing is disabled (in "no-op" mode), no data is sent to the APM server.

## observability.go

The `observability.go` file contains various functions and configurations for setting up observability in the application, including logging, tracing, event handling, and Elasticsearch logging integration. Below is a breakdown of the key elements and explanations of how they work together:

### Core Components

* **Logger**: The logging system is configured with multiple outputs (console, file, and Elasticsearch). It uses zap with different zapcore components to handle logging.
* **Elasticsearch Syncer**: Buffered writing to Elasticsearch is implemented using a custom syncer, which also includes a short “preflight” check to gracefully disable ES logging if credentials or connectivity fail.
* **APM Tracing**: Integration with Elastic APM allows for tracing of application events.
* **Event Bus**: Facilitates the emission and listening of custom events.
* **DID**: Provides a unique identifier for logs and events, helping correlate them across services.

### Observability Workflow

* When the system starts, `Initialize()` sets up the logger, event bus, and APM tracing based on the configuration.
* The logger is initialized with a set of cores, including console, file, and potentially Elasticsearch logging.
  * A short preflight check is performed for Elasticsearch to ensure connectivity and valid credentials, helping avoid indefinite blocking.
  * If Elasticsearch fails (e.g., invalid key, unreachable), logging gracefully falls back to console and file logs.
* Tracing is activated if an APM server URL is provided, and traces are tied to the application’s operations.
* Event bus integration allows for custom events to be emitted and handled asynchronously.
* The system metrics (CPU, memory, disk, etc.) are collected and, if tracing is enabled, sent to the APM server. Otherwise, they’re skipped in no-op mode.
