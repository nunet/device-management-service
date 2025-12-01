// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/tokenomics"
	cardanoClient "gitlab.com/nunet/device-management-service/tokenomics/client/cardano"
	ethereumClient "gitlab.com/nunet/device-management-service/tokenomics/client/ethereum"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/store"
	"gitlab.com/nunet/device-management-service/tokenomics/store/payment"
	"gitlab.com/nunet/device-management-service/tokenomics/store/transaction"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	invokeMessageTimeout     = 20 * time.Second
	invokeSignRequestTimeout = 2 * time.Minute

	cardanoBlockchain  = "CARDANO"
	ethereumBlockchain = "ETHEREUM"
)

// handleContractUsagesCalculate produces the usages and forwards them to
// payment validators
func (n *Node) handleContractUsagesCalculate(msg actor.Envelope) {
	defer msg.Discard()

	resp := contracts.CollectUsagesAndForwardToPaymentProvidersReponse{}

	// Parse request, default to empty request (process all contracts) if no message
	var req contracts.CollectUsagesAndForwardToPaymentProvidersRequest
	if len(msg.Message) > 0 {
		if err := json.Unmarshal(msg.Message, &req); err != nil {
			resp.Error = fmt.Errorf("failed to unmarshal request: %w", err).Error()
			n.sendReply(msg, resp)
			return
		}
	}

	resp = n.collectUsagesAndForwardToPaymentProviders(req)
	errAggregated := ""
	for _, result := range resp.Results {
		if result.Error != "" {
			errAggregated += result.Error + "\n"
		}
	}
	if errAggregated != "" {
		resp.Error = errAggregated
	}

	n.sendReply(msg, resp)
}

// handleNewContract is registered on the contract host
func (n *Node) handleNewContract(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		n.sendReply(msg, contracts.CreateContractResponse{Error: err.Error()})
	}

	var request contracts.CreateContractRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("unmarshal create contract request: %s", err))
		return
	}

	// Service provider path: forward to contract host, persist local copy, relay response.
	if !request.SolutionEnablerDID.Equal(n.actor.Handle().DID) {
		resp, err := n.forwardContractCreateToHost(request)
		if err != nil {
			handleErr(err)
			return
		}
		n.sendReply(msg, resp)
		return
	}

	// Contract host path: existing behaviour
	resp, err := n.createContractOnHost(request)
	if err != nil {
		handleErr(err)
		return
	}

	n.sendReply(msg, resp)
}

func (n *Node) createContractOnHost(request contracts.CreateContractRequest) (contracts.CreateContractResponse, error) {
	privKey, pubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	if err != nil {
		return contracts.CreateContractResponse{}, fmt.Errorf("failed to generate contract keypair: %w", err)
	}

	contractActor, err := tokenomics.NewContractActor(n.actor.Handle(), request.PaymentValidatorDID, n.network, request.ContractParticipants, privKey, pubKey, n.contractStore, n.usageStore)
	if err != nil {
		return contracts.CreateContractResponse{}, fmt.Errorf("failed to create contract actor: %w", err)
	}

	contractObj := contracts.NewContract(contractActor.ContractDID.URI, request)
	if err := n.contractStore.Upsert(contractObj); err != nil {
		return contracts.CreateContractResponse{}, fmt.Errorf("failed to save contract: %w", err)
	}

	// Initialize usage metadata for this contract
	if err := n.usageStore.InitializeContractMetadata(contractActor.ContractDID.URI); err != nil {
		return contracts.CreateContractResponse{}, fmt.Errorf("failed to initialize usage metadata for contract %s: %w", contractActor.ContractDID.URI, err)
	}

	pkBytes, err := crypto.PublicKeyToBytes(pubKey)
	if err != nil {
		return contracts.CreateContractResponse{}, fmt.Errorf("failed to convert public key to bytes: %w", err)
	}

	privKeyBytes, err := crypto.PrivateKeyToBytes(privKey)
	if err != nil {
		return contracts.CreateContractResponse{}, fmt.Errorf("failed to convert private key to bytes: %w", err)
	}

	if err := n.contractStore.InsertContractKey(store.ContractKey{
		ContractDID: contractActor.ContractDID.URI,
		Key:         privKeyBytes,
	}); err != nil {
		return contracts.CreateContractResponse{}, fmt.Errorf("failed to save actor private key for contract %s: %w", contractActor.ContractDID.URI, err)
	}

	if err := contractActor.Start(); err != nil {
		return contracts.CreateContractResponse{}, fmt.Errorf("failed to start actor: %w", err)
	}
	n.addContractActor(contractActor)

	go func() {
		sigs, err := n.proposeContract(contractActor.ContractDID.URI)
		if err == nil {
			contractObj, err := n.contractStore.GetContract(contractActor.ContractDID.URI)
			if err != nil {
				log.Errorf("failed to get contract: %w", err)
				return
			}
			contractObj.Signatures = sigs
			if len(contractObj.Signatures) == 2 {
				contractObj.CurrentState = contracts.ContractAccepted
				contractObj.Transitions = []contracts.StateTransition{
					{
						FromState:   contracts.ContractDraft,
						ToState:     contracts.ContractAccepted,
						Timestamp:   time.Now(),
						Event:       contracts.EventAccepted,
						InitiatedBy: n.actor.Handle().DID,
					},
				}
			}
			if err := n.contractStore.Upsert(contractObj); err != nil {
				log.Errorf("failed to update contract with signatures: %w", err)
			}
		}
	}()

	return contracts.CreateContractResponse{
		ContractRequest: request,
		ContractDID:     contractActor.ContractDID.URI,
		PubKey:          hex.EncodeToString(pkBytes),
	}, nil
}

