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
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/types"
)

func TestMockTelemetry_Trace(t *testing.T) {
	mockTelemetry := NewMockTelemetry(&types.TelemetryConfig{})
	mockCollector := NewMockCollector("mock_collector")
	mockTelemetry.AddCollector(mockCollector)

	ctx := context.Background()
	ctx, cancel := mockTelemetry.SpanContext(ctx, "test_tracer", "test_span", "mock_collector")
	defer cancel()

	mockTelemetry.Trace(ctx, "Test Trace Message", map[string]interface{}{"key": "value"})

	events := mockCollector.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, types.TRACE, events[0].Level)
	assert.Equal(t, "Test Trace Message", events[0].Message)
	assert.Equal(t, "value", events[0].Payload["key"])
}

func TestMockTelemetry_Debug(t *testing.T) {
	mockTelemetry := NewMockTelemetry(&types.TelemetryConfig{})
	mockCollector := NewMockCollector("mock_collector")
	mockTelemetry.AddCollector(mockCollector)

	ctx := context.Background()
	mockTelemetry.Debug(ctx, "Test Debug Message", map[string]interface{}{"key": "value"})

	events := mockCollector.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, types.DEBUG, events[0].Level)
	assert.Equal(t, "Test Debug Message", events[0].Message)
	assert.Equal(t, "value", events[0].Payload["key"])
}

func TestMockTelemetry_Info(t *testing.T) {
	mockTelemetry := NewMockTelemetry(&types.TelemetryConfig{})
	mockCollector := NewMockCollector("mock_collector")
	mockTelemetry.AddCollector(mockCollector)

	ctx := context.Background()
	mockTelemetry.Info(ctx, "Test Info Message", map[string]interface{}{"key": "value"})

	events := mockCollector.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, types.INFO, events[0].Level)
	assert.Equal(t, "Test Info Message", events[0].Message)
	assert.Equal(t, "value", events[0].Payload["key"])
}

func TestMockTelemetry_Warn(t *testing.T) {
	mockTelemetry := NewMockTelemetry(&types.TelemetryConfig{})
	mockCollector := NewMockCollector("mock_collector")
	mockTelemetry.AddCollector(mockCollector)

	ctx := context.Background()
	mockTelemetry.Warn(ctx, "Test Warn Message", map[string]interface{}{"key": "value"})

	events := mockCollector.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, types.WARN, events[0].Level)
	assert.Equal(t, "Test Warn Message", events[0].Message)
	assert.Equal(t, "value", events[0].Payload["key"])
}

func TestMockTelemetry_Error(t *testing.T) {
	mockTelemetry := NewMockTelemetry(&types.TelemetryConfig{})
	mockCollector := NewMockCollector("mock_collector")
	mockTelemetry.AddCollector(mockCollector)

	ctx := context.Background()
	mockTelemetry.Error(ctx, "Test Error Message", map[string]interface{}{"key": "value"})

	events := mockCollector.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, types.ERROR, events[0].Level)
	assert.Equal(t, "Test Error Message", events[0].Message)
	assert.Equal(t, "value", events[0].Payload["key"])
}

func TestMockTelemetry_Fatal(t *testing.T) {
	mockTelemetry := NewMockTelemetry(&types.TelemetryConfig{})
	mockCollector := NewMockCollector("mock_collector")
	mockTelemetry.AddCollector(mockCollector)

	ctx := context.Background()
	mockTelemetry.Fatal(ctx, "Test Fatal Message", map[string]interface{}{"key": "value"})

	events := mockCollector.GetEvents()
	assert.Len(t, events, 1)
	assert.Equal(t, types.FATAL, events[0].Level)
	assert.Equal(t, "Test Fatal Message", events[0].Message)
	assert.Equal(t, "value", events[0].Payload["key"])
}

func TestMockTelemetry_SpanContext(t *testing.T) {
	mockTelemetry := NewMockTelemetry(&types.TelemetryConfig{})
	mockCollector := NewMockCollector("mock_collector")
	mockTelemetry.AddCollector(mockCollector)

	ctx := context.Background()
	ctx, cancel := mockTelemetry.SpanContext(ctx, "test_tracer", "test_span", "mock_collector")
	defer cancel() // Ensure that the cancel function is called to avoid resource leaks

	mockTelemetry.Trace(ctx, "Test Trace with Tracing", map[string]interface{}{"key": "value"})

	traces := mockCollector.GetTraces()
	assert.Len(t, traces, 1) // Ensures that at least one trace was recorded
	assert.Equal(t, "test_span", traces[0].SpanName)
}

func TestMockTelemetry_Flush(_ *testing.T) {
	mockTelemetry := NewMockTelemetry(&types.TelemetryConfig{})
	mockCollector := NewMockCollector("mock_collector")
	mockTelemetry.AddCollector(mockCollector)

	mockTelemetry.Flush()
	// No specific assertions here, as we are just testing that Flush doesn't panic
}

func TestMockTelemetry_Shutdown(_ *testing.T) {
	mockTelemetry := NewMockTelemetry(&types.TelemetryConfig{})
	mockCollector := NewMockCollector("mock_collector")
	mockTelemetry.AddCollector(mockCollector)

	mockTelemetry.Shutdown()
	// No specific assertions here, as we are just testing that Shutdown doesn't panic
}
