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

func (c *Client) CapList(ctx context.Context, req node.CapListRequest, opts ...Option) (node.CapListResponse, error) {
	var response node.CapListResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.CapListBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.CapListBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) CapAnchor(ctx context.Context, req node.CapAnchorRequest, opts ...Option) (node.CapAnchorResponse, error) {
	var response node.CapAnchorResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.CapAnchorBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.CapAnchorBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}
