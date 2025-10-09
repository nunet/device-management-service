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
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/node"
)

func (c *Client) ResourcesAllocated(ctx context.Context, opts ...Option) (node.ResourcesResponse, error) {
	var response node.ResourcesResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ResourcesAllocatedBehavior,
		nil,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ResourcesAllocatedBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) ResourcesFree(ctx context.Context, opts ...Option) (node.ResourcesResponse, error) {
	var response node.ResourcesResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ResourcesFreeBehavior,
		nil,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ResourcesFreeBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) ResourcesOnboarded(ctx context.Context, opts ...Option) (node.ResourcesResponse, error) {
	var response node.ResourcesResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ResourcesOnboardedBehavior,
		nil,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ResourcesOnboardedBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) HardwareSpec(ctx context.Context, opts ...Option) (node.ResourcesResponse, error) {
	var response node.ResourcesResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.HardwareSpecBehavior,
		nil,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.HardwareSpecBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) HardwareUsage(ctx context.Context, opts ...Option) (node.ResourcesResponse, error) {
	var response node.ResourcesResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.HardwareUsageBehavior,
		nil,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.HardwareUsageBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}
