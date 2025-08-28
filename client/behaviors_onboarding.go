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

func (c *Client) Onboard(ctx context.Context, req node.OnboardRequest, opts ...Option) (node.OnboardResponse, error) {
	var response node.OnboardResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.OnboardBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.OnboardBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) Offboard(ctx context.Context, req node.OffboardRequest, opts ...Option) (node.OffboardResponse, error) {
	var response node.OffboardResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.OffboardBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.OffboardBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) OnboardStatus(ctx context.Context, opts ...Option) (node.OnboardStatusResponse, error) {
	var response node.OnboardStatusResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.OnboardStatusBehavior,
		nil,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.OnboardStatusBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}
