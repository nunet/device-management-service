// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package telemetry

import (
	"context"
	"sync"

	"gitlab.com/nunet/device-management-service/types"
)

// MockCollector is a mock implementation of the Collector interface.
type MockCollector struct {
	mu          sync.Mutex
	events      []types.Event
	traces      []MockTrace
	initialized bool
	name        string
}

type MockTrace struct {
	SpanName   string
	Context    context.Context
	CancelFunc context.CancelFunc
}

// NewMockCollector creates a new instance of MockCollector.
func NewMockCollector(name string) *MockCollector {
	return &MockCollector{
		events: []types.Event{},
		traces: []MockTrace{},
		name:   name,
	}
}

// Initialize is a mock implementation of the Collector interface's Initialize method.
func (m *MockCollector) Initialize() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.initialized = true
	return nil
}

// SpanContext is a mock implementation of the Collector interface's SpanContext method.
func (m *MockCollector) SpanContext(ctx context.Context, spanName string) (context.Context, context.CancelFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()

	mockCtx, cancel := context.WithCancel(ctx)
	m.traces = append(m.traces, MockTrace{
		SpanName:   spanName,
		Context:    mockCtx,
		CancelFunc: cancel,
	})

	return mockCtx, cancel
}

// HandleEvent is a mock implementation of the Collector interface's HandleEvent method.
func (m *MockCollector) HandleEvent(event types.Event) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = append(m.events, event)
	return nil
}

// Flush is a mock implementation of the Collector interface's Flush method.
func (m *MockCollector) Flush() error { // Added error return
	m.mu.Lock()
	defer m.mu.Unlock()

	return nil
}

// Shutdown is a mock implementation of the Collector interface's Shutdown method.
func (m *MockCollector) Shutdown() error { // Added error return
	m.mu.Lock()
	defer m.mu.Unlock()

	return nil
}

// GetName returns the name of the mock collector.
func (m *MockCollector) GetName() string {
	return m.name
}

// GetTraces returns the recorded traces.
func (m *MockCollector) GetTraces() []MockTrace {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.traces
}

// GetEvents returns the recorded events.
func (m *MockCollector) GetEvents() []types.Event {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.events
}

// Reset clears all recorded events and traces.
func (m *MockCollector) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.events = []types.Event{}
	m.traces = []MockTrace{}
}

// AssertInitialized checks if the mock collector was initialized.
func (m *MockCollector) AssertInitialized() bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.initialized
}

// MockTelemetry is a mock implementation of the Telemetry system.
type MockTelemetry struct {
	Telemetry
	mu         sync.Mutex
	collectors map[string]*MockCollector
}

// NewMockTelemetry creates a new instance of MockTelemetry that mimics the Telemetry struct.
func NewMockTelemetry(config *types.TelemetryConfig) *MockTelemetry {
	return &MockTelemetry{
		Telemetry: Telemetry{
			config:     config,
			collectors: make(map[string]Collector),
		},
		collectors: make(map[string]*MockCollector),
	}
}

// AddCollector adds a mock collector to the telemetry system.
func (m *MockTelemetry) AddCollector(collector *MockCollector) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.collectors[collector.GetName()] = collector
}

// SpanContext simulates starting a trace with the given collectors.
func (m *MockTelemetry) SpanContext(ctx context.Context, _ string, span string, collectors ...string) (context.Context, context.CancelFunc) { // Renamed unused parameter
	var cancelFuncs []context.CancelFunc

	for _, collectorName := range collectors {
		if collector, ok := m.collectors[collectorName]; ok {
			mockCtx, cancel := collector.SpanContext(ctx, span)
			cancelFuncs = append(cancelFuncs, cancel)
			ctx = mockCtx
		}
	}

	cancel := func() {
		for _, cancelFunc := range cancelFuncs {
			cancelFunc()
		}
	}

	return ctx, cancel
}

// Trace simulates logging a trace event in all added collectors.
func (m *MockTelemetry) Trace(ctx context.Context, message string, payload map[string]interface{}) {
	m.logEvent(ctx, types.TRACE, message, payload)
}

// Debug simulates logging a debug event in all added collectors.
func (m *MockTelemetry) Debug(ctx context.Context, message string, payload map[string]interface{}) {
	m.logEvent(ctx, types.DEBUG, message, payload)
}

// Info simulates logging an info event in all added collectors.
func (m *MockTelemetry) Info(ctx context.Context, message string, payload map[string]interface{}) {
	m.logEvent(ctx, types.INFO, message, payload)
}

// Warn simulates logging a warning event in all added collectors.
func (m *MockTelemetry) Warn(ctx context.Context, message string, payload map[string]interface{}) {
	m.logEvent(ctx, types.WARN, message, payload)
}

// Error simulates logging an error event in all added collectors.
func (m *MockTelemetry) Error(ctx context.Context, message string, payload map[string]interface{}) {
	m.logEvent(ctx, types.ERROR, message, payload)
}

// Fatal simulates logging a fatal event in all added collectors.
func (m *MockTelemetry) Fatal(ctx context.Context, message string, payload map[string]interface{}) {
	m.logEvent(ctx, types.FATAL, message, payload)
}

// logEvent logs an event in all collectors.
func (m *MockTelemetry) logEvent(ctx context.Context, level types.ObservabilityLevel, message string, payload map[string]interface{}) {
	event := types.Event{
		Context: ctx,
		Level:   level,
		Message: message,
		Payload: payload,
	}

	for _, collector := range m.collectors {
		_ = collector.HandleEvent(event) // HandleEvent error is intentionally ignored
	}
}

// GetCollector returns a mock collector by name.
func (m *MockTelemetry) GetCollector(name string) *MockCollector {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.collectors[name]
}

// Reset clears all recorded data in all collectors.
func (m *MockTelemetry) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for _, collector := range m.collectors {
		collector.Reset()
	}
}

// Flush is a mock implementation of the Telemetry system's Flush method.
func (m *MockTelemetry) Flush() {
	for _, collector := range m.collectors {
		_ = collector.Flush() // Flush error is intentionally ignored
	}
}

// Shutdown is a mock implementation of the Telemetry system's Shutdown method.
func (m *MockTelemetry) Shutdown() {
	m.Flush()
	for _, collector := range m.collectors {
		_ = collector.Shutdown() // Shutdown error is intentionally ignored
	}
}
