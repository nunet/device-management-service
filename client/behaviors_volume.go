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
