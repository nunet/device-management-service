// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"context"
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
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/tokenomics"
	cardanoClient "gitlab.com/nunet/device-management-service/tokenomics/client/cardano"
	ethereumClient "gitlab.com/nunet/device-management-service/tokenomics/client/ethereum"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/store"
	payment_quote "gitlab.com/nunet/device-management-service/tokenomics/store/payment_quote"
	"gitlab.com/nunet/device-management-service/tokenomics/store/transaction"
	"gitlab.com/nunet/device-management-service/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
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

	solutionEnablerDID := ""
	handleErr := func(err error) {
		log.Errorw("handle_contract_propose",
			"labels", []string{string(observability.LabelContract)},
			"error", err, "solutionEnablerDID", solutionEnablerDID)
		n.sendReply(msg, contracts.CreateContractResponse{Error: err.Error()})
	}

	var request contracts.CreateContractRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("unmarshal create contract request: %s", err))
		return
	}

	// Validate required fields
	if err := request.Validate(); err != nil {
		handleErr(fmt.Errorf("invalid contract request: %w", err))
		return
	}

	solutionEnablerDID = request.SolutionEnablerDID.String()

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

	creatorOfContract := msg.From
	// Contract host path: existing behaviour
	resp, err := n.createContractOnHost(request, creatorOfContract)
	if err != nil {
		handleErr(err)
		return
	}

	n.sendReply(msg, resp)
}

