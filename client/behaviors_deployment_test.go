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
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/node"
)

func TestClient_DeploymentList(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.DeploymentListBehavior
	tests := []struct {
		name    string
		req     node.DeploymentListRequest
		resp    node.DeploymentListResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.DeploymentListRequest{
				Metadata: map[string]string{},
			},
			node.DeploymentListResponse{
				Deployments: []node.DeploymentInfo{},
				Total:       1,
				HasMore:     false,
				NextOffset:  0,
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

			result, err := c.DeploymentList(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_DeploymentStatus(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.DeploymentStatusBehavior
	tests := []struct {
		name    string
		req     node.DeploymentStatusRequest
		resp    node.DeploymentStatusResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.DeploymentStatusRequest{
				ID: "",
			},
			node.DeploymentStatusResponse{
				Status: "",
				Error:  "",
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

			result, err := c.DeploymentStatus(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_DeploymentLogs(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.DeploymentLogsBehavior
	tests := []struct {
		name    string
		req     node.DeploymentLogsRequest
		resp    node.DeploymentLogsResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.DeploymentLogsRequest{
				EnsembleID:     "",
				AllocationName: "",
			},
			node.DeploymentLogsResponse{
				Error:         "",
				LogsWrittenTo: "",
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

			result, err := c.DeploymentLogs(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_DeploymentManifest(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.DeploymentManifestBehavior
	tests := []struct {
		name    string
		req     node.DeploymentManifestRequest
		resp    node.DeploymentManifestResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.DeploymentManifestRequest{
				ID: "",
			},
			node.DeploymentManifestResponse{
				Manifest: jobtypes.EnsembleManifest{},
				Error:    "",
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

			result, err := c.DeploymentManifest(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result.Manifest.ID, tt.resp.Manifest.ID, "manifest id not equal")
			assert.Equal(t, result.Manifest.Metadata, tt.resp.Manifest.Metadata, "manifest metadata not equal")
			assert.Equal(t, result.Manifest.Allocations, tt.resp.Manifest.Allocations, "manifest allocations not equal")
			assert.Equal(t, result.Manifest.Nodes, tt.resp.Manifest.Nodes, "manifest nodes not equal")
			assert.Equal(t, result.Manifest.Subnet, tt.resp.Manifest.Subnet, "manifest subnet not equal")
			assert.True(t, result.Manifest.Orchestrator.Equal(tt.resp.Manifest.Orchestrator), "manifest orchestrator not equal")
		})
	}
}

func TestClient_DeploymentShutdown(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.DeploymentShutdownBehavior
	tests := []struct {
		name    string
		req     node.DeploymentShutdownRequest
		resp    node.DeploymentShutdownResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.DeploymentShutdownRequest{
				ID: "",
			},
			node.DeploymentShutdownResponse{
				OK:    true,
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

			result, err := c.DeploymentShutdown(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_DeploymentNew(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.NewDeploymentBehavior
	tests := []struct {
		name    string
		req     node.NewDeploymentRequest
		resp    node.NewDeploymentResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.NewDeploymentRequest{
				Ensemble: jobtypes.EnsembleConfig{},
			},
			node.NewDeploymentResponse{
				Error:      "",
				Status:     "",
				EnsembleID: "",
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

			result, err := c.DeploymentNew(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_DeploymentUpdate(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.DeploymentUpdateBehavior
	tests := []struct {
		name    string
		req     node.UpdateDeploymentRequest
		resp    node.UpdateDeploymentResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.UpdateDeploymentRequest{
				EnsembleID: "",
				Ensemble:   jobtypes.EnsembleConfig{},
			},
			node.UpdateDeploymentResponse{
				OK:    true,
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

			result, err := c.DeploymentUpdate(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}