func (n *Node) forwardContractCreateToHost(request contracts.CreateContractRequest) (contracts.CreateContractResponse, error) {
	var resp contracts.CreateContractResponse

	if request.SolutionEnablerDID.Empty() {
		return resp, errors.New("solution enabler DID is empty")
	}

	destination, err := actor.HandleFromDID(request.SolutionEnablerDID.String())
	if err != nil {
		return resp, fmt.Errorf("failed to resolve contract host handle: %w", err)
	}

	envelope, err := n.invokeBehaviour(destination, behaviors.ContractCreateBehavior, request, invokeMessageTimeout)
	if err != nil {
		return resp, fmt.Errorf("failed to forward create contract request to host: %w", err)
	}

	if envelope.Message == nil {
		return resp, errors.New("contract host returned empty response")
	}

	if err := json.Unmarshal(envelope.Message, &resp); err != nil {
		return resp, fmt.Errorf("failed to unmarshal contract host response: %w", err)
	}
	if resp.Error != "" {
		return resp, fmt.Errorf("contract host error: %s", resp.Error)
	}

	localContract := contracts.NewContract(resp.ContractDID, resp.ContractRequest)
	if err := n.contractStore.Upsert(localContract); err != nil {
		return resp, fmt.Errorf("failed to save local contract copy: %w", err)
	}

	return resp, nil
}

// this behaviour is registered by service and compute provider
// its used to sign contracts and send back the response to solution enabler.
func (n *Node) handleContractPropose(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		n.sendReply(msg, contracts.ProposeContractResponse{Error: err.Error()})
	}

	var incomingContract contracts.Contract
	if err := json.Unmarshal(msg.Message, &incomingContract); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal contract: %s", err))
		return
	}

	provider, err := n.rootCap.Trust().GetProvider(n.actor.Security().DID())
	if err != nil {
		handleErr(fmt.Errorf("failed to get provider: %w", err))
		return
	}

	// if requester, no need explicit approval
	if incomingContract.ContractParticipants.Requestor.URI == provider.DID().URI {
		sig, err := incomingContract.Sign(provider)
		if err != nil {
			handleErr(fmt.Errorf("failed to sign contract: %w", err))
			return
		}

		n.sendReply(msg, contracts.ProposeContractResponse{
			Signature: contracts.Signature{
				DID:        provider.DID(),
				Signatures: sig,
			},
		})

		return
	}

	// determine service or compute provider
	// if compute provider make sure to check if its approved
	savedContract, err := n.contractStore.GetContract(incomingContract.ContractDID)
	if err != nil {
		// no contract, save it in the local store
		err := n.contractStore.Upsert(&incomingContract)
		if err != nil {
			handleErr(fmt.Errorf("failed to save contract in the db: %w", err))
			return
		}

		err = errors.New("contract is not signed yet")
		handleErr(err)
		return
	}

	if savedContract.CurrentState == contracts.ContractDraft {
		err := errors.New("contract is not signed")
		handleErr(err)
		return
	}

	sig, err := incomingContract.Sign(provider)
	if err != nil {
		handleErr(fmt.Errorf("failed to sign contract: %w", err))
		return
	}

	n.sendReply(msg, contracts.ProposeContractResponse{
		Signature: contracts.Signature{
			DID:        provider.DID(),
			Signatures: sig,
		},
	})
}

