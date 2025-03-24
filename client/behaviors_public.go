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
