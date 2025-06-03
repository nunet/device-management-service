package jobs

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/actor"
)

func TestAllocation_handleSubnetAddPeer(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := SubnetAddPeerRequest{
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

	var resp SubnetAddPeerResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}

func TestAllocation_handleSubnetAcceptPeer(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := SubnetAcceptPeerRequest{
		SubnetID: "test-subnet-id",
		PeerID:   "test-peer-id",
		IP:       "192.168.1.10",
	}

	reqBytes, err := json.Marshal(req)
	require.NoError(t, err)

	envelope := createTestEnvelope(t, reqBytes)
	alloc.handleSubnetAcceptPeer(envelope)

	// check actor behavior
	noopActor, ok := alloc.Actor.(*actor.NoopActor)
	require.True(t, ok)
	require.NotNil(t, noopActor)

	sent := noopActor.GetSentMessages()
	require.Len(t, sent, 1)

	var resp SubnetAcceptPeerResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}

func TestAllocation_handleSubnetMapPort(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := SubnetMapPortRequest{
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

	var resp SubnetMapPortResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}

func TestAllocation_handleSubnetDNSAddRecords(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := SubnetDNSAddRecordsRequest{
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

	var resp SubnetDNSAddRecordsResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}

func TestAllocation_handleSubnetUnmapPort(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := SubnetUnmapPortRequest{
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

	var resp SubnetUnmapPortResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}

func TestAllocation_handleSubnetDNSRemoveRecord(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := SubnetDNSRemoveRecordRequest{
		SubnetID:   "test-subnet-id",
		DomainName: "test.local",
	}

	reqBytes, err := json.Marshal(req)
	require.NoError(t, err)

	envelope := createTestEnvelope(t, reqBytes)
	alloc.handleSubnetDNSRemoveRecord(envelope)

	// check actor behavior
	noopActor, ok := alloc.Actor.(*actor.NoopActor)
	require.True(t, ok)
	require.NotNil(t, noopActor)

	sent := noopActor.GetSentMessages()
	require.Len(t, sent, 1)

	var resp SubnetDNSRemoveRecordResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}

func TestAllocation_handleSubnetRemovePeer(t *testing.T) {
	alloc, err := createTestAllocation(t)
	require.NoError(t, err)
	require.NotNil(t, alloc)

	req := SubnetRemovePeerRequest{
		IP:       "192.168.1.10",
		SubnetID: "test-subnet-id",
		PeerID:   "test-peer-id",
	}

	reqBytes, err := json.Marshal(req)
	require.NoError(t, err)

	envelope := createTestEnvelope(t, reqBytes)
	alloc.handleSubnetRemovePeer(envelope)

	// check actor behavior
	noopActor, ok := alloc.Actor.(*actor.NoopActor)
	require.True(t, ok)
	require.NotNil(t, noopActor)

	sent := noopActor.GetSentMessages()
	require.Len(t, sent, 1)

	var resp SubnetRemovePeerResponse
	err = json.Unmarshal(sent[0].Message, &resp)
	require.NoError(t, err)

	require.True(t, resp.OK)
	require.Empty(t, resp.Error)
}
