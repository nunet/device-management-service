// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package observability

import (
	"fmt"
	"net/url"
	"os"
	"time"

	"gitlab.com/nunet/device-management-service/internal/config"
	"go.elastic.co/apm"
	"go.elastic.co/apm/transport"
)

// tracingNoOpMode indicates whether tracing is in no-op mode
var tracingNoOpMode bool

// initTracing initializes the Elastic APM tracer
func initTracing(apmConfig config.APM) {
	if IsNoOpMode() {
		tracingNoOpMode = true
		return
	}

	// Check if the necessary configurations are present
	if apmConfig.ServerURL == "" || apmConfig.ServiceName == "" || apmConfig.Environment == "" {
		fmt.Fprintf(os.Stderr, "Warning: APM configurations are incomplete, tracing will be disabled\n")
		tracingNoOpMode = true
		return
	}

	// Create a new APM transport
	tr, err := transport.NewHTTPTransport()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to create APM transport: %v\n", err)
		tracingNoOpMode = true
		return // Proceed without tracing
	}

	// Parse and set the APM Server URL
	serverURL, err := url.Parse(apmConfig.ServerURL)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to parse APM server URL: %v\n", err)
		tracingNoOpMode = true
		return
	}
	tr.SetServerURL(serverURL)

	// Set API key if provided
	if apmConfig.APIKey != "" {
		tr.SetSecretToken(apmConfig.APIKey)
	}

	// Set the environment variable for the tracer's environment
	if err := os.Setenv("ELASTIC_APM_ENVIRONMENT", apmConfig.Environment); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to set environment: %v\n", err)
		tracingNoOpMode = true
		return
	}

	// Initialize the APM tracer
	apm.DefaultTracer, err = apm.NewTracer(apmConfig.ServiceName, "1.0.0")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Failed to initialize APM tracer: %v\n", err)
		tracingNoOpMode = true
		return
	}

	// Set the transport for the tracer
	apm.DefaultTracer.Transport = tr
}

// StartTrace starts a trace for the given operationName and key-value pairs.
// It returns a function that should be deferred to end the trace.
func StartTrace(operationName string, keyValues ...interface{}) func() {
	if IsNoOpMode() || tracingNoOpMode {
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
