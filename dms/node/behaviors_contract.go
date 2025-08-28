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
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/tokenomics"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/tokenomics/store"
)

const (
	invokeMessageTimeout          = 20 * time.Second
	retryMessageTimeout           = 5
	getSignatureRetries           = 500 * 24
	waitForParticipantSigsTimeout = 30 * time.Minute

	invokeSignRequestTimeout = 2 * time.Minute
)

// handleNewContract is registered on the contract host
func (n *Node) handleNewContract(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		n.sendReply(msg, contracts.CreateContractResponseBehaviour{Error: err.Error()})
	}

	var request contracts.CreateContractRequestBehaviour
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("unmarshal create contract request: %s", err))
		return
	}

	privKey, pubKey, err := crypto.GenerateKeyPair(crypto.Ed25519)
	if err != nil {
		handleErr(fmt.Errorf("failed to generate contract keypair: %w", err))
		return
	}

	contractActor, err := tokenomics.NewContractActor(n.actor.Handle(), n.network, request.ContractParticipants, privKey, pubKey, n.contractStore)
	if err != nil {
		handleErr(fmt.Errorf("failed to create contract actor: %w", err))
		return
	}

	contractObj := contracts.NewContract(contractActor.ContractDID.URI, request)
	err = n.contractStore.Upsert(contractObj)
	if err != nil {
		handleErr(fmt.Errorf("failed to save contract: %w", err))
		return
	}

	pkBytes, err := crypto.PublicKeyToBytes(pubKey)
	if err != nil {
		handleErr(fmt.Errorf("failed to convert public key to bytes: %w", err))
		return
	}

	privKeyBytes, err := crypto.PrivateKeyToBytes(privKey)
	if err != nil {
		handleErr(fmt.Errorf("failed to convert private key to bytes: %w", err))
		return
	}

	err = n.contractStore.InsertContractKey(store.ContractKey{
		ContractDID: contractActor.ContractDID.URI,
		Key:         privKeyBytes,
	})
	if err != nil {
		handleErr(fmt.Errorf("failed to save actor private key for contract %s: %w", contractActor.ContractDID.URI, err))
		return
	}

	err = contractActor.Start()
	if err != nil {
		handleErr(fmt.Errorf("failed to start actor: %w", err))
		return
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
			err = n.contractStore.Upsert(contractObj)
			if err != nil {
				log.Errorf("failed to update contract with signatures: %w", err)
			}
		}
	}()

	n.sendReply(msg, contracts.CreateContractResponseBehaviour{
		ContractDID: contractActor.ContractDID.URI,
		PubKey:      hex.EncodeToString(pkBytes),
	})
}

// this behaviour is registered by service and compute provider
// its used to sign contracts and send back the response to solution enabler.
func (n *Node) handleContractPropose(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		n.sendReply(msg, contracts.ProposeContractResponseBehaviour{Error: err.Error()})
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

		n.sendReply(msg, contracts.ProposeContractResponseBehaviour{
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

	n.sendReply(msg, contracts.ProposeContractResponseBehaviour{
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
			var response contracts.ProposeContractResponseBehaviour
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
		n.sendReply(msg, contracts.ContractApproveLocalResponseBehaviour{Error: err.Error()})
	}

	var req contracts.ContractApproveLocalRequestBehaviour
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

	signReq := contracts.ContractSignRequestBehaviour{
		ContractDID: savedContract.ContractDID,
		Signature:   sig,
	}
	reply, err := n.invokeBehaviour(destination, behaviors.ContractSignBehavior, signReq, invokeSignRequestTimeout)
	if err != nil {
		handleErr(fmt.Errorf("failed to get invoke sign contract on contract host: %w", err))
		return
	}

	var signResp contracts.ContractSignResponseBehaviour
	if err := json.Unmarshal(reply.Message, &signReq); err != nil {
		handleErr(fmt.Errorf("failed to unmarshal contract host response: %w", err))
		return
	}

	if signResp.Error != "" {
		handleErr(fmt.Errorf("error from contract host: %w", err))
		return
	}

	n.sendReply(msg, contracts.ContractApproveLocalResponseBehaviour{
		Success: true,
	})
}

// compute provider can list incoming contracts for approval
func (n *Node) handleListIncomingContracts(msg actor.Envelope) {
	defer msg.Discard()
	handleErr := func(err error) {
		n.sendReply(msg, contracts.ContractListIncomingResponseBehaviour{Error: err.Error()})
	}

	// get all contracts
	allContracts, err := n.contractStore.GetAllContracts()
	if err != nil {
		handleErr(fmt.Errorf("failed to get all contracts: %w", err))
		return
	}

	n.sendReply(msg, contracts.ContractListIncomingResponseBehaviour{
		Contracts: allContracts,
	})
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

		contractActor, err := tokenomics.NewContractActor(n.actor.Handle(), n.network, v.ContractParticipants, privKey, pubKey, n.contractStore)
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