func (n *Node) createContractOnHost(request contracts.CreateContractRequest, creatorOfContract actor.Handle) (contracts.CreateContractResponse, error) {
	privKey, pubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	if err != nil {
		return contracts.CreateContractResponse{}, fmt.Errorf("failed to generate contract keypair: %w", err)
	}

	// Validate payment details before creating contract
	processor, err := contracts.GetPaymentModelProcessor(request.PaymentDetails.PaymentModel)
	if err != nil {
		return contracts.CreateContractResponse{}, fmt.Errorf("invalid payment model %q: %w", request.PaymentDetails.PaymentModel, err)
	}

	if err := processor.Validate(request.PaymentDetails); err != nil {
		return contracts.CreateContractResponse{}, fmt.Errorf("invalid payment details: %w", err)
	}

	// Create forwardInvoice function to forward invoices from contract actor to payment validator
	forwardInvoice := func(req contracts.ContractUsageRequest) error {
		destination, err := actor.HandleFromDID(request.PaymentValidatorDID.URI)
		if err != nil {
			return fmt.Errorf("failed to get payment validator handle: %w", err)
		}
		envelope, err := n.invokeBehaviour(destination, behaviors.ContractUsageBehavior, req, invokeMessageTimeout)
		if envelope.Message == nil || err != nil {
			return fmt.Errorf("failed to forward invoice to payment validator: %w", err)
		}
		log.Infof("Successfully sent invoice for contract %s to payment validator (payment_model: %s)", req.Contract.ContractDID, req.Contract.PaymentDetails.PaymentModel)
		return nil
	}

	contractActor, err := tokenomics.NewContractActor(n.actor.Handle(), request.PaymentValidatorDID, n.network, request.ContractParticipants, privKey, pubKey, n.contractStore, n.usageStore, forwardInvoice)
	if err != nil {
		return contracts.CreateContractResponse{}, fmt.Errorf("failed to create contract actor: %w", err)
	}

	contractObj := contracts.NewContract(contractActor.ContractDID.URI, request)

	// Determine if this is a Head Contract in a chain and set metadata
	// Head Contract: Provider = Organization (not a compute provider)
	// This can be detected from ContractParticipants structure or explicit flag
	// For now, we'll use a helper function that can be enhanced based on actual use cases
	if isHeadContractFromRequest(request) {
		if contractObj.Metadata == nil {
			contractObj.Metadata = make(map[string]string)
		}
		contractObj.Metadata[contracts.ContractChainRoleMetadataKey] = contracts.ContractChainRoleHead
	}

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

	// Register with billing scheduler
	// NOTE: This is safe even if the contract is later loaded by StartContracts()
	// because RegisterContract() is idempotent and checks for existing registration
	if n.billingScheduler != nil {
		err = contractActor.RegisterBilling(n.billingScheduler)
		if err != nil {
			log.Warnw("failed to register contract for billing",
				"contract_did", contractActor.ContractDID.URI,
				"error", err)
			// Don't fail contract creation if billing registration fails
		}
	}

	// Store actor reference in map for O(1) lookup
	n.addContractActor(contractActor)

	// if solution enabler, propose to parties
	if request.SolutionEnablerDID.Equal(n.actor.Handle().DID) {
		go func() {
			sigs, err := n.proposeContract(contractActor.ContractDID.URI, creatorOfContract)
			if err != nil {
				log.Errorf("failed to propose contract: %w", err)
				return
			}

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
		}()
	}

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

	contractID := ""
	handleErr := func(err error) {
		log.Errorw("handle_contract_propose",
			"labels", []string{string(observability.LabelContract)},
			"error", err, "contractID", contractID)
		n.sendReply(msg, contracts.ProposeContractResponse{Error: err.Error()})
	}

	var request contracts.ProposeContractRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal contract: %s", err))
		return
	}
	incomingContract := request.Contract
	contractID = request.Contract.ContractDID

	provider, err := n.rootCap.Trust().GetProvider(n.actor.Security().DID())
	if err != nil {
		handleErr(fmt.Errorf("failed to get provider: %w", err))
		return
	}

	providerDID := provider.DID()
	isCreator := request.CreatorOfContract.DID.Equal(providerDID)
	isRequestor := providerDID.Equal(incomingContract.ContractParticipants.Requestor)
	isProvider := providerDID.Equal(incomingContract.ContractParticipants.Provider)

	// Auto-sign if creator and is a participant
	if isCreator && (isRequestor || isProvider) {
		log.Infof("automatically signing contract for creator/requestor: %s", providerDID.URI)
		sig, err := incomingContract.Sign(provider)
		if err != nil {
			handleErr(fmt.Errorf("failed to sign proposed contract: %w", err))
			return
		}

		err = n.contractStore.Upsert(&incomingContract)
		if err != nil {
			handleErr(fmt.Errorf("failed to save proposed contract in the db: %w", err))
			return
		}

		n.sendReply(msg, contracts.ProposeContractResponse{
			Signature: contracts.Signature{
				DID:        providerDID,
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

func (n *Node) proposeContract(contractDID string, creatorOfContract actor.Handle) ([]contracts.Signature, error) {
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
		envelope, err := n.invokeBehaviour(handle, behaviors.ContractProposeBehavior, contracts.ProposeContractRequest{
			Contract:          *contractObj,
			CreatorOfContract: creatorOfContract,
		}, invokeMessageTimeout)
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

	contractID := ""
	handleErr := func(err error) {
		log.Errorw("handle_contract_approval_local",
			"labels", []string{string(observability.LabelContract)},
			"error", err, "contractID", contractID)
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
	contractID = savedContract.ContractDID

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
	contractID := ""
	handleErr := func(err error) {
		log.Errorw("handle_list_incoming_contracts",
			"labels", []string{string(observability.LabelContract)},
			"error", err, "contractID", contractID)
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

// handleContractInfo handles requests for contract information
// This is used by deployment handlers to retrieve Provider and Requestor DIDs
// from the contract host when they're not specified in the ensemble
func (n *Node) handleContractInfo(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorw("handle_contract_info",
			"labels", []string{string(observability.LabelContract)},
			"error", err)
		n.sendReply(msg, behaviors.ContractInfoResponse{
			OK:    false,
			Error: err.Error(),
		})
	}

	var req behaviors.ContractInfoRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal contract info request: %w", err))
		return
	}

	if req.ContractDID == "" {
		handleErr(errors.New("contract_did is required"))
		return
	}

	// Get contract from store
	contract, err := n.contractStore.GetContract(req.ContractDID)
	if err != nil {
		handleErr(fmt.Errorf("contract not found: %w", err))
		return
	}

	// Return Provider and Requestor DIDs
	resp := behaviors.ContractInfoResponse{
		OK:        true,
		Provider:  contract.ContractParticipants.Provider.String(),
		Requestor: contract.ContractParticipants.Requestor.String(),
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

		// Create forwardInvoice function to forward invoices from contract actor to payment validator
		forwardInvoice := func(req contracts.ContractUsageRequest) error {
			destination, err := actor.HandleFromDID(v.PaymentValidatorDID.URI)
			if err != nil {
				return fmt.Errorf("failed to get payment validator handle: %w", err)
			}
			envelope, err := n.invokeBehaviour(destination, behaviors.ContractUsageBehavior, req, invokeMessageTimeout)
			if envelope.Message == nil || err != nil {
				return fmt.Errorf("failed to forward invoice to payment validator: %w", err)
			}
			log.Infof("Successfully sent invoice for contract %s to payment validator (payment_model: %s)", req.Contract.ContractDID, req.Contract.PaymentDetails.PaymentModel)
			return nil
		}

		contractActor, err := tokenomics.NewContractActor(n.actor.Handle(), v.PaymentValidatorDID, n.network, v.ContractParticipants, privKey, pubKey, n.contractStore, n.usageStore, forwardInvoice)
		if err != nil {
			continue
		}
		err = contractActor.Start()
		if err != nil {
			continue
		}

		// Register with billing scheduler
		// NOTE: This is idempotent - if a contract was already registered
		// (e.g., created while node was running), this will be a no-op
		if n.billingScheduler != nil {
			err = contractActor.RegisterBilling(n.billingScheduler)
			if err != nil {
				log.Warnw("failed to register contract for billing",
					"contract_did", contractActor.ContractDID.URI,
					"error", err)
			}
		}

		// Store actor reference in map for O(1) lookup
		n.addContractActor(contractActor)
	}

	return nil
}

// payment validator to accept validation requests
func (n *Node) handleContractPaymentValidationRequestFromContractHost(msg actor.Envelope) {
	defer msg.Discard()

	contractID := ""
	handleErr := func(err error) {
		log.Errorw("handle_contract_payment_validation_request_from_contract_host",
			"labels", []string{string(observability.LabelContract)},
			"error", err, "contractID", contractID)
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

	// Check if there's a quote for this transaction
	var expectedAmount string
	if req.QuoteID != "" {
		// If quote_id is provided, get quote by ID
		quote, err := n.paymentQuoteStore.GetQuote(req.QuoteID)
		if err != nil {
			handleErr(fmt.Errorf("failed to get quote: %w", err))
			return
		}
		if quote.Used {
			expectedAmount = quote.ConvertedAmount
		} else {
			// Fallback to payment amount if quote not found or not used
			expectedAmount = payment.Amount
		}
	} else {
		// Try to find used quote by unique_id
		quote, err := n.paymentQuoteStore.GetQuoteByUniqueID(req.UniqueID)
		if quote != nil && quote.Used {
			expectedAmount = quote.ConvertedAmount
		} else if err != nil { // meaning no quote was used
			// Fallback to payment amount (for backward compatibility)
			expectedAmount = payment.Amount
		}
	}

	verified := false
	errorMsg := ""
	contractID = payment.Contract.ContractDID

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

		blockNum, err := ethereumClient.GetBlockNumber(c)
		if err != nil {
			handleErr(fmt.Errorf("failed to get block number: %w", err))
			return
		}

		// deduct some block numbers
		blockNum -= 1800 // 5 hours back approx
		blockNumHex := fmt.Sprintf("0x%x", blockNum)

		txs, err := ethereumClient.GetERC20Transfers(
			c,
			n.dmsConfig.PaymentProvider.NtxContractAddress,
			ethAddr.ProviderAddr,
			blockNumHex,
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

				ok, err := compareDecimals(tx.Amount, expectedAmount)
				if err != nil {
					errorMsg = err.Error() + " tx amount: " + tx.Amount + " expected amount: " + expectedAmount
				}
				if ok {
					verified = true
				} else {
					errorMsg = "not verified: tx amount: " + tx.Amount + " expected amount: " + expectedAmount
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

				ok, err := compareDecimals(tx.Quantity, expectedAmount)
				if err != nil {
					errorMsg = err.Error() + " tx amount: " + tx.Quantity + " expected amount: " + expectedAmount
				}
				if ok {
					verified = true
				} else {
					errorMsg = "not verified: tx amount: " + tx.Quantity + " expected amount: " + expectedAmount
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

	contractID := ""
	handleErr := func(err error) {
		log.Errorw("handle_confirm_local_transaction",
			"labels", []string{string(observability.LabelContract)},
			"error", err, "contractID", contractID)
		n.sendReply(msg, contracts.ContractConfirmLocalTransactionResponse{Error: err.Error()})
	}

	var req contracts.ContractConfirmLocalTransactionRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal incoming transaction confirm request: %s", err))
		return
	}

	paymentProviderDID, err := n.transactionStore.GetPaymentValidatorDID(req.UniqueID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get payment validator did: %s", err))
		return
	}
	contractID = paymentProviderDID

	// If quote_id is provided, validate and mark as used
	if req.QuoteID != "" {
		// Validate quote is still valid (not expired, not used)
		quote, err := n.paymentQuoteStore.ValidateQuote(req.QuoteID)
		if err != nil {
			handleErr(fmt.Errorf("quote validation failed: %w", err))
			return
		}

		if quote.UniqueID != req.UniqueID {
			handleErr(fmt.Errorf("quote does not match transaction"))
			return
		}

		// Mark quote as used
		if err := n.paymentQuoteStore.MarkQuoteAsUsed(req.QuoteID); err != nil {
			handleErr(fmt.Errorf("failed to mark quote as used: %w", err))
			return
		}
	}

	paymentValidationReq := contracts.ContractPaymentValidationRequest{
		TxHash:     req.TxHash,
		UniqueID:   req.UniqueID,
		Blockchain: req.Blockchain,
		QuoteID:    req.QuoteID,
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

	_, err = n.transactionStore.MarkAsPaid(req.UniqueID, req.TxHash)
	if err != nil {
		handleErr(fmt.Errorf("failed to get mark transaction as paid: %s", err))
		return
	}

	// Forward paid transaction to compute provider
	tx, err := n.transactionStore.GetTransactionByUniqueID(req.UniqueID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get transaction: %w", err))
		return
	}

	contract, err := n.contractStore.GetContract(tx.ContractDID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get contract: %w", err))
		return
	}
	// Send paid transaction to compute provider
	computeProviderTxReq := contracts.TransactionForServiceProviderRequest{
		PaymentValidatorDID: tx.PaymentValidatorDID,
		UniqueID:            tx.UniqueID,
		ContractDID:         tx.ContractDID,
		ToAddress:           tx.ToAddress,
		Amount:              tx.Amount,
		Status:              "paid", // Mark as paid
		TxHash:              req.TxHash,
		Metadata:            tx.Metadata,
	}

	destination, err := actor.HandleFromDID(contract.ContractParticipants.Provider.URI)
	if err != nil {
		handleErr(fmt.Errorf("failed to get destination handle: %w", err))
		return
	}

	_, err = n.invokeBehaviour(
		destination,
		behaviors.ContractTransactionBehavior,
		computeProviderTxReq,
		invokeMessageTimeout,
	)
	if err != nil {
		handleErr(fmt.Errorf("failed to forward paid transaction to compute provider: %w", err))
		return
	}

	// metric
	if m := observability.TxPaidAmount; m != nil {
		amount, err := strconv.ParseFloat(tx.Amount, 64)
		if err == nil {
			m.Add(n.ctx, amount, metric.WithAttributes(
				observability.AttrDID,
				attribute.String("ContractDID", tx.ContractDID),
			))
		}
	}

	log.Infof("successfully forwarded paid transaction %s to compute provider", req.UniqueID)
	n.sendReply(msg, contracts.ContractConfirmLocalTransactionResponse{})
}

func (n *Node) handleListLocalTransactions(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorw("handle_list_local_transactions",
			"labels", []string{string(observability.LabelContract)},
			"error", err)
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

	contractID := ""
	handleErr := func(err error) {
		log.Errorw("handle_incoming_transaction",
			"labels", []string{string(observability.LabelContract)},
			"error", err, "contractID", contractID)
		n.sendReply(msg, contracts.TransactionForServiceProviderResponse{Error: err.Error()})
	}

	var req contracts.TransactionForServiceProviderRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal incoming transaction request: %s", err))
		return
	}
	contractID = req.ContractDID

	err := n.transactionStore.Upsert(transaction.Transaction{
		UniqueID:            req.UniqueID,
		PaymentValidatorDID: req.PaymentValidatorDID,
		ContractDID:         req.ContractDID,
		ToAddress:           req.ToAddress,
		Amount:              req.Amount,
		Status:              req.Status, // Use provided status, or "" defaults to "unpaid" in Upsert
		TxHash:              req.TxHash, // Use provided tx hash
		Metadata:            req.Metadata,

		// Store conversion metadata if provided
		OriginalAmount:     req.OriginalAmount,     // Amount in pricing currency (USDT)
		PricingCurrency:    req.PricingCurrency,    // Currency of original amount
		RequiresConversion: req.RequiresConversion, // True if conversion is needed
	})
	if err != nil {
		handleErr(fmt.Errorf("failed to insert transaction into the store: %w", err))
		return
	}

	// metric
	if m := observability.TxCreatedAmount; m != nil {
		amount, err := strconv.ParseFloat(req.Amount, 64)
		if err == nil {
			m.Add(n.ctx, amount, metric.WithAttributes(
				observability.AttrDID,
				attribute.String("ContractDID", req.ContractDID),
			))
		}
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
		log.Errorf("handleIncomingContractUsage error: %v", err)
		n.sendReply(msg, contracts.ContractUsageResponse{Error: err.Error()})
	}

	var req contracts.ContractUsageRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal incoming contract usages request: %s", err))
		return
	}

	log.Infof("handleIncomingContractUsage: payment_model=%s, usages=%d, hasTimeUtilization=%v, hasResourceUtilization=%v, hasPeriodicDetails=%v, contractDID=%s",
		req.Contract.PaymentDetails.PaymentModel, req.Usages,
		req.TimeUtilization != nil, req.ResourceUtilization != nil, req.PeriodicDetails != nil, req.Contract.ContractDID)

	// Get processor for this payment model
	processor, err := contracts.GetPaymentModelProcessor(req.Contract.PaymentDetails.PaymentModel)
	if err != nil {
		handleErr(fmt.Errorf("unsupported payment model: %w", err))
		return
	}

	// Convert request to UsageData format
	usageData, err := n.convertRequestToUsageData(&req)
	if err != nil {
		handleErr(fmt.Errorf("failed to convert request to usage data: %w", err))
		return
	}

	// Edge case: Periodic with no deployments - skip processing
	if usageData == nil {
		resp := contracts.ContractUsageResponse{}
		n.sendReply(msg, resp)
		return
	}

	// Calculate payment items using processor
	items, err := processor.CalculatePayment(usageData, &req.Contract)
	if err != nil {
		handleErr(fmt.Errorf("failed to calculate payment: %w", err))
		return
	}

	// Process payment items (save and forward)
	if err := n.paymentProcessor.ProcessPaymentItems(&req.Contract, items, req.UniqueID); err != nil {
		handleErr(fmt.Errorf("failed to process payment items: %w", err))
		return
	}

	resp := contracts.ContractUsageResponse{}
	n.sendReply(msg, resp)
}

func (n *Node) handleContractChainVerification(msg actor.Envelope) {
	defer msg.Discard()

	resp := contracts.ContractChainVerificationResponse{}

	var req contracts.ContractChainVerificationRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		resp.Error = fmt.Sprintf("failed to unmarshal request: %v", err)
		n.sendReply(msg, resp)
		return
	}

	// Verify that the caller is the Provider mentioned in the request
	// This ensures only the actual Provider can verify chains involving itself
	providerDID, err := did.FromString(req.ProviderDID)
	if err != nil {
		resp.Error = fmt.Sprintf("invalid provider DID: %v", err)
		n.sendReply(msg, resp)
		return
	}

	// Security check: Only the Provider mentioned in the request can verify the chain
	if msg.From.DID != providerDID {
		resp.Error = fmt.Sprintf("caller DID (%s) does not match ProviderDID in request (%s)",
			msg.From.DID.String(), req.ProviderDID)
		n.sendReply(msg, resp)
		return
	}

	orchestratorDID, err := did.FromString(req.OrchestratorDID)
	if err != nil {
		resp.Error = fmt.Sprintf("invalid orchestrator DID: %v", err)
		n.sendReply(msg, resp)
		return
	}

	// Step 1: Verify Contract A (Orchestrator ↔ Organization)
	// Use the provided ContractADID to get the contract
	contractA, err := n.contractStore.GetContract(req.ContractDID)
	if err != nil {
		resp.Error = fmt.Sprintf("contract A not found: %v", err)
		n.sendReply(msg, resp)
		return
	}

	// Validate Contract A participants match the provided DIDs
	provStr := contractA.ContractParticipants.Provider.String()
	reqStr := contractA.ContractParticipants.Requestor.String()
	orchStr := orchestratorDID.String()

	// Check that Contract A is between Orchestrator and Organization
	if reqStr != orchStr && provStr == req.ProviderDID {
		resp.Error = "contract A participants do not match orchestrator and organization"
		n.sendReply(msg, resp)
		return
	}

	// Ensure Contract A is active
	if contractA.CurrentState != contracts.ContractAccepted && contractA.CurrentState != contracts.ContractActive {
		resp.Error = fmt.Sprintf("contract A is not in active state: %s", contractA.CurrentState)
		n.sendReply(msg, resp)
		return
	}

	// Step 2: Find Contract B (Organization ↔ Provider)
	// The orchestrator specifies Contract A (head contract) which includes the Organization DID.
	// The provider finds Contract B by matching the Organization DID from Contract A.
	// This handles the case where a provider has contracts with multiple organizations:
	// the provider will find the correct Contract B based on which organization is specified
	// in the head contract (Contract A).
	//
	// Example: If Provider has contracts with Org1 and Org2:
	// - Orchestrator specifies Contract A with Org1 → Provider finds Contract B with Org1
	// - Orchestrator specifies Contract A with Org2 → Provider finds Contract B with Org2
	contractB, err := n.contractStore.FindContractByParticipants(contractA.ContractParticipants.Provider, providerDID)
	if err != nil {
		resp.Error = fmt.Sprintf("no active contract found between organization and provider: %v", err)
		n.sendReply(msg, resp)
		return
	}

	// Step 3: Validate both contracts are in acceptable state
	validA := contractA.CurrentState == contracts.ContractAccepted || contractA.CurrentState == contracts.ContractActive
	validB := contractB.CurrentState == contracts.ContractAccepted || contractB.CurrentState == contracts.ContractActive

	if !validA || !validB {
		resp.Error = fmt.Sprintf("contract chain invalid: Contract A state=%s, Contract B state=%s",
			contractA.CurrentState, contractB.CurrentState)
		n.sendReply(msg, resp)
		return
	}

	// Chain is valid
	resp.Valid = true
	resp.OrganizationDID = contractA.ContractParticipants.Provider.String()
	resp.OrchestratorContract = contractA
	resp.ProviderContract = contractB

	n.sendReply(msg, resp)
}

// convertRequestToUsageData converts ContractUsageRequest to UsageData format
func (n *Node) convertRequestToUsageData(req *contracts.ContractUsageRequest) (*contracts.UsageData, error) {
	paymentModel := req.Contract.PaymentDetails.PaymentModel

	switch paymentModel {
	case contracts.PayPerAllocation:
		return &contracts.UsageData{
			ContractDID:  req.Contract.ContractDID,
			PaymentModel: paymentModel,
			Data:         req.Usages, // Simple count
		}, nil

	case contracts.PayPerDeployment:
		return &contracts.UsageData{
			ContractDID:  req.Contract.ContractDID,
			PaymentModel: paymentModel,
			Data:         req.Usages, // Simple count
		}, nil

	case contracts.PayPerTimeUtilization:
		if req.TimeUtilization == nil {
			return nil, errors.New("time_utilization is required for pay_per_time_utilization payment model")
		}
		return &contracts.UsageData{
			ContractDID:  req.Contract.ContractDID,
			PaymentModel: paymentModel,
			Data:         req.TimeUtilization,
		}, nil

	case contracts.PayPerResourceUtilization:
		if req.ResourceUtilization == nil {
			return nil, errors.New("resource_utilization is required for pay_per_resource_utilization payment model")
		}
		return &contracts.UsageData{
			ContractDID:  req.Contract.ContractDID,
			PaymentModel: paymentModel,
			Data:         req.ResourceUtilization,
		}, nil

	case contracts.FixedRental:
		if req.FixedRentalDetails == nil {
			return nil, errors.New("fixed_rental_details is required for fixed_rental payment model")
		}
		return &contracts.UsageData{
			ContractDID:  req.Contract.ContractDID,
			PaymentModel: paymentModel,
			Data:         req.FixedRentalDetails,
		}, nil

	case contracts.Periodic:
		if req.PeriodicDetails == nil {
			return nil, errors.New("periodic_details is required for periodic payment model")
		}
		// Edge case: No deployments - return nil to skip processing
		if len(req.PeriodicDetails.Deployments) == 0 {
			log.Infow("received periodic invoice request with no deployments (zero runtime), skipping payment processing",
				"contract_did", req.Contract.ContractDID,
				"period_start", req.PeriodicDetails.PeriodStart,
				"period_end", req.PeriodicDetails.PeriodEnd)
			return nil, nil
		}
		return &contracts.UsageData{
			ContractDID:  req.Contract.ContractDID,
			PaymentModel: paymentModel,
			Data:         req.PeriodicDetails,
		}, nil

	default:
		return nil, fmt.Errorf("unsupported payment model: %s", paymentModel)
	}
}

// isHeadContractFromRequest determines if a contract is a Head Contract in a chain
// This can be enhanced based on actual contract creation context
// For now, returns false (contracts without metadata are treated as P2P)
func isHeadContractFromRequest(request contracts.CreateContractRequest) bool {
	return request.Metadata[contracts.ContractChainRoleMetadataKey] == contracts.ContractChainRoleHead
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

// handleGetPaymentQuote handles requests for payment quotes
func (n *Node) handleGetPaymentQuote(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorw("handle_get_payment_quote",
			"labels", []string{string(observability.LabelContract)},
			"error", err)
		n.sendReply(msg, contracts.ContractGetPaymentQuoteResponse{Error: err.Error()})
	}

	var req contracts.ContractGetPaymentQuoteRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal payment quote request: %w", err))
		return
	}

	// Get transaction
	tx, err := n.transactionStore.GetTransactionByUniqueID(req.UniqueID)
	if err != nil {
		handleErr(fmt.Errorf("failed to get transaction: %w", err))
		return
	}

	// Check if conversion is needed
	if !tx.RequiresConversion || tx.PricingCurrency == "" {
		handleErr(fmt.Errorf("transaction does not require conversion"))
		return
	}

	// Check if there's already an active quote for this transaction
	existingQuote, err := n.paymentQuoteStore.HasActiveQuote(req.UniqueID)
	if err != nil {
		handleErr(fmt.Errorf("failed to check for existing quote: %w", err))
		return
	}
	if existingQuote != nil {
		handleErr(fmt.Errorf("active quote already exists for this transaction (quote_id: %s). Please cancel the existing quote before creating a new one", existingQuote.QuoteID))
		return
	}

	// Get payment currency from transaction addresses
	paymentCurrency := "NTX"
	if len(tx.ToAddress) > 0 {
		paymentCurrency = tx.ToAddress[0].Currency
	}

	// Perform real-time conversion
	if n.priceConverter == nil {
		handleErr(fmt.Errorf("price converter not configured"))
		return
	}

	ctx := context.Background()
	oracle := n.priceConverter.GetOracle()
	if oracle == nil {
		handleErr(fmt.Errorf("price oracle not available"))
		return
	}

	convertedAmount, err := oracle.ConvertAmount(ctx, tx.OriginalAmount, tx.PricingCurrency, paymentCurrency)
	if err != nil {
		handleErr(fmt.Errorf("failed to convert amount: %w", err))
		return
	}

	// Get exchange rate
	rate, err := oracle.GetPrice(ctx, tx.PricingCurrency, paymentCurrency)
	if err != nil {
		handleErr(fmt.Errorf("failed to get exchange rate: %w", err))
		return
	}

	// Generate quote ID
	quoteID := fmt.Sprintf("quote_%s_%d", req.UniqueID, time.Now().UnixNano())

	// Get quote TTL from config (default: 2 minutes)
	quoteTTL := 2 * time.Minute
	if n.dmsConfig.CoinMarketCap.QuoteTTL != "" {
		if parsedTTL, err := time.ParseDuration(n.dmsConfig.CoinMarketCap.QuoteTTL); err == nil {
			quoteTTL = parsedTTL
		}
	}

	// Create quote
	quote := payment_quote.PaymentQuote{
		QuoteID:         quoteID,
		UniqueID:        req.UniqueID,
		OriginalAmount:  tx.OriginalAmount,
		ConvertedAmount: convertedAmount,
		PricingCurrency: tx.PricingCurrency,
		PaymentCurrency: paymentCurrency,
		ExchangeRate:    fmt.Sprintf("%.8f", rate),
		CreatedAt:       time.Now(),
		ExpiresAt:       time.Now().Add(quoteTTL),
		Used:            false,
	}

	// Store quote
	if err := n.paymentQuoteStore.CreateQuote(quote); err != nil {
		handleErr(fmt.Errorf("failed to create quote: %w", err))
		return
	}

	// Return response
	resp := contracts.ContractGetPaymentQuoteResponse{
		QuoteID:         quoteID,
		OriginalAmount:  tx.OriginalAmount,
		ConvertedAmount: convertedAmount,
		PricingCurrency: tx.PricingCurrency,
		PaymentCurrency: paymentCurrency,
		ExchangeRate:    fmt.Sprintf("%.8f", rate),
		ExpiresAt:       quote.ExpiresAt,
	}

	n.sendReply(msg, resp)
}

// handleValidatePaymentQuote handles validation requests for payment quotes
func (n *Node) handleValidatePaymentQuote(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		n.sendReply(msg, contracts.ContractValidatePaymentQuoteResponse{
			Valid: false,
			Error: err.Error(),
		})
	}

	var req contracts.ContractValidatePaymentQuoteRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal validate quote request: %w", err))
		return
	}

	// Validate quote (checks expiration and usage)
	quote, err := n.paymentQuoteStore.ValidateQuote(req.QuoteID)
	if err != nil {
		handleErr(fmt.Errorf("quote validation failed: %w", err))
		return
	}

	// Return validation result with quote details
	resp := contracts.ContractValidatePaymentQuoteResponse{
		Valid:           true,
		QuoteID:         quote.QuoteID,
		OriginalAmount:  quote.OriginalAmount,
		ConvertedAmount: quote.ConvertedAmount,
		PricingCurrency: quote.PricingCurrency,
		PaymentCurrency: quote.PaymentCurrency,
		ExchangeRate:    quote.ExchangeRate,
		ExpiresAt:       quote.ExpiresAt,
	}

	n.sendReply(msg, resp)
}

// handleCancelPaymentQuote handles cancellation requests for payment quotes
func (n *Node) handleCancelPaymentQuote(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		n.sendReply(msg, contracts.ContractCancelPaymentQuoteResponse{Error: err.Error()})
	}

	var req contracts.ContractCancelPaymentQuoteRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal cancel quote request: %w", err))
		return
	}

	// Validate quote exists
	quote, err := n.paymentQuoteStore.GetQuote(req.QuoteID)
	if err != nil {
		handleErr(fmt.Errorf("quote not found: %w", err))
		return
	}

	// Check if already used
	if quote.Used {
		handleErr(fmt.Errorf("quote already used"))
		return
	}

	// Invalidate quote (mark as used)
	if err := n.paymentQuoteStore.InvalidateQuote(req.QuoteID); err != nil {
		handleErr(fmt.Errorf("failed to invalidate quote: %w", err))
		return
	}

	n.sendReply(msg, contracts.ContractCancelPaymentQuoteResponse{})
}
