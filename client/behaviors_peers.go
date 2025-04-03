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
