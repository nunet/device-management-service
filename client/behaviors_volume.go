package client

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/node"
)

func (c *Client) CreateVolume(ctx context.Context, req node.CreateVolumeRequest, opts ...Option) (node.CreateVolumeResponse, error) {
	var response node.CreateVolumeResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.VolumeCreateBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.VolumeCreateBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) DeleteVolume(ctx context.Context, req node.DeleteVolumeRequest, opts ...Option) (node.DeleteVolumeResponse, error) {
	var response node.DeleteVolumeResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.VolumeDeleteBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.VolumeDeleteBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) StartVolume(ctx context.Context, req node.StartVolumeRequest, opts ...Option) (node.StartVolumeResponse, error) {
	var response node.StartVolumeResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.VolumeStartBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.VolumeStartBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}
