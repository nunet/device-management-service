// tracing.go
package observability

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/shirou/gopsutil/host"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/net"
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

	// Set the metrics collection interval
	tracer.SetMetricsInterval(10 * time.Second) // Adjust the interval as needed

	// Set the default tracer
	apm.DefaultTracer = tracer
	currentTracer = tracer
	tracingNoOpMode = false

	// Register custom metrics
	registerCustomMetrics(tracer)
}

func collectSystemMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// Get CPU usage
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
			if usage, err := disk.Usage(part.Mountpoint); err == nil {
				used += usage.Used
				total += usage.Total
			}
		}
		metrics["diskUsed"] = used
		metrics["diskTotal"] = total
	}

	// Get uptime
	if uptime, err := host.Uptime(); err == nil {
		metrics["uptime"] = float64(uptime)
	}

	// Get load average
	if avg, err := load.Avg(); err == nil {
		metrics["load15"] = avg.Load15
	}

	// Get network RX/TX
	if ioStats, err := net.IOCounters(false); err == nil && len(ioStats) > 0 {
		metrics["rxBytes"] = ioStats[0].BytesRecv
		metrics["txBytes"] = ioStats[0].BytesSent
	}

	return metrics
}

func registerCustomMetrics(tracer *apm.Tracer) {
	gatherer := apm.GatherMetricsFunc(func(_ context.Context, m *apm.Metrics) error {
		metrics := collectSystemMetrics()

		// Use DID as hostname
		didAsHostname := didID.String()
		didLabel := []apm.MetricLabel{{Name: "hostdid", Value: didAsHostname}}

		// Add CPU usage
		if cpuUsage, ok := metrics["cpuUsage"].(float64); ok {
			m.Add("system.cpu.total.norm.pct", didLabel, cpuUsage/100.0)
		}

		// Add RAM metrics
		if ramUsed, ok := metrics["ramUsed"].(uint64); ok {
			m.Add("system.memory.actual.used.bytes", didLabel, float64(ramUsed))
		}
		if ramTotal, ok := metrics["ramTotal"].(uint64); ok {
			m.Add("system.memory.total", didLabel, float64(ramTotal))
		}

		// Add Disk metrics
		if diskUsed, ok := metrics["diskUsed"].(uint64); ok {
			m.Add("system.filesystem.used.bytes", didLabel, float64(diskUsed))
		}
		if diskTotal, ok := metrics["diskTotal"].(uint64); ok {
			m.Add("system.filesystem.total", didLabel, float64(diskTotal))
		}

		// Add uptime
		if uptime, ok := metrics["uptime"].(float64); ok {
			m.Add("system.uptime", didLabel, uptime)
		}

		// Add load average
		if load15, ok := metrics["load15"].(float64); ok {
			m.Add("system.load.15", didLabel, load15)
		}

		// Add network RX/TX
		if rxBytes, ok := metrics["rxBytes"].(uint64); ok {
			m.Add("system.network.in.bytes", didLabel, float64(rxBytes))
		}
		if txBytes, ok := metrics["txBytes"].(uint64); ok {
			m.Add("system.network.out.bytes", didLabel, float64(txBytes))
		}

		return nil
	})

	tracer.RegisterMetricsGatherer(gatherer)
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

	// Log the start of the operation
	logger := log.With("operation", operationName).
		With("startTime", startTime).
		With("trace.id", tx.TraceContext().Trace.String()).
		With("transaction.id", tx.TraceContext().Span.String()).
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
		// Calculate duration
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		// Log the end of the operation
		logger := log.With("operation", operationName).
			With("endTime", endTime).
			With("duration", duration).
			With("trace.id", tx.TraceContext().Trace.String()).
			With("transaction.id", tx.TraceContext().Span.String()).
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

// shutdownTracer cleans up resources used by the tracer
func shutdownTracer() {
	tracerMutex.Lock()
	defer tracerMutex.Unlock()

	if currentTracer != nil {
		currentTracer.Close()
		currentTracer = nil
	}
}
