package telemetry

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/types"
)

func TestInitMockTelemetry(t *testing.T) {
	// Initialize the mock telemetry system for testing
	mockTelemetry := NewMockTelemetry(&types.TelemetryConfig{})
	mockCollector := NewMockCollector("mock_collector")
	mockTelemetry.AddCollector(mockCollector)

	// Verify that the mock telemetry and collectors are properly initialized
	assert.NotNil(t, mockTelemetry, "Mock telemetry should be initialized")
	assert.NotNil(t, mockCollector, "Mock collector should be initialized")
	assert.Len(t, mockTelemetry.collectors, 1, "There should be one collector in the mock telemetry")

	// Perform other tests as necessary
}

func TestStartPeriodicFlush(t *testing.T) {
	// Initialize global telemetry
	err := InitGlobalTelemetry()
	assert.NoError(t, err, "Global telemetry initialization should not return an error")

	// Start periodic flush with a short interval for testing
	StartPeriodicFlush(100 * time.Millisecond)

	// Ensure the periodic flush doesn't cause any errors or panics
	time.Sleep(500 * time.Millisecond)

	// Clean up after the test
	instance.Shutdown()
}

func TestFlushAndShutdown(t *testing.T) {
	// Initialize global telemetry
	err := InitGlobalTelemetry()
	assert.NoError(t, err, "Global telemetry initialization should not return an error")

	// Ensure Flush and Shutdown don't cause any issues
	instance.Flush()
	instance.Shutdown()
}
