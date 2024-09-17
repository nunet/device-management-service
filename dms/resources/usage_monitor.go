package resources

import (
	"context"

	"gitlab.com/nunet/device-management-service/types"
)

// defaultUsageMonitor implements the UsageMonitor interface
type defaultUsageMonitor struct{}

// newUsageMonitor creates a new defaultUsageMonitor
func newUsageMonitor() *defaultUsageMonitor {
	return &defaultUsageMonitor{}
}

var _ types.UsageMonitor = (*defaultUsageMonitor)(nil)

// GetUsage returns the resources used by the machine
func (um *defaultUsageMonitor) GetUsage(_ context.Context) (types.Resources, error) {
	panic("implement me")
}
