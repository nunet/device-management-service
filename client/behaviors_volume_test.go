package client_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/node"
)

func TestClient_CreateVolume(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.VolumeCreateBehavior
	tests := []struct {
		name    string
		req     node.CreateVolumeRequest
		resp    node.CreateVolumeResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.CreateVolumeRequest{
				Name:      "",
				ClientPEM: "",
			},
			node.CreateVolumeResponse{
				OK:    false,
				Error: "",
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.CreateVolume(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_DeleteVolume(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.VolumeDeleteBehavior
	tests := []struct {
		name    string
		req     node.DeleteVolumeRequest
		resp    node.DeleteVolumeResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.DeleteVolumeRequest{
				Name: "",
			},
			node.DeleteVolumeResponse{
				OK:    false,
				Error: "",
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.DeleteVolume(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_StartVolume(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.VolumeStartBehavior
	tests := []struct {
		name    string
		req     node.StartVolumeRequest
		resp    node.StartVolumeResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.StartVolumeRequest{
				Name: "",
			},
			node.StartVolumeResponse{
				OK:    false,
				Error: "",
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.StartVolume(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}
