package node

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/types"
)

func TestHandleListIncomingContractsAggregatesRemoteHosts(t *testing.T) {
	t.Skip()
	t.Parallel()

	netSubstrate := network.NewSubstrate()

	// Service provider node under test.
	spNode, spRootTrust, spRootDID := newMockNode(t, netSubstrate)
	t.Cleanup(func() {
		_ = spNode.actor.Stop()
	})
	t.Logf("SP root DID: %s", spRootDID)
	t.Logf("SP actor DID: %s", spNode.actor.Handle().DID)
	t.Logf("SP node rootCap DID: %s", spNode.rootCap.DID())

	// Contract host node containing the authoritative contract.
	chNode, chRootTrust, chRootDID := newMockNode(t, netSubstrate)
	t.Cleanup(func() {
		_ = chNode.actor.Stop()
	})
	t.Logf("CH root DID: %s", chRootDID)
	t.Logf("CH actor DID: %s", chNode.actor.Handle().DID)
	t.Logf("CH node rootCap DID: %s", chNode.rootCap.DID())

	// Allow the SP root actor to invoke list incoming on the CH root actor and vice versa.
	actor.AllowReciprocal(
		t,
		spNode.actor.Security().Capability(),
		spRootTrust,
		spRootDID,
		chNode.actor.Handle().DID,
		"/",
	)
	actor.AllowReciprocal(
		t,
		chNode.actor.Security().Capability(),
		chRootTrust,
		chRootDID,
		spNode.actor.Handle().DID,
		"/",
	)

	contractID := "contract-test"

	authoritativeContract := &contracts.Contract{
		ContractDID:        contractID,
		SolutionEnablerDID: chNode.actor.Handle().DID,
		ContractParticipants: contracts.ContractParticipants{
			Provider:  spNode.actor.Handle().DID,
			Requestor: did.DID{URI: "did:nunet:requestor"},
		},
		CurrentState: contracts.ContractAccepted,
		Transitions: []contracts.StateTransition{
			{
				FromState:   contracts.ContractDraft,
				ToState:     contracts.ContractAccepted,
				Event:       contracts.EventAccepted,
				Timestamp:   time.Now(),
				InitiatedBy: chNode.actor.Handle().DID,
			},
		},
	}
	require.NoError(t, chNode.contractStore.Upsert(authoritativeContract))

	staleContract := *authoritativeContract
	staleContract.CurrentState = contracts.ContractDraft
	staleContract.Transitions = nil
	require.NoError(t, spNode.contractStore.Upsert(&staleContract))

	req := contracts.ContractListIncomingRequest{Role: contracts.ContractRoleProvider}

	// Sanity check: SP can invoke CH directly.
	directMsg, err := actor.Message(
		spNode.actor.Handle(),
		chNode.actor.Handle(),
		behaviors.ContractListBehavior,
		req,
	)
	require.NoError(t, err)

	directReplyCh, err := spNode.actor.Invoke(directMsg)
	require.NoError(t, err)

	select {
	case reply := <-directReplyCh:
		defer reply.Discard()

		var resp contracts.ContractListIncomingResponse
		require.NoError(t, json.Unmarshal(reply.Message, &resp))
		require.Len(t, resp.Contracts, 1)
		require.Equal(t, contracts.ContractAccepted, resp.Contracts[0].CurrentState)

	case <-time.After(5 * time.Second):
		t.Fatal("direct invocation to CH timed out")
	}

	replyBehavior := fmt.Sprintf("/test/reply/%d", time.Now().UnixNano())
	replyCh := make(chan actor.Envelope, 1)
	require.NoError(t, spNode.actor.AddBehavior(
		replyBehavior,
		func(msg actor.Envelope) {
			replyCh <- msg
		},
		actor.WithBehaviorCapability(),
	))
	t.Cleanup(func() {
		spNode.actor.RemoveBehavior(replyBehavior)
	})

	msg, err := actor.Message(
		spNode.actor.Handle(),
		spNode.actor.Handle(),
		behaviors.ContractListBehavior,
		req,
		actor.WithMessageReplyTo(replyBehavior),
		actor.WithMessageSignature(
			spNode.actor.Security(),
			[]actor.Capability{actor.Capability(behaviors.ContractListBehavior)},
			[]actor.Capability{actor.Capability(replyBehavior)},
		),
	)
	require.NoError(t, err)

	require.NoError(t, spNode.actor.Send(msg))

	select {
	case reply := <-replyCh:
		defer reply.Discard()

		var resp contracts.ContractListIncomingResponse
		require.NoError(t, json.Unmarshal(reply.Message, &resp))
		require.Empty(t, resp.Error)
		require.Len(t, resp.Contracts, 1)

		contract := resp.Contracts[0]
		assert.Equal(t, contractID, contract.ContractDID)
		assert.Equal(t, contracts.ContractAccepted, contract.CurrentState)
		assert.Equal(t, authoritativeContract.ContractParticipants.Provider, contract.ContractParticipants.Provider)
		assert.Equal(t, authoritativeContract.ContractParticipants.Requestor, contract.ContractParticipants.Requestor)

	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for list incoming reply")
	}
}

