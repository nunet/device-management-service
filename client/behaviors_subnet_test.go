package client_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
)

func TestClient_SubnetCreate(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.SubnetCreateBehavior.Static
	tests := []struct {
		name    string
		req     orchestrator.SubnetCreateRequest
		resp    orchestrator.SubnetCreateResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			orchestrator.SubnetCreateRequest{
				SubnetID:     "",
				IP:           "",
				RoutingTable: map[string]string{},
			},
			orchestrator.SubnetCreateResponse{
				Error: "",
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.SubnetCreate(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_SubnetDestroy(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.SubnetDestroyBehavior.Static
	tests := []struct {
		name    string
		req     orchestrator.SubnetDestroyRequest
		resp    orchestrator.SubnetDestroyResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			orchestrator.SubnetDestroyRequest{
				SubnetID: "",
			},
			orchestrator.SubnetDestroyResponse{
				Error: "",
				OK:    false,
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.SubnetDestroy(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_SubnetJoin(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.SubnetJoinBehavior.Static
	tests := []struct {
		name    string
		req     orchestrator.SubnetJoinRequest
		resp    orchestrator.SubnetJoinResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			orchestrator.SubnetJoinRequest{
				SubnetID: "",
				PeerID:   "",
				IP:       "",
				Records:  map[string]string{},
			},
			orchestrator.SubnetJoinResponse{
				Error: "",
				OK:    false,
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.SubnetJoin(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_SubnetAddPeer(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.SubnetAddPeerBehavior
	tests := []struct {
		name    string
		req     jobs.SubnetAddPeerRequest
		resp    jobs.SubnetAddPeerResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			jobs.SubnetAddPeerRequest{
				SubnetID: "",
				PeerID:   "",
				IP:       "",
			},
			jobs.SubnetAddPeerResponse{
				OK:    false,
				Error: "",
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.SubnetAddPeer(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_SubnetRemovePeer(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.SubnetRemovePeerBehavior
	tests := []struct {
		name    string
		req     jobs.SubnetRemovePeerRequest
		resp    jobs.SubnetRemovePeerResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			jobs.SubnetRemovePeerRequest{
				IP:       "",
				SubnetID: "",
				PeerID:   "",
			},
			jobs.SubnetRemovePeerResponse{
				Error: "",
				OK:    false,
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.SubnetRemovePeer(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_SubnetAcceptPeer(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.SubnetAcceptPeerBehavior
	tests := []struct {
		name    string
		req     jobs.SubnetAcceptPeerRequest
		resp    jobs.SubnetAcceptPeerResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			jobs.SubnetAcceptPeerRequest{
				SubnetID: "",
				PeerID:   "",
				IP:       "",
			},
			jobs.SubnetAcceptPeerResponse{
				Error: "",
				OK:    false,
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.SubnetAcceptPeer(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_SubnetMapPort(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.SubnetMapPortBehavior
	tests := []struct {
		name    string
		req     jobs.SubnetMapPortRequest
		resp    jobs.SubnetMapPortResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			jobs.SubnetMapPortRequest{},
			jobs.SubnetMapPortResponse{
				Error: "",
				OK:    false,
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.SubnetMapPort(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_SubnetUnmapPort(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.SubnetUnmapPortBehavior
	tests := []struct {
		name    string
		req     jobs.SubnetUnmapPortRequest
		resp    jobs.SubnetUnmapPortResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			jobs.SubnetUnmapPortRequest{
				SubnetID:   "",
				Protocol:   "",
				SourceIP:   "",
				SourcePort: "",
				DestIP:     "",
				DestPort:   "",
			},
			jobs.SubnetUnmapPortResponse{
				Error: "",
				OK:    false,
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.SubnetUnmapPort(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_SubnetDNSAddRecords(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.SubnetDNSAddRecordsBehavior
	tests := []struct {
		name    string
		req     jobs.SubnetDNSAddRecordsRequest
		resp    jobs.SubnetDNSAddRecordsResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			jobs.SubnetDNSAddRecordsRequest{
				Records:  map[string]string{},
				SubnetID: "",
			},
			jobs.SubnetDNSAddRecordsResponse{
				Error: "",
				OK:    false,
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.SubnetDNSAddRecords(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_SubnetDNSRemoveRecord(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.SubnetDNSRemoveRecordBehavior
	tests := []struct {
		name    string
		req     jobs.SubnetDNSRemoveRecordRequest
		resp    jobs.SubnetDNSRemoveRecordResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			jobs.SubnetDNSRemoveRecordRequest{
				DomainName: "",
				SubnetID:   "",
			},
			jobs.SubnetDNSRemoveRecordResponse{
				Error: "",
				OK:    false,
			},
			nil,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.SubnetDNSRemoveRecord(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}
