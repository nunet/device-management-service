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
