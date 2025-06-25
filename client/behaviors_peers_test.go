package client_test

import (
	"context"
	"testing"

	kbucket "github.com/libp2p/go-libp2p-kbucket"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/network"
)

func TestClient_PeersSelf(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.PeerAddrInfoBehavior
	tests := []struct {
		name    string
		resp    node.PeerAddrInfoResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.PeerAddrInfoResponse{
				ID:      "",
				Address: "",
			},
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				assert.Empty(t, envelope.Message)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.PeersSelf(context.Background(), tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_PeersList(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.PeersListBehavior
	tests := []struct {
		name    string
		resp    node.PeersListResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.PeersListResponse{
				Peers: []peer.ID{},
			},
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				assert.Empty(t, envelope.Message)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.PeersList(context.Background(), tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_PeersListFromDHT(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.PeerDHTBehavior
	tests := []struct {
		name    string
		resp    node.PeerDHTResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.PeerDHTResponse{
				Peers: []kbucket.PeerInfo{},
			},
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				assert.Empty(t, envelope.Message)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.PeersListFromDHT(context.Background(), tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_PeerPing(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.PeerPingBehavior
	tests := []struct {
		name    string
		req     node.PingRequest
		resp    node.PingResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.PingRequest{
				Host: "",
			},
			node.PingResponse{
				Error: "",
				RTT:   0,
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

			result, err := c.PeerPing(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_PeerConnect(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.PeerConnectBehavior
	tests := []struct {
		name    string
		req     node.PeerConnectRequest
		resp    node.PeerConnectResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.PeerConnectRequest{
				Address: "",
			},
			node.PeerConnectResponse{
				Status: "",
				Error:  "",
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

			result, err := c.PeerConnect(context.Background(), tt.req, tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}

func TestClient_PeerScore(t *testing.T) {
	expectedPath := client.ActorInvokeEndpoint
	expectedBehavior := behaviors.PeerScoreBehavior
	tests := []struct {
		name    string
		resp    node.PeerScoreResponse
		opts    []client.Option
		wantErr bool
	}{
		{
			"success",
			node.PeerScoreResponse{
				Score: map[string]*network.PeerScoreSnapshot{},
			},
			nil,
			false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, _, err := makeMockBehaviorClient(t, expectedPath, func(t *testing.T, envelope *actor.Envelope) (int, any) {
				assert.Equal(t, envelope.Behavior, expectedBehavior)
				assert.Empty(t, envelope.Message)
				return 200, tt.resp
			})
			assert.NoError(t, err, "create client")

			result, err := c.PeerScore(context.Background(), tt.opts...)
			if tt.wantErr {
				assert.Error(t, err)
			}
			assert.NoError(t, err)
			assert.Equal(t, result, tt.resp)
		})
	}
}
