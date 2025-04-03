package client

import (
	"context"
	"fmt"

	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
)

func (c *Client) SubnetCreate(ctx context.Context, req orchestrator.SubnetCreateRequest, opts ...Option) (orchestrator.SubnetCreateResponse, error) {
	var response orchestrator.SubnetCreateResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.SubnetCreateBehavior.Static,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.SubnetCreateBehavior.Static, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) SubnetDestroy(ctx context.Context, req orchestrator.SubnetDestroyRequest, opts ...Option) (orchestrator.SubnetDestroyResponse, error) {
	var response orchestrator.SubnetDestroyResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.SubnetDestroyBehavior.Static,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.SubnetDestroyBehavior.Static, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) SubnetJoin(ctx context.Context, req orchestrator.SubnetJoinRequest, opts ...Option) (orchestrator.SubnetJoinResponse, error) {
	var response orchestrator.SubnetJoinResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.SubnetJoinBehavior.Static,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.SubnetJoinBehavior.Static, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) SubnetAddPeer(ctx context.Context, req jobs.SubnetAddPeerRequest, opts ...Option) (jobs.SubnetAddPeerResponse, error) {
	var response jobs.SubnetAddPeerResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.SubnetAddPeerBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.SubnetAddPeerBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) SubnetRemovePeer(ctx context.Context, req jobs.SubnetRemovePeerRequest, opts ...Option) (jobs.SubnetRemovePeerResponse, error) {
	var response jobs.SubnetRemovePeerResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.SubnetRemovePeerBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.SubnetRemovePeerBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) SubnetAcceptPeer(ctx context.Context, req jobs.SubnetAcceptPeerRequest, opts ...Option) (jobs.SubnetAcceptPeerResponse, error) {
	var response jobs.SubnetAcceptPeerResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.SubnetAcceptPeerBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.SubnetAcceptPeerBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) SubnetMapPort(ctx context.Context, req jobs.SubnetMapPortRequest, opts ...Option) (jobs.SubnetMapPortResponse, error) {
	var response jobs.SubnetMapPortResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.SubnetMapPortBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.SubnetMapPortBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) SubnetUnmapPort(ctx context.Context, req jobs.SubnetUnmapPortRequest, opts ...Option) (jobs.SubnetUnmapPortResponse, error) {
	var response jobs.SubnetUnmapPortResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.SubnetUnmapPortBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.SubnetUnmapPortBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) SubnetDNSAddRecords(ctx context.Context, req jobs.SubnetDNSAddRecordsRequest, opts ...Option) (jobs.SubnetDNSAddRecordsResponse, error) {
	var response jobs.SubnetDNSAddRecordsResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.SubnetDNSAddRecordsBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.SubnetDNSAddRecordsBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}

func (c *Client) SubnetDNSRemoveRecord(ctx context.Context, req jobs.SubnetDNSRemoveRecordRequest, opts ...Option) (jobs.SubnetDNSRemoveRecordResponse, error) {
	var response jobs.SubnetDNSRemoveRecordResponse

	resp, err := c.InvokeBehavior(
		ctx,
		behaviors.SubnetDNSRemoveRecordBehavior,
		req,
		opts...,
	)
	if err != nil {
		return response, fmt.Errorf("%s: %w", behaviors.SubnetDNSRemoveRecordBehavior, err)
	}

	err = c.unmarshalResponse(resp, &response)
	return response, err
}