func (n *Node) proposeContract(contractDID string) ([]contracts.Signature, error) {
	contractObj, err := n.contractStore.GetContract(contractDID)
	if err != nil {
		return nil, fmt.Errorf("failed to find contract %s: %w", contractDID, err)
	}

	providerHandle, err := actor.HandleFromDID(contractObj.ContractParticipants.Provider.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to get provider's DID: %w", err)
	}

	requesterHandle, err := actor.HandleFromDID(contractObj.ContractParticipants.Requestor.URI)
	if err != nil {
		return nil, fmt.Errorf("failed to get requester's DID: %w", err)
	}

	propose := func(handle actor.Handle) (*contracts.Signature, error) {
		envelope, err := n.invokeBehaviour(handle, behaviors.ContractProposeBehavior, *contractObj, invokeMessageTimeout)
		if envelope.Message != nil && err == nil {
			var response contracts.ProposeContractResponse
			err := json.Unmarshal(envelope.Message, &response)
			if err == nil && response.Error == "" {
				return &response.Signature, nil
			}
		}

		return nil, fmt.Errorf("failed to get back response from %s", handle.DID)
	}

	sigs := make([]contracts.Signature, 0)
	providerSig, err := propose(providerHandle)
	if err == nil {
		sigs = append(sigs, *providerSig)
	}
	requesterSig, err := propose(requesterHandle)
	if err == nil {
		sigs = append(sigs, *requesterSig)
	}

	return sigs, nil
}

// compute provider handles approvals
func (n *Node) handleContractApprovalLocal(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		n.sendReply(msg, contracts.ContractApproveLocalResponse{Error: err.Error()})
	}

	var req contracts.ContractApproveLocalRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal contract approve request: %s", err))
		return
	}

	savedContract, err := n.contractStore.GetContract(req.ContractDID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get contract: %w", err))
		return
	}

	savedContract.CurrentState = contracts.ContractAccepted
	err = n.contractStore.Upsert(savedContract)
	if err != nil {
		handleErr(fmt.Errorf("failed to update contract: %w", err))
		return
	}

	// sign the contract and send it to the contract host
	contractDID, err := did.FromString(savedContract.ContractDID)
	if err != nil {
		handleErr(fmt.Errorf("failed to convert contract did: %w", err))
		return
	}

	pubKey, err := did.PublicKeyFromDID(contractDID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get public key from contract host did: %w", err))
		return
	}

	pubKeySolutionEnabler, err := did.PublicKeyFromDID(savedContract.SolutionEnablerDID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get public key: %w", err))
		return
	}

	soltionEnablerPeerID, err := peer.IDFromPublicKey(pubKeySolutionEnabler)
	if err != nil {
		handleErr(fmt.Errorf("failed to peer id from public key: %w", err))
		return
	}

	destination, err := actor.HandleFromPublicKeyWithInboxAddress(pubKey, savedContract.ContractDID, soltionEnablerPeerID.String())
	if err != nil {
		handleErr(fmt.Errorf("failed to get get contract host handle: %w", err))
		return
	}

	provider, err := n.rootCap.Trust().GetProvider(n.actor.Security().DID())
	if err != nil {
		handleErr(fmt.Errorf("failed to get provider: %w", err))
		return
	}

	sig, err := savedContract.Sign(provider)
	if err != nil {
		handleErr(fmt.Errorf("failed to sign contract: %w", err))
		return
	}

	signReq := contracts.ContractSignRequest{
		ContractDID: savedContract.ContractDID,
		Signature:   sig,
	}
	reply, err := n.invokeBehaviour(destination, behaviors.ContractSignBehavior, signReq, invokeSignRequestTimeout)
	if err != nil {
		handleErr(fmt.Errorf("failed to get invoke sign contract on contract host: %w", err))
		return
	}

	var signResp contracts.ContractSignResponse
	if err := json.Unmarshal(reply.Message, &signResp); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal contract host response: %w", err))
		return
	}

	if signResp.Error != "" {
		handleErr(fmt.Errorf("error from contract host: %s", signResp.Error))
		return
	}

	n.sendReply(msg, contracts.ContractApproveLocalResponse{
		Success: true,
	})
}

