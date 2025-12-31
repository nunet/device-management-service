package provider

import (
	"context"

	"gitlab.com/nunet/device-management-service/types"
)

// Provider defines a common interface for all infrastructure providers (VM, hosting, DCs).
type Provider interface {
	// Name returns a identifier for the provider
	Name() string

	// ListPlans returns all available server/VM plans for this provider.
	ListPlans(ctx context.Context) ([]Plan, error)

	// ProvisionServer provisions a new server using the ensemble config
	ProvisionServer(ctx context.Context, plan Plan, name, image, orchestratorDID string) (*Server, error)

	// DeleteServer permanently removes a server instance.
	DeleteServer(ctx context.Context, serverID string) error

	// RestartServer performs a reboot or restart operation on a running server.
	RestartServer(ctx context.Context, serverID string) error

	// GetServerStatus fetches the latest information about a server.
	GetServerStatus(ctx context.Context, serverID string) (*Server, error)

	// SelectMatchingPlan should returns a good match given the target resource requirements
	SelectMatchingPlan(plans []Plan, target types.Resources) (*Plan, error)
}
