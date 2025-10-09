// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package client_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/types"
)

func TestClient_ResourcesAllocated(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.ResourcesAllocatedBehavior
	tests := []struct {
		name    string
		resp    node.ResourcesResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.ResourcesResponse{
				OK:        false,
				Resources: types.Resources{},
				Error:     "",
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

			result, err := c.ResourcesAllocated(context.Background(), tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_ResourcesFree(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.ResourcesFreeBehavior
	tests := []struct {
		name    string
		resp    node.ResourcesResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.ResourcesResponse{
				Error:     "",
				OK:        false,
				Resources: types.Resources{},
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

			result, err := c.ResourcesFree(context.Background(), tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_ResourcesOnboarded(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.ResourcesOnboardedBehavior
	tests := []struct {
		name    string
		resp    node.ResourcesResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.ResourcesResponse{
				Error:     "",
				OK:        false,
				Resources: types.Resources{},
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

			result, err := c.ResourcesOnboarded(context.Background(), tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_HardwareSpec(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.HardwareSpecBehavior
	tests := []struct {
		name    string
		resp    node.ResourcesResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.ResourcesResponse{
				Error:     "",
				OK:        false,
				Resources: types.Resources{},
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

			result, err := c.HardwareSpec(context.Background(), tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_HardwareUsage(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.HardwareUsageBehavior
	tests := []struct {
		name    string
		resp    node.ResourcesResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.ResourcesResponse{
				Error:     "",
				OK:        false,
				Resources: types.Resources{},
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

			result, err := c.HardwareUsage(context.Background(), tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}
