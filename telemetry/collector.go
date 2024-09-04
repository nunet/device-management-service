package telemetry

import (
	"context"

	"gitlab.com/nunet/device-management-service/types"
)

type Collector interface {
	Initialize() error
	SpanContext(ctx context.Context, span string) (context.Context, context.CancelFunc)
	HandleEvent(event types.Event) error
	Flush() error
	Shutdown() error
	GetName() string
}
