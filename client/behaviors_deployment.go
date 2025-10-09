// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package client

import (
	"context"
	"encoding/json"
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/node"
)

func (c *Client) DeploymentList(
	ctx context.Context, req node.DeploymentListRequest,
	opts ...Option,
) (node.DeploymentListResponse, error) {
	var response node.DeploymentListResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.DeploymentListBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response,
			fmt.Errorf("%s: %w", behaviors.DeploymentListBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) DeploymentStatus(ctx context.Context, req node.DeploymentStatusRequest, opts ...Option) (node.DeploymentStatusResponse, error) {
	var response node.DeploymentStatusResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.DeploymentStatusBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.DeploymentStatusBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) DeploymentLogs(ctx context.Context, req node.DeploymentLogsRequest, opts ...Option) (node.DeploymentLogsResponse, error) {
	var response node.DeploymentLogsResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.DeploymentLogsBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.DeploymentLogsBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) DeploymentManifest(ctx context.Context, req node.DeploymentManifestRequest, opts ...Option) (node.DeploymentManifestResponse, error) {
	var response node.DeploymentManifestResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.DeploymentManifestBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.DeploymentManifestBehavior, err)
	}

	// Unmarshal response
	if err = json.Unmarshal(resp.Message, &response); err != nil {
		return response, fmt.Errorf("unmarshal response: %w", err)
	}

	return response, nil
}

func (c *Client) DeploymentShutdown(ctx context.Context, req node.DeploymentShutdownRequest, opts ...Option) (node.DeploymentShutdownResponse, error) {
	var response node.DeploymentShutdownResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.DeploymentShutdownBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.DeploymentShutdownBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) DeploymentNew(ctx context.Context, req node.NewDeploymentRequest, opts ...Option) (node.NewDeploymentResponse, error) {
	var response node.NewDeploymentResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.NewDeploymentBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.NewDeploymentBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) DeploymentUpdate(
	ctx context.Context, req node.UpdateDeploymentRequest,
	opts ...Option,
) (node.UpdateDeploymentResponse, error) {
	var response node.UpdateDeploymentResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.DeploymentUpdateBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.DeploymentUpdateBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) DeploymentPrune(ctx context.Context, req node.DeploymentPruneRequest, opts ...Option) (node.DeploymentPruneResponse, error) {
	var response node.DeploymentPruneResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.DeploymentPruneBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.DeploymentPruneBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) DeploymentDelete(ctx context.Context, req node.DeploymentDeleteRequest, opts ...Option) (node.DeploymentDeleteResponse, error) {
	var response node.DeploymentDeleteResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.DeploymentDeleteBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.DeploymentDeleteBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}