// compute provider can list incoming contracts for approval
func (n *Node) handleListIncomingContracts(msg actor.Envelope) {
	defer msg.Discard()
	handleErr := func(err error) {
		n.sendReply(msg, contracts.ContractListIncomingResponse{Error: err.Error()})
	}

	var req contracts.ContractListIncomingRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal list incoming request: %w", err))
		return
	}

	allContracts, err := n.contractStore.GetAllContracts()
	if err != nil {
		handleErr(fmt.Errorf("failed to get all contracts: %w", err))
		return
	}

	callerDID := msg.From.DID.String()
	rootDID := n.rootCap.DID().String()
	filteredLocal := filterContractsByRole(allContracts, req.Role, callerDID)

	if callerDID == rootDID {
		solutionHosts := uniqueSolutionEnablerDIDs(allContracts)
		if len(solutionHosts) == 0 {
			log.Warnf("no solution hosts found (i.e: no contracts created yet) for caller %s", callerDID)
			handleErr(fmt.Errorf("no solution hosts found to retrieve contracts from for caller %s", callerDID))
			return
		}
		aggregated := make(map[string]*contracts.Contract, len(filteredLocal))

		for _, hostDID := range solutionHosts {
			if hostDID == "" {
				continue
			}

			// if the solution enabler is this node, use local data
			if hostDID == rootDID {
				for _, c := range filterContractsByRole(allContracts, req.Role, callerDID) {
					aggregated[c.ContractDID] = c
				}
				continue
			}

			handle, err := actor.HandleFromDID(hostDID)
			if err != nil {
				log.Warnf("failed to build handle for host %s: %v", hostDID, err)
				continue
			}

			reply, err := n.invokeBehaviour(handle, behaviors.ContractListBehavior, req, invokeMessageTimeout)
			if err != nil || reply.Message == nil {
				log.Warnf("failed to invoke list incoming on host %s: %v", hostDID, err)
				continue
			}

			var remoteResp contracts.ContractListIncomingResponse
			if err := json.Unmarshal(reply.Message, &remoteResp); err != nil {
				log.Warnf("failed to decode contract host response %s: %v", hostDID, err)
				continue
			}
			if remoteResp.Error != "" {
				log.Warnf("host %s returned error listing incoming contracts: %s", hostDID, remoteResp.Error)
				continue
			}

			for _, c := range remoteResp.Contracts {
				aggregated[c.ContractDID] = c
			}
		}

		contractsSlice := make([]*contracts.Contract, 0, len(aggregated))
		for _, c := range aggregated {
			contractsSlice = append(contractsSlice, c)
		}

		n.sendReply(msg, contracts.ContractListIncomingResponse{
			Contracts: contractsSlice,
		})
		return
	}

	// Contract host invocation: respond with local contracts only
	resp := contracts.ContractListIncomingResponse{
		Contracts: filteredLocal,
	}
	n.sendReply(msg, resp)
}

func filterContractsByRole(contractsList []*contracts.Contract, role contracts.ContractListIncomingRole, targetDID string) []*contracts.Contract {
	result := make([]*contracts.Contract, 0, len(contractsList))
	for _, c := range contractsList {
		switch role {
		case contracts.ContractRoleProvider:
			if targetDID == "" || c.ContractParticipants.Provider.String() == targetDID {
				result = append(result, c)
			}
		case contracts.ContractRoleRequestor:
			if targetDID == "" || c.ContractParticipants.Requestor.String() == targetDID {
				result = append(result, c)
			}
		default:
			if targetDID == "" || c.SolutionEnablerDID.String() == targetDID || c.ContractParticipants.Provider.String() == targetDID || c.ContractParticipants.Requestor.String() == targetDID {
				result = append(result, c)
			}
		}
	}
	return result
}

