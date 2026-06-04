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
	"gitlab.com/nunet/device-management-service/tokenomics/store/transaction"
	"gitlab.com/nunet/device-management-service/types"
)

func TestHandleListLocalTransactions(t *testing.T) {
	t.Parallel()

	node, sActor, _ := newMockNodeWithSender(t, behaviors.ContractListLocalTransactionsBehavior)

	// define additional with some params such as requestor repeating
	require.NoError(t, node.transactionStore.Upsert(transaction.Transaction{
		UniqueID: "tx-001", ContractDID: "did:key:contract-1",
		ToAddress: []types.PaymentAddressInfo{
			{Blockchain: "ETHEREUM", RequesterAddr: "0xREQ1", ProviderAddr: "0xPROV1"},
		},
		Metadata: map[string]interface{}{
			"fee_type": "orchestration",
		},
	}))
	require.NoError(t, node.transactionStore.Upsert(transaction.Transaction{
		UniqueID: "tx-002", Status: "paid", // one is paid
		ContractDID: "did:key:contract-2",
		ToAddress: []types.PaymentAddressInfo{
			{Blockchain: "ETHEREUM", RequesterAddr: "0xREQ1", ProviderAddr: "0xPROV2"},
		},
	}))
	require.NoError(t, node.transactionStore.Upsert(transaction.Transaction{
		UniqueID: "tx-003", ContractDID: "did:key:contract-1",
		ToAddress: []types.PaymentAddressInfo{
			{Blockchain: "ETHEREUM", RequesterAddr: "0xREQ2", ProviderAddr: "0xPROV1"},
		},
	}))
	require.NoError(t, node.transactionStore.Upsert(transaction.Transaction{
		UniqueID: "tx-004", ContractDID: "did:key:contract-1",
		ToAddress: []types.PaymentAddressInfo{
			{Blockchain: "ETHEREUM", RequesterAddr: "0xREQ2", ProviderAddr: "0xPROV2"},
		},
	}))
	require.NoError(t, node.transactionStore.Upsert(transaction.Transaction{
		UniqueID: "tx-005", ContractDID: "did:key:contract-2",
		ToAddress: []types.PaymentAddressInfo{
			{Blockchain: "CARDANO", RequesterAddr: "0xREQ3", ProviderAddr: "0xPROV3"},
		},
	}))
	require.NoError(t, node.transactionStore.Upsert(transaction.Transaction{
		UniqueID: "tx-006", ContractDID: "did:key:contract-2",
		ToAddress: []types.PaymentAddressInfo{
			{Blockchain: "CARDANO", RequesterAddr: "0xREQ1", ProviderAddr: "0xPROV3"},
		},
	}))

	t.Run("filters by fields and metadata", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.ContractListLocalTransactionsBehavior,
			contracts.ContractListLocalTransactionsRequest{
				Status:      []string{"unpaid"},
				ContractDID: "did:key:contract-1",
				Metadata: map[string]string{
					"fee_type": "orchestration",
				},
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		require.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp contracts.ContractListLocalTransactionsResponse
		require.NoError(t, json.Unmarshal(reply.Message, &resp))
		require.Empty(t, resp.Error)
		require.Len(t, resp.Transactions, 1)
		assert.Equal(t, "tx-001", resp.Transactions[0].UniqueID)
		assert.Equal(t, 1, resp.Total)
		assert.False(t, resp.HasMore)
	})

	t.Run("paginates sorted results", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.ContractListLocalTransactionsBehavior,
			contracts.ContractListLocalTransactionsRequest{
				SortBy: "-unique_id",
				Limit:  2,
				Offset: 0,
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		require.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		t.Logf("reply: %s", string(reply.Message))

		var resp contracts.ContractListLocalTransactionsResponse
		require.NoError(t, json.Unmarshal(reply.Message, &resp))
		require.Empty(t, resp.Error)
		require.Len(t, resp.Transactions, 2)
		assert.Equal(t, "tx-006", resp.Transactions[0].UniqueID)
		assert.Equal(t, "tx-005", resp.Transactions[1].UniqueID)
		assert.Equal(t, 6, resp.Total)
		assert.True(t, resp.HasMore)
		assert.Equal(t, 2, resp.NextOffset)
	})

	t.Run("sorts by created_at desc", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.ContractListLocalTransactionsBehavior,
			contracts.ContractListLocalTransactionsRequest{
				SortBy: "-created_at",
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		require.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp contracts.ContractListLocalTransactionsResponse
		require.NoError(t, json.Unmarshal(reply.Message, &resp))
		require.Empty(t, resp.Error)
		require.Len(t, resp.Transactions, 6)
		assert.Equal(t, "tx-006", resp.Transactions[0].UniqueID)
		assert.Equal(t, "tx-001", resp.Transactions[5].UniqueID)
		assert.NotEqual(t, int64(0), resp.Transactions[0].CreatedAt)
	})

	t.Run("filters for specific one", func(t *testing.T) {
		t.Parallel()

		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.ContractListLocalTransactionsBehavior,
			contracts.ContractListLocalTransactionsRequest{
				Blockchain:  "Cardano",
				FromAddress: "0xREQ1",
				ToAddress:   "0xPROV3",
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		require.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp contracts.ContractListLocalTransactionsResponse
		require.NoError(t, json.Unmarshal(reply.Message, &resp))
		require.Empty(t, resp.Error)
		require.Len(t, resp.Transactions, 1)
		assert.Equal(t, "tx-006", resp.Transactions[0].UniqueID)
	})

	t.Run("for multiple", func(t *testing.T) {
		t.Parallel()

		// filter on eth from req1
		msg, err := actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.ContractListLocalTransactionsBehavior,
			contracts.ContractListLocalTransactionsRequest{
				Blockchain:  "ethereum", // expecting 2 records with eth+req1 (paid/unpaid)
				FromAddress: "0xREQ1",
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err := sActor.Invoke(msg)
		require.NoError(t, err)

		reply := <-replyChan
		defer reply.Discard()

		var resp contracts.ContractListLocalTransactionsResponse
		require.NoError(t, json.Unmarshal(reply.Message, &resp))
		require.Empty(t, resp.Error)
		require.Len(t, resp.Transactions, 2)

		// same filter but with status unpaid should be only one
		msg, err = actor.Message(
			sActor.Handle(),
			node.actor.Handle(),
			behaviors.ContractListLocalTransactionsBehavior,
			contracts.ContractListLocalTransactionsRequest{
				Blockchain:  "ethereum",
				FromAddress: "0xREQ1",
				Status:      []string{"unpaid"}, // expecting 1 records with eth+req1+unpaid
			},
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		require.NoError(t, err)

		replyChan, err = sActor.Invoke(msg)
		require.NoError(t, err)

		reply = <-replyChan
		defer reply.Discard()

		var nResp contracts.ContractListLocalTransactionsResponse
		require.NoError(t, json.Unmarshal(reply.Message, &nResp))
		require.Empty(t, nResp.Error)
		require.Len(t, nResp.Transactions, 1)
	})
}

func TestMatchesTransactionAddressFilters(t *testing.T) {
	t.Parallel()

	tx := &transaction.Transaction{
		ToAddress: []types.PaymentAddressInfo{
			{
				Blockchain:    "Ethereum",
				RequesterAddr: "0xREQ111",
				ProviderAddr:  "0xPROV111",
			},
			{
				Blockchain:    "Polygon",
				RequesterAddr: "0xREQ222",
				ProviderAddr:  "0xPROV222",
			},
		},
	}

	t.Run("single blockchain filter matches regardless of addresses", func(t *testing.T) {
		t.Parallel()
		assert.True(t, matchesTransactionAddressFilters(tx, contracts.ContractListLocalTransactionsRequest{
			Blockchain: "polygon",
		}))
	})

	t.Run("multiple filters are ANDed across provided params", func(t *testing.T) {
		t.Parallel()
		assert.True(t, matchesTransactionAddressFilters(tx, contracts.ContractListLocalTransactionsRequest{
			Blockchain:  "ethereum",
			FromAddress: "0xREQ111",
		}))
	})

	t.Run("fail when one param is missing", func(t *testing.T) {
		t.Parallel()
		assert.False(t, matchesTransactionAddressFilters(tx, contracts.ContractListLocalTransactionsRequest{
			Blockchain:  "ethereum",
			FromAddress: "0xREQ999",
		}))
	})

	t.Run("AND allows values to come from different address entries", func(t *testing.T) {
		t.Parallel()
		assert.True(t, matchesTransactionAddressFilters(tx, contracts.ContractListLocalTransactionsRequest{
			Blockchain:  "ethereum",
			FromAddress: "0xREQ222",
		}))
	})
}

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
