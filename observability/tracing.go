// tracing.go
package observability

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/disk"
	"github.com/shirou/gopsutil/v4/mem"
	"gitlab.com/nunet/device-management-service/internal/config"
	"go.elastic.co/apm"
	"go.elastic.co/apm/transport"
)

var (
	// tracingNoOpMode indicates whether tracing is in no-op mode
	tracingNoOpMode bool
	tracerMutex     sync.Mutex
	currentTracer   *apm.Tracer
)

// initTracing initializes or reinitializes the Elastic APM tracer
func initTracing(apmConfig config.APM) {
	tracerMutex.Lock()
	defer tracerMutex.Unlock()

	if IsNoOpMode() {
		tracingNoOpMode = true
		return
	}

	// Close existing tracer if any
	if currentTracer != nil {
		currentTracer.Close()
		currentTracer = nil
	}

	// Check if the necessary configurations are present
	if apmConfig.ServerURL == "" || apmConfig.ServiceName == "" || apmConfig.Environment == "" {
		log.Warn("APM configurations are incomplete, tracing will be disabled")
		tracingNoOpMode = true
		return
	}

	// Create a new APM transport
	tr, err := transport.NewHTTPTransport()
	if err != nil {
		log.Warnf("Failed to create APM transport: %v", err)
		tracingNoOpMode = true
		return
	}

	// Parse and set the APM Server URL
	serverURL, err := url.Parse(apmConfig.ServerURL)
	if err != nil {
		log.Warnf("Failed to parse APM server URL: %v", err)
		tracingNoOpMode = true
		return
	}
	tr.SetServerURL(serverURL)

	// Set API key if provided
	if apmConfig.APIKey != "" {
		tr.SetAPIKey(apmConfig.APIKey)
	}

	// Initialize the APM tracer
	tracer, err := apm.NewTracerOptions(apm.TracerOptions{
		ServiceName:    apmConfig.ServiceName,
		ServiceVersion: "1.0.0",
		Transport:      tr,
	})
	if err != nil {
		log.Warnf("Failed to initialize APM tracer: %v", err)
		tracingNoOpMode = true
		return
	}

	// Set the default tracer
	apm.DefaultTracer = tracer
	currentTracer = tracer
	tracingNoOpMode = false
}

// StartTrace starts a trace for the given operation.
// It can be called in one of the following ways:
// - StartTrace(operationName string, keyValues ...interface{}) func()
// - StartTrace(ctx context.Context, operationName string, keyValues ...interface{}) func()
// - StartTrace(c *gin.Context, operationName string, keyValues ...interface{}) func()
func StartTrace(args ...interface{}) func() {
	var ctx context.Context
	var operationName string
	var keyValues []interface{}

	if len(args) == 0 {
		// Invalid usage, return a no-op function
		log.Error("StartTrace called without arguments")
		return func() {}
	}

	// Determine if the first argument is a context or operation name
	switch v := args[0].(type) {
	case string:
		// No context provided
		ctx = context.Background()
		operationName = v
		keyValues = args[1:]
	case *gin.Context:
		ctx = v.Request.Context()
		if len(args) < 2 {
			log.Error("StartTrace called with *gin.Context but without operation name")
			return func() {}
		}
		if opName, ok := args[1].(string); ok {
			operationName = opName
			keyValues = args[2:]
		} else {
			log.Error("StartTrace operation name must be a string when called with *gin.Context")
			return func() {}
		}
	case context.Context:
		ctx = v
		if len(args) < 2 {
			log.Error("StartTrace called with context.Context but without operation name")
			return func() {}
		}
		if opName, ok := args[1].(string); ok {
			operationName = opName
			keyValues = args[2:]
		} else {
			log.Error("StartTrace operation name must be a string when called with context.Context")
			return func() {}
		}
	default:
		log.Error("StartTrace unsupported first argument type")
		return func() {}
	}

	return startTrace(ctx, operationName, keyValues...)
}

