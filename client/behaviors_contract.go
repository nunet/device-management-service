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

func (c *Client) NewContract(ctx context.Context, req contracts.CreateContractRequest, opts ...Option) (contracts.CreateContractResponse, error) {
	var response contracts.CreateContractResponse

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

func (c *Client) ContractStatus(ctx context.Context, req contracts.ContractStatusRequest, opts ...Option) (contracts.ContractStatusResponse, error) {
	var response contracts.ContractStatusResponse

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

func (c *Client) ApproveLocal(ctx context.Context, req contracts.ContractApproveLocalRequest, opts ...Option) (contracts.ContractApproveLocalResponse, error) {
	var response contracts.ContractApproveLocalResponse

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

func (c *Client) ListIncoming(ctx context.Context, req contracts.ContractListIncomingRequest, opts ...Option) (contracts.ContractListIncomingResponse, error) {
	var response contracts.ContractListIncomingResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ContractListBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ContractListBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) ListTransactions(ctx context.Context, opts ...Option) (contracts.ContractListLocalTransactionsResponse, error) {
	var response contracts.ContractListLocalTransactionsResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ContractListLocalTransactionsBehavior,
		struct{}{},
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ContractListLocalTransactionsBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) ConfirmTransaction(ctx context.Context, req contracts.ContractConfirmLocalTransactionRequest, opts ...Option) (contracts.ContractConfirmLocalTransactionResponse, error) {
	var response contracts.ContractConfirmLocalTransactionResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ContractConfirmLocalTransactionBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ContractConfirmLocalTransactionBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) CollectUsagesAndForwardToPaymentProviders(ctx context.Context, opts ...Option) (contracts.CollectUsagesAndForwardToPaymentProvidersResponse, error) {
	var response contracts.CollectUsagesAndForwardToPaymentProvidersResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ContractUsagesCalculateBehavior,
		struct{}{},
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ContractUsagesCalculateBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) GetPaymentStatus(ctx context.Context, req contracts.ContractPaymentStatusRequest, opts ...Option) (contracts.ContractPaymentStatusResponse, error) {
	var response contracts.ContractPaymentStatusResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ContractPaymentStatusBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ContractPaymentStatusBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) TerminateContract(ctx context.Context, req contracts.ContractTerminationRequest, opts ...Option) (contracts.ContractTerminationResponse, error) {
	var response contracts.ContractTerminationResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ContractTerminationBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ContractTerminationBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) SettleContract(ctx context.Context, req contracts.ContractSettleRequest, opts ...Option) (contracts.ContractSettleResponse, error) {
	var response contracts.ContractSettleResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ContractSettleBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ContractSettleBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) CompleteContract(ctx context.Context, req contracts.ContractCompletionRequest, opts ...Option) (contracts.ContractCompletionResponse, error) {
	var response contracts.ContractCompletionResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ContractCompleteBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ContractCompleteBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) ValidateContract(ctx context.Context, req contracts.ContractValidateRequest, opts ...Option) (contracts.ContractValidateResponse, error) {
	var response contracts.ContractValidateResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.ContractValidationBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.ContractValidationBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}
