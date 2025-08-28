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
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
)

func (c *Client) NewContract(ctx context.Context, req contracts.CreateContractRequestBehaviour, opts ...Option) (contracts.CreateContractResponseBehaviour, error) {
	var response contracts.CreateContractResponseBehaviour

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ContractCreateBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ContractCreateBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) ContractStatus(ctx context.Context, req contracts.ContractStatusRequestBehaviour, opts ...Option) (contracts.ContractStatusResponseBehaviour, error) {
	var response contracts.ContractStatusResponseBehaviour

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ContractStatusBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ContractStatusBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) ApproveLocal(ctx context.Context, req contracts.ContractApproveLocalRequestBehaviour, opts ...Option) (contracts.ContractApproveLocalResponseBehaviour, error) {
	var response contracts.ContractApproveLocalResponseBehaviour

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ContractApproveLocalBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ContractApproveLocalBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) ListIncoming(ctx context.Context, opts ...Option) (contracts.ContractListIncomingResponseBehaviour, error) {
	var response contracts.ContractListIncomingResponseBehaviour

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ContractListIncomingBehavior,
		struct{}{},
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ContractListIncomingBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}
