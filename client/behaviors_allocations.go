package client

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/node"
)

func (c *Client) AllocationsList(
	ctx context.Context,
	opts ...Option,
) (node.AllocationsListResponse, error) {
	var response node.AllocationsListResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.AllocationsListBehavior,
		nil,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.AllocationsListBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}
