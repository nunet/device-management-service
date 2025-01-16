// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
	"github.com/shirou/gopsutil/v4/host"
	"github.com/shirou/gopsutil/v4/load"
	"github.com/shirou/gopsutil/v4/mem"
	"github.com/shirou/gopsutil/v4/net"
	"gitlab.com/nunet/device-management-service/internal/config"
	"go.elastic.co/apm"
	"go.elastic.co/apm/transport"
)

var (
	tracingNoOpMode bool
	tracerMutex     sync.Mutex
	currentTracer   *apm.Tracer
)

// initTracing initializes or reinitializes the Elastic APM tracer.
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

	// Check required APM configs
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

	tracer.SetMetricsInterval(10 * time.Second)
	apm.DefaultTracer = tracer
	currentTracer = tracer
	tracingNoOpMode = false

	// Register custom metrics
	registerCustomMetrics(tracer)
}

func collectSystemMetrics() map[string]interface{} {
	metrics := make(map[string]interface{})

	// CPU usage
	if cpuUsage, err := cpu.Percent(0, false); err == nil && len(cpuUsage) > 0 {
		metrics["cpuUsage"] = cpuUsage[0]
	}

	// RAM usage
	if v, err := mem.VirtualMemory(); err == nil {
		metrics["ramUsed"] = v.Used
		metrics["ramTotal"] = v.Total
	}

	// Disk usage
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

	// Uptime
	if uptime, err := host.Uptime(); err == nil {
		metrics["uptime"] = float64(uptime)
	}

	// Load average
	if avg, err := load.Avg(); err == nil {
		metrics["load15"] = avg.Load15
	}

	// Network RX/TX
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

		// CPU usage
		if cpuUsage, ok := metrics["cpuUsage"].(float64); ok {
			m.Add("system.cpu.total.norm.pct", didLabel, cpuUsage/100.0)
		}

		// RAM usage
		if ramUsed, ok := metrics["ramUsed"].(uint64); ok {
			m.Add("system.memory.actual.used.bytes", didLabel, float64(ramUsed))
		}
		if ramTotal, ok := metrics["ramTotal"].(uint64); ok {
			m.Add("system.memory.total", didLabel, float64(ramTotal))
		}

		// Disk usage
		if diskUsed, ok := metrics["diskUsed"].(uint64); ok {
			m.Add("system.filesystem.used.bytes", didLabel, float64(diskUsed))
		}
		if diskTotal, ok := metrics["diskTotal"].(uint64); ok {
			m.Add("system.filesystem.total", didLabel, float64(diskTotal))
		}

		// Uptime
		if uptime, ok := metrics["uptime"].(float64); ok {
			m.Add("system.uptime", didLabel, uptime)
		}

		// Load average
		if load15, ok := metrics["load15"].(float64); ok {
			m.Add("system.load.15", didLabel, load15)
		}

		// Network RX/TX
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

// StartTrace is a unified entry point to start instrumentation.
//
// Usage patterns:
//   - StartTrace(operationName string, keyValues ...interface{})
//   - StartTrace(ctx context.Context, operationName string, keyValues ...interface{})
//   - StartTrace(c *gin.Context, operationName string, keyValues ...interface{})
//
// Logic:
//  1. If we find an existing transaction in ctx (e.g., from apmgin), start a span.
//  2. If no existing transaction is found, start a new "request"-type transaction.
func StartTrace(args ...interface{}) func() {
	var ctx context.Context
	var operationName string
	var keyValues []interface{}

	if len(args) == 0 {
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
			log.Error("Operation name must be a string when called with *gin.Context")
			return func() {}
		}
	case context.Context:
		ctx = v
		if len(args) < 2 {
			log.Error("StartTrace called with context but without operation name")
			return func() {}
		}
		if opName, ok := args[1].(string); ok {
			operationName = opName
			keyValues = args[2:]
		} else {
			log.Error("Operation name must be a string when called with context.Context")
			return func() {}
		}
	default:
		log.Error("Unsupported first argument type for StartTrace")
		return func() {}
	}

	return startTrace(ctx, operationName, keyValues...)
}

func startTrace(ctx context.Context, operationName string, keyValues ...interface{}) func() {
	tracerMutex.Lock()
	noOp := tracingNoOpMode
	tracer := currentTracer
	tracerMutex.Unlock()

	if IsNoOpMode() || noOp || tracer == nil {
		return func() {}
	}

	existingTx := apm.TransactionFromContext(ctx)
	if existingTx != nil {
		// create a span inside existing request transaction
		span, _ := apm.StartSpan(ctx, operationName, "custom")
		if span.Dropped() {
			return func() {}
		}

		span.Context.SetLabel("did", didID.String())
		for i := 0; i < len(keyValues); i += 2 {
			if i+1 < len(keyValues) {
				if key, ok := keyValues[i].(string); ok {
					span.Context.SetLabel(key, keyValues[i+1])
				}
			}
		}

		startTime := time.Now()
		log.Info("Operation started inside existing request transaction",
			"operation", operationName,
			"trace.id", existingTx.TraceContext().Trace.String(),
			"transaction.id", existingTx.TraceContext().Span.String())

		return func() {
			duration := time.Since(startTime)
			log.Info("Operation ended",
				"operation", operationName,
				"duration", duration,
				"trace.id", existingTx.TraceContext().Trace.String(),
				"transaction.id", existingTx.TraceContext().Span.String())
			span.End()
		}
	}

	// else: start a new "request"-type transaction
	newTx := tracer.StartTransaction(operationName, "request")
	ctx = apm.ContextWithTransaction(ctx, newTx)
	newTx.Context.SetLabel("did", didID.String())

	span, _ := apm.StartSpan(ctx, operationName+"_span", "custom")
	if span.Dropped() {
		newTx.End()
		return func() {}
	}

	span.Context.SetLabel("did", didID.String())
	for i := 0; i < len(keyValues); i += 2 {
		if i+1 < len(keyValues) {
			if key, ok := keyValues[i].(string); ok {
				span.Context.SetLabel(key, keyValues[i+1])
			}
		}
	}

	startTime := time.Now()
	log.Info("Operation started with new request-like transaction",
		"operation", operationName,
		"trace.id", newTx.TraceContext().Trace.String(),
		"transaction.id", newTx.TraceContext().Span.String())

	return func() {
		duration := time.Since(startTime)
		log.Info("Operation ended",
			"operation", operationName,
			"duration", duration,
			"trace.id", newTx.TraceContext().Trace.String(),
			"transaction.id", newTx.TraceContext().Span.String())
		span.End()
		newTx.End()
	}
}

// shutdownTracer closes the current tracer
func shutdownTracer() {
	tracerMutex.Lock()
	defer tracerMutex.Unlock()

	if currentTracer != nil {
		currentTracer.Close()
		currentTracer = nil
	}
}
