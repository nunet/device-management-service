package jobs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
)

func TestAllocation_handleSubnetAddPeer(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := behaviors.SubnetAddPeerRequest{
		SubnetID: "test-subnet-id",
		PeerID:   "test-peer-id",
		IP:       "192.168.1.10",
	}

	reqBytes, err := json.Marshal(req)
	require.NoError(t, err)

	envelope := createTestEnvelope(t, reqBytes)
	alloc.handleSubnetAddPeer(envelope)

	// check actor behavior
	noopActor, ok := alloc.Actor.(*actor.NoopActor)
	require.True(t, ok)
	require.NotNil(t, noopActor)

	sent := noopActor.GetSentMessages()
	require.Len(t, sent, 1)

	var resp behaviors.SubnetAddPeerResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}

func TestAllocation_handleSubnetAcceptPeer(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := behaviors.SubnetAcceptPeersRequest{
		SubnetID: "test-subnet-id",
		PartialRoutingTable: map[string]string{
			"test-peer-id": "192.168.1.10",
		},
	}

	reqBytes, err := json.Marshal(req)
	require.NoError(t, err)

	envelope := createTestEnvelope(t, reqBytes)
	alloc.handleSubnetAcceptPeers(envelope)

	// check actor behavior
	noopActor, ok := alloc.Actor.(*actor.NoopActor)
	require.True(t, ok)
	require.NotNil(t, noopActor)

	sent := noopActor.GetSentMessages()
	require.Len(t, sent, 1)

	var resp behaviors.SubnetAcceptPeersResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}

func TestAllocation_handleSubnetMapPort(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := behaviors.SubnetMapPortRequest{
		SubnetID:   "test-subnet-id",
		Protocol:   "tcp",
		SourceIP:   "192.168.1.10",
		SourcePort: "8080",
		DestIP:     "192.168.1.20",
		DestPort:   "9090",
	}

	reqBytes, err := json.Marshal(req)
	require.NoError(t, err)

	envelope := createTestEnvelope(t, reqBytes)
	alloc.handleSubnetMapPort(envelope)

	// check actor behavior
	noopActor, ok := alloc.Actor.(*actor.NoopActor)
	require.True(t, ok)
	require.NotNil(t, noopActor)

	sent := noopActor.GetSentMessages()
	require.Len(t, sent, 1)

	var resp behaviors.SubnetMapPortResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}

func TestAllocation_handleSubnetDNSAddRecords(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := behaviors.SubnetDNSAddRecordsRequest{
		SubnetID: "test-subnet-id",
		Records: map[string]string{
			"test.local":    "192.168.1.10",
			"service.local": "192.168.1.20",
		},
	}

	reqBytes, err := json.Marshal(req)
	require.NoError(t, err)

	envelope := createTestEnvelope(t, reqBytes)
	alloc.handleSubnetDNSAddRecords(envelope)

	// check actor behavior
	noopActor, ok := alloc.Actor.(*actor.NoopActor)
	require.True(t, ok)
	require.NotNil(t, noopActor)

	sent := noopActor.GetSentMessages()
	require.Len(t, sent, 1)

	var resp behaviors.SubnetDNSAddRecordsResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}

func TestAllocation_handleSubnetUnmapPort(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := behaviors.SubnetUnmapPortRequest{
		SubnetID:   "test-subnet-id",
		Protocol:   "tcp",
		SourceIP:   "192.168.1.10",
		SourcePort: "8080",
		DestIP:     "192.168.1.20",
		DestPort:   "9090",
	}

	reqBytes, err := json.Marshal(req)
	require.NoError(t, err)

	envelope := createTestEnvelope(t, reqBytes)
	alloc.handleSubnetUnmapPort(envelope)

	// check actor behavior
	noopActor, ok := alloc.Actor.(*actor.NoopActor)
	require.True(t, ok)
	require.NotNil(t, noopActor)

	sent := noopActor.GetSentMessages()
	require.Len(t, sent, 1)

	var resp behaviors.SubnetUnmapPortResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}

func TestAllocation_handleSubnetDNSRemoveRecord(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := behaviors.SubnetDNSRemoveRecordsRequest{
		SubnetID:    "test-subnet-id",
		DomainNames: []string{"a"},
	}

	reqBytes, err := json.Marshal(req)
	require.NoError(t, err)

	envelope := createTestEnvelope(t, reqBytes)
	alloc.handleSubnetDNSRemoveRecords(envelope)

	// check actor behavior
	noopActor, ok := alloc.Actor.(*actor.NoopActor)
	require.True(t, ok)
	require.NotNil(t, noopActor)

	sent := noopActor.GetSentMessages()
	require.Len(t, sent, 1)

	var resp behaviors.SubnetDNSRemoveRecordsResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}

func TestAllocation_handleSubnetRemovePeers(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := behaviors.SubnetRemovePeersRequest{
		SubnetID:            "test-subnet-id",
		PartialRoutingTable: map[string]string{},
	}

	reqBytes, err := json.Marshal(req)
	require.NoError(t, err)

	envelope := createTestEnvelope(t, reqBytes)
	alloc.handleSubnetRemovePeers(envelope)

	// check actor behavior
	noopActor, ok := alloc.Actor.(*actor.NoopActor)
	require.True(t, ok)
	require.NotNil(t, noopActor)

	sent := noopActor.GetSentMessages()
	require.Len(t, sent, 1)

	var resp behaviors.SubnetRemovePeersResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}
