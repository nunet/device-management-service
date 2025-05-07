package client_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/node"
)

func TestAllocationsList(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.AllocationsListBehavior
	tests := []struct {
		name    string
		resp    node.AllocationsListResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.AllocationsListResponse{
				Allocations: []jobs.AllocationInfo{},
				Error:       "",
			},
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				assert.Empty(t, envelope.Message)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.AllocationsList(context.Background(), tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}
