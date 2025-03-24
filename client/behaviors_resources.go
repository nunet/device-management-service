package client

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/node"
)

func (c *Client) ResourcesAllocated(ctx context.Context, opts ...Option) (node.ResourcesResponse, error) {
	var response node.ResourcesResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ResourcesAllocatedBehavior,
		nil,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ResourcesAllocatedBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) ResourcesFree(ctx context.Context, opts ...Option) (node.ResourcesResponse, error) {
	var response node.ResourcesResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ResourcesFreeBehavior,
		nil,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ResourcesFreeBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) ResourcesOnboarded(ctx context.Context, opts ...Option) (node.ResourcesResponse, error) {
	var response node.ResourcesResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ResourcesOnboardedBehavior,
		nil,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ResourcesOnboardedBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) HardwareSpec(ctx context.Context, opts ...Option) (node.ResourcesResponse, error) {
	var response node.ResourcesResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.HardwareSpecBehavior,
		nil,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.HardwareSpecBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) HardwareUsage(ctx context.Context, opts ...Option) (node.ResourcesResponse, error) {
	var response node.ResourcesResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.HardwareUsageBehavior,
		nil,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.HardwareUsageBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}
