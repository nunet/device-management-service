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

func (c *Client) PeersSelf(ctx context.Context, msgOpts ...Option) (node.PeerAddrInfoResponse, error) {
	var response node.PeerAddrInfoResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.PeerAddrInfoBehavior,
		nil,
		msgOpts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.PeerAddrInfoBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) PeersList(ctx context.Context, msgOpts ...Option) (node.PeersListResponse, error) {
	var response node.PeersListResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.PeersListBehavior,
		nil,
		msgOpts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.PeersListBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) PeersListFromDHT(ctx context.Context, msgOpts ...Option) (node.PeerDHTResponse, error) {
	var response node.PeerDHTResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.PeerDHTBehavior,
		nil,
		msgOpts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.PeerDHTBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) PeerPing(ctx context.Context, req node.PingRequest, msgOpts ...Option) (node.PingResponse, error) {
	var response node.PingResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.PeerPingBehavior,
		req,
		msgOpts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.PeerPingBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) PeerConnect(ctx context.Context, req node.PeerConnectRequest, msgOpts ...Option) (node.PeerConnectResponse, error) {
	var response node.PeerConnectResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.PeerConnectBehavior,
		req,
		msgOpts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.PeerConnectBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) PeerScore(ctx context.Context, msgOpts ...Option) (node.PeerScoreResponse, error) {
	var response node.PeerScoreResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.PeerScoreBehavior,
		nil,
		msgOpts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.PeerScoreBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}