func startTrace(ctx context.Context, operationName string, keyValues ...interface{}) func() {
	tracerMutex.Lock()
	noOp := tracingNoOpMode
	tracer := currentTracer
	tracerMutex.Unlock()

	if IsNoOpMode() || noOp {
		return func() {}
	}

	if tracer == nil {
		return func() {}
	}

	// Start an Elastic APM transaction
	tx := tracer.StartTransaction(operationName, "custom")
	// Create a context with the transaction for propagation
	ctx = apm.ContextWithTransaction(ctx, tx)

	// Add the DID label to the transaction
	tx.Context.SetLabel("did", didID.String())

	// Collect initial system metrics
	initialMetrics := collectSystemMetrics()

	// Start a span within the transaction
	span, _ := apm.StartSpan(ctx, operationName+"_span", "custom")
	if span.Dropped() {
		// If the span was dropped due to sampling, proceed without it
		span = nil
	} else {
		// Add the DID label to the span
		span.Context.SetLabel("did", didID.String())
	}

	// Record the start time
	startTime := time.Now()

	// Log the start of the operation with metrics
	logger := log.With("operation", operationName).
		With("startTime", startTime).
		With("trace.id", tx.TraceContext().Trace.String()).
		With("transaction.id", tx.TraceContext().Span.String()).
		With("metrics", initialMetrics).
		With("did", didID.String()) // Include DID in the log entry

	if span != nil {
		logger = logger.With("span.id", span.TraceContext().Span.String())
	}

	// Include additional key-values
	for i := 0; i < len(keyValues); i += 2 {
		if i+1 < len(keyValues) {
			key, ok := keyValues[i].(string)
			if ok {
				logger = logger.With(key, keyValues[i+1])
			}
		}
	}

	logger.Info("Operation started")

	// Return the end function
	return func() {
		// Collect final system metrics
		finalMetrics := collectSystemMetrics()

		// Calculate duration
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		// Log the end of the operation with metrics
		logger := log.With("operation", operationName).
			With("endTime", endTime).
			With("duration", duration).
			With("trace.id", tx.TraceContext().Trace.String()).
			With("transaction.id", tx.TraceContext().Span.String()).
			With("finalMetrics", finalMetrics).
			With("did", didID.String()) // Include DID in the log entry

		if span != nil {
			logger = logger.With("span.id", span.TraceContext().Span.String())
			span.End()
		}

		// Include additional key-values
		for i := 0; i < len(keyValues); i += 2 {
			if i+1 < len(keyValues) {
				key, ok := keyValues[i].(string)
				if ok {
					logger = logger.With(key, keyValues[i+1])
				}
			}
		}

		logger.Info("Operation ended")

		tx.End()
	}
}

// collectSystemMetrics gathers system metrics like CPU, RAM, and Disk usage
func collectSystemMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Get CPU usage (non-blocking)
	if cpuUsage, err := cpu.Percent(0, false); err == nil && len(cpuUsage) > 0 {
		metrics["cpuUsage"] = cpuUsage[0]
	}

	// Get RAM usage
	if v, err := mem.VirtualMemory(); err == nil {
		metrics["ramUsed"] = v.Used
		metrics["ramTotal"] = v.Total
	}

	// Get Disk usage
	if partitions, err := disk.Partitions(false); err == nil {
		var used, total uint64
		for _, part := range partitions {
			select {
			case <-context.Background().Done():
				return metrics
			default:
				if usage, err := disk.Usage(part.Mountpoint); err == nil {
					used += usage.Used
					total += usage.Total
				}
			}
		}
		metrics["diskUsed"] = used
		metrics["diskTotal"] = total
	}

	return metrics
}

// shutdownTracer cleans up resources used by the tracer
func shutdownTracer() {
	tracerMutex.Lock()
	defer tracerMutex.Unlock()

	if currentTracer != nil {
		currentTracer.Close()
		currentTracer = nil
	}
}