func TestHandleContractCreateForwardAndStoreLocally(t *testing.T) {
	t.Skip()
	t.Parallel()

	netSubstrate := network.NewSubstrate()

	// Setup service provider node (under test) and contract host node.
	spNode, spRootTrust, spRootDID := newMockNode(t, netSubstrate)
	t.Cleanup(func() { _ = spNode.actor.Stop() })

	chNode, chRootTrust, chRootDID := newMockNode(t, netSubstrate)
	t.Cleanup(func() { _ = chNode.actor.Stop() })

	// Allow SP to invoke create-contract on contract host and vice versa.
	actor.AllowReciprocal(
		t,
		spNode.actor.Security().Capability(),
		spRootTrust,
		spRootDID,
		chNode.actor.Handle().DID,
		behaviors.ContractCreateBehavior,
	)
	actor.AllowReciprocal(
		t,
		chNode.actor.Security().Capability(),
		chRootTrust,
		chRootDID,
		spNode.actor.Handle().DID,
		behaviors.ContractCreateBehavior,
	)

	req := contracts.CreateContractRequest{
		SolutionEnablerDID:  chNode.actor.Handle().DID,
		PaymentValidatorDID: chNode.actor.Handle().DID,
		ResourceConfiguration: types.Resources{
			CPU: types.CPU{Cores: 1},
		},
		ContractParticipants: contracts.ContractParticipants{
			Provider:  chNode.actor.Handle().DID,
			Requestor: spNode.actor.Handle().DID,
		},
		PaymentDetails: contracts.PaymentDetails{
			PaymentType:      contracts.BlockchainMethod,
			FeePerAllocation: "10",
		},
	}

	replyBehavior := fmt.Sprintf("/test/reply/%d", time.Now().UnixNano())
	replyCh := make(chan actor.Envelope, 1)
	require.NoError(t, spNode.actor.AddBehavior(
		replyBehavior,
		func(msg actor.Envelope) {
			replyCh <- msg
		},
		actor.WithBehaviorCapability(),
	))
	t.Cleanup(func() {
		spNode.actor.RemoveBehavior(replyBehavior)
	})

	msg, err := actor.Message(
		spNode.actor.Handle(),
		spNode.actor.Handle(),
		behaviors.ContractCreateBehavior,
		req,
		actor.WithMessageReplyTo(replyBehavior),
		actor.WithMessageSignature(
			spNode.actor.Security(),
			[]actor.Capability{actor.Capability(behaviors.ContractCreateBehavior)},
			[]actor.Capability{actor.Capability(replyBehavior)},
		),
	)
	require.NoError(t, err)

	require.NoError(t, spNode.actor.Send(msg))

	var resp contracts.CreateContractResponse
	select {
	case reply := <-replyCh:
		defer reply.Discard()
		require.NoError(t, json.Unmarshal(reply.Message, &resp))
		require.Empty(t, resp.Error)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for create contract response")
	}

	require.NotEmpty(t, resp.ContractDID)

	hostContract, err := chNode.contractStore.GetContract(resp.ContractDID)
	require.NoError(t, err)
	assert.Equal(t, resp.ContractDID, hostContract.ContractDID)

	localContract, err := spNode.contractStore.GetContract(resp.ContractDID)
	require.NoError(t, err)
	assert.Equal(t, resp.ContractDID, localContract.ContractDID)
	assert.Equal(t, resp.ContractRequest.SolutionEnablerDID.String(), localContract.SolutionEnablerDID.String())
}
