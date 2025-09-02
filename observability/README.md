# Observability Package

## Table of Contents

1. [Overview](#overview)  
2. [Features](#features)  
3. [Dependencies](#dependencies)  
4. [Initialization](#initialization)  
5. [Usage](#usage)  
   1. [Logging](#logging)  
   2. [Event Emission](#event-emission)  
   3. [Tracing](#tracing)  
   4. [Label-Based Routing](#label-based-routing)  
6. [Runtime Configuration](#runtime-configuration)  
7. [Testing](#testing)  
8. [Shutdown](#shutdown)  

---

## Overview

The **Observability** package provides a comprehensive solution for monitoring and troubleshooting DMS. It includes:

- Configurable, structured logging with outputs to the console, files, and optionally Elasticsearch.
- Custom event emission using a LibP2P event bus.
- Distributed tracing with Elastic APM integration.
- Label-based routing for log messages to control log destinations.
- Dynamic runtime configuration and a no-op mode for testing.

---

## Features

1. **Logging**  
   - Console logging with colorized output for ease of development.
   - File-based logging using a rolling file writer (via lumberjack).
   - Optional Elasticsearch logging for centralized log management.
   - Dynamic log level configuration (DEBUG, INFO, WARN, ERROR, etc.).

2. **Custom Event Emission**  
   - Leverages LibP2P's `EventBus` to emit application events.
   - Events adhere to the `CustomEvent` schema for downstream processing.

3. **Distributed Tracing**  
   - Integrated with Elastic APM for distributed tracing.
   - Automatic system metrics collection (CPU, memory, disk, network).
   - Custom instrumentation via the `StartTrace` helper.

4. **Label-Based Routing**  
   - Attach labels to log messages to determine routing decisions.
   - Override default Elasticsearch index or skip ES logging based on labels.
   - Predefined labels include: `default`, `accounting`, `metric`, `deployment`, `allocation`, `node`, and `user`.

  **Internal Architecture and Event Bus Explanation:**

  We Do Actually Have an “Internal Event Bus.”

  The “internal event bus” is created via Libp2p’s `host.EventBus()` inside `observability.initEventBus(...)`. We attach a custom emitter to that bus (`customEventEmitter`), and the code in `newEventEmitterCore(...)` in `observability.go` will emit every log entry as a `CustomEvent` to that bus.

  **Why You Don’t See the Word “Collector.”**

  In an older design we talked about “collectors” as separate pipelines for sending logs to console, file, Elasticsearch, etc. In the current code, that same concept is implemented using multiple Zap cores in a “tee.” Each core (console, file, Elasticsearch, event bus) is effectively a “collector,” but we just don’t label them that way in the code anymore. You can see this in `observability.initLogger(...)`, which calls `createConsoleCore(...)`, `createFileCore(...)`, `createElasticsearchCore(...)`, and `newEventEmitterCore(...)`. Then, all those cores are combined via `zapcore.NewTee(cores...)`.

  **Where Is the “Internal Event Bus” Actually Implemented?**

  It is initialized in `initEventBus` in `observability.go`. We then wrap it with a custom Zap Core in the `newEventEmitterCore` function, which creates the `eventEmitterCore`. Inside that core’s `Write(...)` method, the log data is packaged into a `CustomEvent` struct and emitted via `customEventEmitter.Emit(customEvent)`. This is exactly what ships logs onto the Libp2p event bus.

  **Summary:**

  - **Zap + Multiple Cores** = The old notion of multiple “collectors” (console, file, Elasticsearch).
  - **Event Emitter Core** = The “internal event bus” hook, which you can subscribe to for advanced use cases (like a future reputation system).
  - **Default Label** = The fallback if a caller doesn’t specify any label at all.

  Everything happens in the observability package, but we rely on the Libp2p host’s event bus. No extra “collector” objects are defined by name—the pattern is just a combination of Zap “tee’d cores” plus the event bus emitter.

5. **Runtime Configuration**  
   - Dynamically enable/disable Elasticsearch logging.
   - Update Elasticsearch endpoint, APM server URL, API key, and flush interval at runtime.
   - No-op mode to disable all observability features during testing.

---

## Dependencies

- **LibP2P** ([go-libp2p](https://github.com/libp2p/go-libp2p))  
  Provides the `host.Host` for event bus integration.

- **Zap** ([go.uber.org/zap](https://github.com/uber-go/zap))  
  The primary structured logging library.

- **Lumberjack** ([natefinch/lumberjack](https://github.com/natefinch/lumberjack))  
  Manages rolling log files for file-based logging.

- **Elasticsearch Client** ([olivere/elastic](https://github.com/olivere/elastic))  
  Used to index logs into Elasticsearch.

- **Elastic APM** ([go.elastic.co/apm](https://github.com/elastic/apm-agent-go))  
  Enables distributed tracing and custom metric collection.

---

## Initialization

Before using any observability features, we initialize the package. This is done in the network package where the LibP2P host is set up.

initialization snippet from `network\libp2p\libp2p.go`:

 ```go
	// Initialize the observability package with the host and DID
	if err := observability.Initialize(l.Host, didInstance, cfg); err != nil {
		return fmt.Errorf("failed to initialize observability: %w", err)
	}
 ```

This setup:
- Configures logging (console, file, and optionally Elasticsearch).
- Sets up the event emitter on the LibP2P `EventBus`.
- Initializes Elastic APM tracing if `cfg.APM.ServerURL` is provided.

---

## Usage

### Logging

The package uses [Zap](https://github.com/uber-go/zap) for structured logging. After initialization, you can produce logs as follows:

```go
// Example 1: Log only a plain message (no extra fields) goes to default lable
log.Info("System started successfully")

// Example 2: Log a message with additional string parameters (no labels goes to default lable) 
log.Infow("Operation completed",
	"executorID", e.ID,
	"status", "success",
)

// Example 3: Log an info message with labels
log.Infow("docker_executor_cleanup_success",
	"labels", string(observability.LabelDeployment),
	"executorID", e.ID,
)

```
#### Dynamic Log Level Adjustment

Log levels can be updated at runtime:
```go
    err := observability.SetLogLevel("debug")
    if err != nil {
        // handle error
    }
```

### Tracing

Elastic APM is integrated for distributed tracing. Use the helper function `StartTrace` to begin new transactions or spans.

Usage examples:
```go
    // Starting a trace without an existing context.
    func processOrder(orderID string) {
        endTrace := observability.StartTrace("process_order", "orderID", orderID)
        defer endTrace()
    
        // Processing logic...
    }

    // Starting a trace with an existing context.
    func handleRequest(ctx context.Context, orderID string) {
        endTrace := observability.StartTrace(ctx, "handle_request", "orderID", orderID)
        defer endTrace()
    
        // Request handling logic...
    }
```
If a transaction is already present (e.g., via APM-enabled Gin middleware), `StartTrace` creates a child span; otherwise, it starts a new transaction.

### Label-Based Routing

Logs can be annotated with labels to control routing:
```go
    log.Infow("Payment processed",
        "paymentID", "PAY123",
        "labels", []string{"accounting"},
    )
```
The label injection core processes the `labels` field:
- Logs tagged with specific labels may be routed to an override Elasticsearch index (e.g., "accounting-index").
- Alternatively, logs can be skipped for Elasticsearch if configured.

### Adding a New Zap Core

This observability package sets up multiple zap cores (console, file, Elasticsearch, etc.) and combines them into a single logger in the `initLogger` function. To add a new core you can follow the same pattern:

1. **Create Your `createXCore` Function**  
   In your new function, mirror the style of our existing core-creation functions (e.g., `createConsoleCore`, `createFileCore`, `createElasticsearchCore`). Typically, you will:
   - Configure an encoder (`zapcore.EncoderConfig`).
   - Set up a writer (e.g., to a file, stdout, a network sink, or anything else).
   - Return a new core using `zapcore.NewCore(...)`, along with the `atomicLevel` for controlling log levels.

2. **Add the New Core in `initLogger`**  
   Inside `initLogger` (where we build the logger), follow the approach used for the other cores:
   - Call your new `createXCore(...)` function.
   - Handle any errors (e.g., if creation fails, log a warning or decide how you want to proceed).
   - Append the newly created core to the `cores` slice, which is eventually passed to `zapcore.NewTee(cores...)`.

3. **Leverage Shared Fields and Wrappers**  
   Just like our other cores, you can attach shared fields (e.g., the DID field) by calling `.With([]zapcore.Field{ ... })` on the core. Also note that the package wraps the combined Tee core with `newLabelInjectionCore` (for label-based routing) and includes an event emitter core. Your new core will automatically benefit from these features so long as you add it in `initLogger` just like the others.

4. **Verify No-Op Mode and ES Disabled States (If Relevant)**  
   If your new core relies on external resources, consider how `noOpMode` or similar package-level flags (like `esDisabled`) might affect it. For instance, you may want to skip initialization if the environment or config dictates that your service shouldn’t produce logs to that destination at certain times.

5. **Document Configuration**  
   If your new core requires additional config fields (such as a new URL or API key), add them to `config.Observability` or wherever your configuration is stored, and update your function accordingly so that everything is initialized properly.

By following these steps and modeling your function after our existing `create*Core` methods, you can seamlessly add a new core to the combined logger in a way that remains consistent with the rest of the observability package.

---

## Runtime Configuration

Observability supports dynamic reconfiguration at runtime:

1. **Enable/Disable Elasticsearch Logging:**
```go
       err := observability.EnableElasticsearchLogging(true) // or false
       if err != nil {
           // handle error
       }
```
2. **Set Elasticsearch Endpoint:**
```go
       err := observability.SetElasticsearchEndpoint("https://new-elasticsearch.example.com:9200")
       if err != nil {
           // handle error
       }
```
3. **Set APM Server URL:**
```go
       err := observability.SetAPMURL("https://new-apm-server.example.com")
       if err != nil {
           // handle error
       }
```
4. **Update API Key (for Elasticsearch and APM):**
```go
       err := observability.SetAPIKey("newApiKey123")
       if err != nil {
           // handle error
       }
```
5. **Set Flush Interval for Elasticsearch Logs:**
```go
       err := observability.SetFlushInterval(15) // flush interval in seconds
       if err != nil {
           // handle error
       }
```
6. **Toggle No-Op Mode:**
```go
       observability.SetNoOpMode(true) // disables logging, tracing, and event emission
```
---

## Testing

For unit tests, disable observability side-effects by enabling no-op mode:
```go
  func TestNew(t *testing.T) {
    // Set observability to no-op mode for testing
    observability.SetNoOpMode(true)
    // some tests
    }
```
This ensures that logging, Elasticsearch, and APM tracing are effectively disabled during tests.

---

## Shutdown

To properly release resources and flush logs/traces, call:
```go
    observability.Shutdown()
```
This function:
- Flushes and closes Elasticsearch log buffers.
- Closes the custom event emitter.
- Shuts down the Elastic APM tracer if enabled.

Example usage in the main function:
```go
    func main() {
        // Application logic...
    
        // On graceful shutdown:
        observability.Shutdown()
    }
```
but it also happens automatically

## Local ELK Stack

Source: [Getting started with the Elastic Stack and Docker Compose: Part 2](https://www.elastic.co/blog/getting-started-with-the-elastic-stack-and-docker-compose-part-2)
Login: elastic / changeme

1. `git clone https://github.com/elkninja/elastic-stack-docker-part-two.git`
2. `cd elastic-stack-docker-part-two`
3. `docker-compose up`
4. Generate a certificate
   1. `docker cp es-cluster-es01-1:/usr/share/elasticsearch/config/certs/ca/ca.crt /tmp/.`
   2. `openssl x509 -fingerprint -sha256 -noout -in /tmp/ca.crt | awk -F"=" {' print $2 '} | sed s/://g`
   3. `cat /tmp/ca.crt`
5. Visit https://localhost:5601/app/fleet/settings/outputs/fleet-default-output
   - make the form [look like this image](https://static-www.elastic.co/v3/assets/bltefdd0b53724fa2ce/blt0fd5e8f5c53241e8/65256f40fbc4f6c0df9ae3af/Screenshot_2023-10-10_at_9.35.17_AM.png)
   - update "Hosts" to `https://es01:9200`
   - update "... fingerprint" to the output from step 4.2
   - update and indent "Advanced YAML config"
     - then "Save and Deploy"
```go
ssl:
  certificate_authorities:
  - |
    OUTPUT_STEP_4_3
```
6. [Create a data view](https://localhost:5601/app/management/kibana/dataViews) called "dms-logs"
  - "Index pattern": `nunet-dms,*-index`
7. Create an API key
   - https://localhost:5601/app/management/security/api_keys/create
8. Update config:
    - `apm.secret_token = "supersecrettoken"`
    - `apm.server_url = "http://localhost:8200"`
    - `observability.elasticsearch_url = "https://localhost:9200"`
    - `observability.elasticsearch_api_key = "STEP6"`
9. Results available after stopping the DMS and waiting a bit:
   - Traces: [Observability -> APM -> nunet-dms -> Traces -> DMS](https://localhost:5601/app/apm/services/nunet-dms/overview?comparisonEnabled=true&environment=ENVIRONMENT_ALL&kuery=&latencyAggregationType=avg&offset=1d&rangeFrom=now-15m&rangeTo=now&serviceGroup=)
   - Logs: [Analytics -> Discover -> dms-logs](https://localhost:5601/app/management/data/index_management/data_streams/logs-nunet-dms) and [Management -> Stack Management -> Index Management](https://localhost:5601/app/discover#)