func uniqueSolutionEnablerDIDs(contractsList []*contracts.Contract) []string {
	unique := make(map[string]struct{}, len(contractsList))
	for _, c := range contractsList {
		host := c.SolutionEnablerDID.String()
		if host == "" {
			continue
		}
		unique[host] = struct{}{}
	}

	hosts := make([]string, 0, len(unique))
	for host := range unique {
		hosts = append(hosts, host)
	}

	return hosts
}

func (n *Node) StartContracts() error {
	allContracts, err := n.contractStore.GetAllContracts()
	if err != nil {
		return fmt.Errorf("failed to starts contracts: %w", err)
	}

	for _, v := range allContracts {
		key, err := n.contractStore.GetContractKey(v.ContractDID)
		if err != nil {
			log.Warnf("failed to get contract %s private key: %v", v.ContractDID, err)
			continue
		}

		privKey, err := crypto.BytesToPrivateKey(key.Key)
		if err != nil {
			continue
		}

		pubKey := privKey.GetPublic()

		contractActor, err := tokenomics.NewContractActor(n.actor.Handle(), v.PaymentValidatorDID, n.network, v.ContractParticipants, privKey, pubKey, n.contractStore, n.usageStore)
		if err != nil {
			continue
		}
		err = contractActor.Start()
		if err != nil {
			continue
		}
	}

	return nil
}

// payment validator to accept validation requests
func (n *Node) handleContractPaymentValidationRequestFromContractHost(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		n.sendReply(msg, contracts.ContractPaymentValidationResponse{Error: err.Error()})
	}

	var req contracts.ContractPaymentValidationRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal payment validation request: %s", err))
		return
	}

	payment, err := n.paymentStore.GetByUniqueID(req.UniqueID)
	if err != nil {
		handleErr(fmt.Errorf("failed to find payment with unique id: %s", req.UniqueID))
		return
	}
	verified := false
	errorMsg := ""

	switch req.Blockchain {
	case ethereumBlockchain:
		ethAddr := types.PaymentAddressInfo{}
		foundEthAddr := false
		for _, v := range payment.Contract.PaymentDetails.Addresses {
			if v.Blockchain == ethereumBlockchain {
				ethAddr = v
				foundEthAddr = true
				break
			}
		}
		if !foundEthAddr {
			handleErr(fmt.Errorf("ethereum address was not found in payment addresses: %w", err))
			return
		}

		c := ethereumClient.NewClient(
			n.dmsConfig.PaymentProvider.EthereumRPCURL,
			n.dmsConfig.PaymentProvider.EthereumRPCToken,
		)
		txs, err := ethereumClient.GetERC20Transfers(
			c,
			n.dmsConfig.PaymentProvider.NtxContractAddress,
			ethAddr.ProviderAddr,
			n.dmsConfig.PaymentProvider.StartingBlockScanning,
			"latest",
		)
		if err != nil {
			handleErr(fmt.Errorf("failed to get erc20 transfer: %w", err))
			return
		}

		for _, tx := range txs {
			if strings.EqualFold(tx.TxHash, req.TxHash) {
				if !strings.EqualFold(tx.From, ethAddr.RequesterAddr) {
					handleErr(fmt.Errorf("requester transaction address %s doesn't match the one in transaction: %s", ethAddr.RequesterAddr, tx.From))
					return
				}

				ok, err := compareDecimals(tx.Amount, payment.Amount)
				if err != nil {
					errorMsg = err.Error() + " tx amount: " + tx.Amount + " payment amount: " + payment.Amount
				}
				if ok {
					verified = true
				} else {
					errorMsg = "not verified: tx amount: " + tx.Amount + " payment amount: " + payment.Amount
				}

				break
			}
		}

	case cardanoBlockchain:
		cardanoAddr := types.PaymentAddressInfo{}
		foundCardanoAddr := false
		for _, v := range payment.Contract.PaymentDetails.Addresses {
			if v.Blockchain == cardanoBlockchain {
				cardanoAddr = v
				foundCardanoAddr = true
				break
			}
		}
		if !foundCardanoAddr {
			handleErr(fmt.Errorf("cardano address was not found in payment addresses: %w", err))
			return
		}

		client := cardanoClient.NewClient(
			n.dmsConfig.PaymentProvider.BlockFrostAPIKey,
			n.dmsConfig.PaymentProvider.BlockFrostAPIURL,
		)
		asset := n.dmsConfig.PaymentProvider.CardanoAssetPolicyID + hex.EncodeToString([]byte(n.dmsConfig.PaymentProvider.CardanoAssetName))
		txs, err := client.FindTxsToAddressForAsset(n.ctx, asset, cardanoAddr.ProviderAddr)
		if err != nil {
			handleErr(fmt.Errorf("failed to get cardano transactions: %w", err))
			return
		}

		for _, tx := range txs {
			if strings.EqualFold(tx.TxHash, req.TxHash) {
				foundFrom := false
				for _, v := range tx.FromAddrs {
					if v == cardanoAddr.RequesterAddr {
						foundFrom = true
						break
					}
				}

				if !foundFrom {
					handleErr(fmt.Errorf("requester transaction address not found: %s", cardanoAddr.RequesterAddr))
					return
				}

				ok, err := compareDecimals(tx.Quantity, payment.Amount)
				if err != nil {
					errorMsg = err.Error() + " tx amount: " + tx.Quantity + " payment amount: " + payment.Amount
				}
				if ok {
					verified = true
				} else {
					errorMsg = "not verified: tx amount: " + tx.Quantity + " payment amount: " + payment.Amount
				}

				break
			}
		}

	default:
		handleErr(fmt.Errorf("unsupported blockchain payment info: %s", req.Blockchain))
		return
	}

	resp := contracts.ContractPaymentValidationResponse{}
	if verified {
		payment.Paid = true
		err := n.paymentStore.Update(payment)
		if err != nil {
			resp.Error = err.Error()
		}
	} else {
		if errorMsg != "" {
			resp.Error = errorMsg
		} else {
			resp.Error = "not verified"
		}
	}

	n.sendReply(msg, resp)
}

