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

// Hello sends a hello message
func (c *Client) Hello(ctx context.Context, msgOpts ...Option) (node.HelloResponse, error) {
	var response node.HelloResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.PublicHelloBehavior,
		nil,
		msgOpts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.PublicStatusBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

// BroadcastHello broadcasts a hello message to a topic
func (c *Client) BroadcastHello(ctx context.Context, msgOpts ...Option) ([]node.HelloResponse, error) {
	resp, err := c.BroadcastMessage(
		ctx,
		behaviors.BroadcastHelloBehavior,
		behaviors.BroadcastHelloTopic,
		nil,
		msgOpts...,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", behaviors.BroadcastHelloBehavior, err)
	}

	response := make([]node.HelloResponse, 0, len(resp))

	for _, r := range resp {
		var msg node.HelloResponse
		if err = c.unmarshalResponse(r, &msg); err != nil {
			return nil, err
		}

		response = append(response, msg)
	}

	return response, nil
}

func (c *Client) Status(ctx context.Context, msgOpts ...Option) (node.PublicStatusResponse, error) {
	var response node.PublicStatusResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.PublicStatusBehavior,
		nil,
		msgOpts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.PublicStatusBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) Discovery(ctx context.Context, msgOpts ...Option) (node.DiscoveryStatusResponse, error) {
	var response node.DiscoveryStatusResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.StatusDiscoveryBehavior,
		nil,
		msgOpts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.StatusDiscoveryBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) DiscoveryBroadcast(ctx context.Context, msgOpts ...Option) ([]node.DiscoveryStatusResponse, error) {
	resp, err := c.BroadcastMessage(
		ctx,
		behaviors.BroadcastStatusDiscoveryBehavior,
		behaviors.BroadcastStatusDiscoveryTopic,
		nil,
		msgOpts...,
	)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", behaviors.BroadcastStatusDiscoveryBehavior, err)
	}

	response := make([]node.DiscoveryStatusResponse, 0, len(resp))

	for _, r := range resp {
		var msg node.DiscoveryStatusResponse
		if err = c.unmarshalResponse(r, &msg); err != nil {
			return nil, err
		}

		response = append(response, msg)
	}
	return response, err
}
