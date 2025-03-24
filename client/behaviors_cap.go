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