func (n *Node) handleConfirmLocalTransaction(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		n.sendReply(msg, contracts.ContractConfirmLocalTransactionResponse{Error: err.Error()})
	}

	var req contracts.ContractConfirmLocalTransactionRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal incoming transaction confirm request: %s", err))
		return
	}

	paymentProviderDID, err := n.transactionStore.MarkAsPaid(req.UniqueID, req.TxHash)
	if err != nil {
		handleErr(fmt.Errorf("failed to get mark transaction as paid: %s", err))
		return
	}

	paymentValidationReq := contracts.ContractPaymentValidationRequest{
		TxHash:     req.TxHash,
		UniqueID:   req.UniqueID,
		Blockchain: req.Blockchain,
	}
	paymentProvider, err := actor.HandleFromDID(paymentProviderDID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get payment provider hande: %w", err))
		return
	}
	reply, err := n.invokeBehaviour(paymentProvider, behaviors.ContractPaymentValidationRequestBehavior, paymentValidationReq, invokeMessageTimeout)
	if err != nil {
		handleErr(fmt.Errorf("failed to send transaction confirmation to payment provider: %w", err))
		return
	}

	var replyResponse contracts.ContractPaymentValidationResponse
	_ = json.Unmarshal(reply.Message, &replyResponse)
	if replyResponse.Error != "" {
		handleErr(fmt.Errorf("payment validation response from payment provider: %s", replyResponse.Error))
		return
	}

	resp := contracts.ContractConfirmLocalTransactionResponse{}

	n.sendReply(msg, resp)
}

func (n *Node) handleListLocalTransactions(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		n.sendReply(msg, contracts.ContractListLocalTransactionsResponse{Error: err.Error()})
	}

	txs, err := n.transactionStore.AllTransactions()
	if err != nil {
		handleErr(fmt.Errorf("failed to get local transactions: %s", err))
		return
	}

	resp := contracts.ContractListLocalTransactionsResponse{
		Transactions: txs,
	}

	n.sendReply(msg, resp)
}

func (n *Node) handlePaymentStatus(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		n.sendReply(msg, contracts.ContractPaymentStatusResponse{Error: err.Error()})
	}

	var req contracts.ContractPaymentStatusRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal get payment request: %s", err))
		return
	}

	p, err := n.paymentStore.GetByUniqueID(req.UniqueID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get payment: %s", err))
		return
	}

	resp := contracts.ContractPaymentStatusResponse{
		UniqueID: p.UniqueID,
		Paid:     p.Paid,
	}

	n.sendReply(msg, resp)
}

