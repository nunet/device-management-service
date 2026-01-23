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

func (c *Client) ProvideCapAnchor(ctx context.Context, req node.CapTokenAnchorRequest, opts ...Option) (node.CapAnchorResponse, error) {
	var response node.CapAnchorResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ProvideCapAnchorBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ProvideCapAnchorBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) RequireCapAnchor(ctx context.Context, req node.CapTokenAnchorRequest, opts ...Option) (node.CapAnchorResponse, error) {
	var response node.CapAnchorResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.RequireCapAnchorBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.RequireCapAnchorBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) RevokeCapAnchor(ctx context.Context, req node.CapTokenAnchorRequest, opts ...Option) (node.CapAnchorResponse, error) {
	var response node.CapAnchorResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.RevokeCapAnchorBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.RevokeCapAnchorBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

// BroadcastCapRevoke broadcasts a capability revocation message
func (c *Client) BroadcastCapRevoke(ctx context.Context, req node.CapTokenAnchorRequest, msgOpts ...Option) ([]node.CapAnchorResponse, error) {
	resp, err := c.BroadcastMessage(
		ctx,
		behaviors.BroadcastRevokeCapBehavior,
		behaviors.BroadcastRevocationTopic,
		req,
		msgOpts...,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", behaviors.BroadcastRevokeCapBehavior, err)
	}

	response := make([]node.CapAnchorResponse, 0, len(resp))

	for _, r := range resp {
		var msg node.CapAnchorResponse
		if err = c.unmarshalResponse(r, &msg); err != nil {
			return nil, err
		}

		response = append(response, msg)
	}

	return response, nil
}
