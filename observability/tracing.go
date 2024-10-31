package observability

import (
	"fmt"
	"net/url"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"gitlab.com/nunet/device-management-service/internal/config"
	"go.elastic.co/apm"
	"go.elastic.co/apm/transport"
)

var (
	// noOpMode indicates whether tracing is in no-op mode
	noOpMode bool
	// mutex to protect access to noOpMode
	mutex sync.RWMutex
	// log is the logger for the observability package
	log = logging.Logger("observability")
)

func initTracing(apmConfig config.APM) error {
	// Create a new APM transport
	tr, err := transport.NewHTTPTransport()
	if err != nil {
		return fmt.Errorf("failed to create APM transport: %w", err)
	}

	// Parse the APM Server URL
	serverURL, err := url.Parse(apmConfig.ServerURL)
	if err != nil {
		return fmt.Errorf("failed to parse APM server URL: %w", err)
	}

	// Set the APM Server URL
	tr.SetServerURL(serverURL)

	// Create a new tracer with the transport and set the environment
	apm.DefaultTracer, err = apm.NewTracerOptions(apm.TracerOptions{
		ServiceName:        apmConfig.ServiceName,
		ServiceVersion:     "1.0.0",
		ServiceEnvironment: apmConfig.Environment,
		Transport:          tr,
	})
	if err != nil {
		return fmt.Errorf("failed to create APM tracer: %w", err)
	}

	return nil
}

// StartTrace starts a trace for the given operationName and key-value pairs.
// It returns a function that should be deferred to end the trace.
func StartTrace(operationName string, keyValues ...interface{}) func() {
	if isNoOp() {
		return func() {}
	}

	// Start an Elastic APM transaction
	tx := apm.DefaultTracer.StartTransaction(operationName, "custom")

	// Record the start time
	startTime := time.Now()

	// Log the start of the operation with the original naming
	logFields := append([]interface{}{
		"startTime", startTime,
		"trace.id", tx.TraceContext().Trace.String(),
		"transaction.id", tx.TraceContext().Span.String(),
	}, keyValues...)
	log.Infow(operationName+"_start", logFields...)

	// Return the EndTrace function
	return func() {
		// Calculate duration
		endTime := time.Now()
		duration := endTime.Sub(startTime)

		// Log the end of the operation
		logFields = append([]interface{}{
			"endTime", endTime,
			"duration", duration,
			"trace.id", tx.TraceContext().Trace.String(),
			"transaction.id", tx.TraceContext().Span.String(),
		}, keyValues...)
		log.Infow(operationName+"_end", logFields...)

		// End the transaction
		tx.End()
	}
}

// SetNoOpMode enables or disables the no-op mode for tracing.
func SetNoOpMode(enabled bool) {
	mutex.Lock()
	defer mutex.Unlock()
	noOpMode = enabled
}

// isNoOp checks if the tracer is in no-op mode.
func isNoOp() bool {
	mutex.RLock()
	defer mutex.RUnlock()
	return noOpMode
}