func (n *Node) handleIncomingTransaction(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		n.sendReply(msg, contracts.TransactionForServiceProviderResponse{Error: err.Error()})
	}

	var req contracts.TransactionForServiceProviderRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal incoming transaction request: %s", err))
		return
	}

	err := n.transactionStore.Upsert(transaction.Transaction{
		UniqueID:            req.UniqueID,
		PaymentValidatorDID: req.PaymentValidatorDID,
		ContractDID:         req.ContractDID,
		ToAddress:           req.ToAddress,
		Amount:              req.Amount,
	})
	if err != nil {
		handleErr(fmt.Errorf("failed to insert transaction into the store: %w", err))
		return
	}

	resp := contracts.TransactionForServiceProviderResponse{}
	n.sendReply(msg, resp)
}

// payment provider listens for requests from contract host
// about usages of a contracts. As a payment provider, we should
// contact the service provider for what amount to pay.
func (n *Node) handleIncomingContractUsage(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		n.sendReply(msg, contracts.ContractUsageResponse{Error: err.Error()})
	}

	var req contracts.ContractUsageRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal incoming contract usages request: %s", err))
		return
	}

	// Calculate payment based on payment model
	var finalAmount string
	var calcErr error

	switch req.Contract.PaymentDetails.PaymentModel {
	case contracts.PayPerAllocation:
		if req.Contract.PaymentDetails.FeesPerAllocation == "" {
			handleErr(errors.New("fees_per_allocation is required for pay_per_allocation payment model"))
			return
		}
		finalAmount, calcErr = calculateTotal(req.Usages, req.Contract.PaymentDetails.FeesPerAllocation)
	case contracts.PayPerDeployment:
		if req.Contract.PaymentDetails.FeePerDeployment == "" {
			handleErr(errors.New("fee_per_deployment is required for pay_per_deployment payment model"))
			return
		}
		finalAmount, calcErr = calculateTotal(req.Usages, req.Contract.PaymentDetails.FeePerDeployment)
	default:
		handleErr(fmt.Errorf("unsupported payment model: %s", req.Contract.PaymentDetails.PaymentModel))
		return
	}

	if calcErr != nil {
		handleErr(fmt.Errorf("failed to calculate final tx amount: %w", calcErr))
		return
	}

	err := n.paymentStore.Insert(payment.Payment{
		UniqueID: req.UniqueID,
		Contract: req.Contract,
		Usages:   req.Usages,
		Paid:     false,
		Amount:   finalAmount,
	})
	if err != nil {
		handleErr(errors.New("error while upserting payment"))
		return
	}

	txReq := contracts.TransactionForServiceProviderRequest{
		PaymentValidatorDID: req.Contract.PaymentValidatorDID.URI,
		UniqueID:            req.UniqueID,
		ContractDID:         req.Contract.ContractDID,
		ToAddress:           req.Contract.PaymentDetails.Addresses,
		Amount:              finalAmount,
	}
	go func() {
		destination, err := actor.HandleFromDID(req.Contract.ContractParticipants.Requestor.URI)
		if err != nil {
			log.Errorf("failed to get service provider's DID: %w", err)
			return
		}
		reply, err := n.invokeBehaviour(destination, behaviors.ContractTransactionBehavior, txReq, invokeMessageTimeout)
		if reply.Message == nil || err != nil {
			log.Errorf("failed to forward transaction info to service provider: %w", err)
		}
	}()

	resp := contracts.ContractUsageResponse{}
	n.sendReply(msg, resp)
}

func calculateTotal(numItems int, feePerItem string) (string, error) {
	fee, err := strconv.ParseFloat(feePerItem, 64)
	if err != nil {
		return "", fmt.Errorf("invalid fee: %v", err)
	}

	numItemsBig := big.NewFloat(float64(numItems))
	feeBig := big.NewFloat(fee)
	totalBig := new(big.Float).Mul(numItemsBig, feeBig)

	totalStr := fmt.Sprintf("%.6f", totalBig)
	return totalStr, nil
}

// true if a is bigger than b
func compareDecimals(a, b string) (bool, error) {
	af, _, err := big.ParseFloat(a, 10, 256, big.ToNearestEven)
	if err != nil {
		return false, err
	}
	bf, _, err := big.ParseFloat(b, 10, 256, big.ToNearestEven)
	if err != nil {
		return false, err
	}
	return af.Cmp(bf) >= 0, nil
}
