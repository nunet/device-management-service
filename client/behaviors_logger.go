package client

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/node"
)

func (c *Client) LoggerConfig(ctx context.Context, req node.LoggerConfigRequest, opts ...Option) (node.LoggerConfigResponse, error) {
	var response node.LoggerConfigResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.LoggerConfigBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.LoggerConfigBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}
