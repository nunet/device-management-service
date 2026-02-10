// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/gateway/provider"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/tokenomics/contracts"
	"gitlab.com/nunet/device-management-service/types"
)

const bidStateTimeout = 5 * time.Minute

func (n *Node) getExecutor(execType jobs.AllocationExecutor) (executorMetadata, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	e, ok := n.executors[string(execType)]
	if !ok {
		return executorMetadata{}, errors.New("executor not available")
	}

	return e, nil
}

func (n *Node) storeBid(eid string, nonce uint64, req jobtypes.BidRequest) {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.bids[eid] = &bidState{
		expire:  time.Now().Add(bidStateTimeout),
		nonce:   nonce,
		request: req,
	}

	n.answeredBids[eid] = append(n.answeredBids[eid], nonce)
}

func (n *Node) getBid(eid string) (*bidState, bool) {
	n.lock.Lock()
	defer n.lock.Unlock()

	b, exists := n.bids[eid]

	return b, exists
}

func (n *Node) bidAnswered(eid string, nonce uint64) bool {
	n.lock.Lock()
	defer n.lock.Unlock()

	for e, n := range n.answeredBids {
		if e == eid && slices.Contains(n, nonce) {
			return true
		}
	}
	return false
}

func (n *Node) location() jobtypes.Location {
	n.lock.RLock()
	defer n.lock.RUnlock()

	return jobtypes.Location{
		Continent: n.hostLocation.Continent,
		Country:   n.hostLocation.Country,
		City:      n.hostLocation.City,
	}
}

func (n *Node) verifyContract(bidContracts map[string]types.ContractConfig, orchestratorDID did.DID) error {
	for contractKey, contractConfig := range bidContracts {
		// Chain verification mode (Contract A: Orchestrator ↔ Organization)
		// Provider will verify Contract B (Organization ↔ Provider) via chain verification
		if err := n.verifyContractChain(contractConfig, orchestratorDID, contractKey); err != nil {
			log.Debugw("contract_chain_verification_failed",
				"labels", string(observability.LabelDeployment),
				"contract_key", contractKey,
				"error", err,
				"reason", "likely not a chain contract",
			)
			log.Debugw("performing p2p contract verification since contract chain verification failed",
				"labels", string(observability.LabelDeployment),
				"contract_key", contractKey,
			)
			// Traditional P2P verification (provider is directly in the contract)
			if err := n.verifyP2PContract(contractConfig); err != nil {
				return fmt.Errorf("contract verification failed for %s: %w", contractKey, err)
			}
		} else {
			log.Debugw("contract_verification_success",
				"labels", string(observability.LabelDeployment),
				"contract_key", contractKey,
				"contract_chain", true)
		}
	}
	return nil
}

// verifyP2PContract performs traditional P2P contract verification
func (n *Node) verifyP2PContract(contractConfig types.ContractConfig) error {
	hostDID, err := did.FromString(contractConfig.Host)
	if err != nil {
		return fmt.Errorf("failed to get contracts host did: %w", err)
	}

	pubKey, err := did.PublicKeyFromDID(hostDID)
	if err != nil {
		return fmt.Errorf("failed to get contracts host public key from did: %w", err)
	}

	pid, err := peer.IDFromPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("failed to get peer id: %w", err)
	}

	contractActorDID, err := did.FromString(contractConfig.DID)
	if err != nil {
		return fmt.Errorf("failed to get contracts actor did: %w", err)
	}

	pubKeyContractActor, err := did.PublicKeyFromDID(contractActorDID)
	if err != nil {
		return fmt.Errorf("failed to get contracts actor public key from did: %w", err)
	}

	destination, err := actor.HandleFromPublicKeyWithInboxAddress(pubKeyContractActor, contractConfig.DID, pid.String())
	if err != nil {
		return fmt.Errorf("failed to get contracts host handle: %w", err)
	}

	req := contracts.ContractValidateRequest{ContractDID: contractConfig.DID}
	reply, err := n.invokeBehaviour(destination, behaviors.ContractValidationBehavior, req, invokeMessageTimeout)
	if err != nil {
		return fmt.Errorf("failed to send message to contract host: %w", err)
	}

	var respEnvelope contracts.ContractValidateResponse
	err = json.Unmarshal(reply.Message, &respEnvelope)
	if err != nil {
		return fmt.Errorf("failed to unmarshal contract hosts response payload: %w", err)
	}

	if !respEnvelope.Valid {
		return fmt.Errorf("contract is invalid")
	}

	return nil
}

// verifyContractChain performs contract chain verification
// contractConfig contains Contract A (head contract: Orchestrator ↔ Organization)
// The orchestrator specifies the head contract, and the provider finds Contract B
// with the same organization specified in the head contract
func (n *Node) verifyContractChain(contractConfig types.ContractConfig, orchestratorDID did.DID, contractKey string) error {
	hostDID, err := did.FromString(contractConfig.Host)
	if err != nil {
		return fmt.Errorf("failed to parse host DID: %w", err)
	}

	// Get provider DID (self)
	providerDID := n.actor.Handle().DID

	pubKeyContractHost, err := did.PublicKeyFromDID(hostDID)
	if err != nil {
		return fmt.Errorf("failed to get contract actor public key: %w", err)
	}

	pid, err := peer.IDFromPublicKey(pubKeyContractHost)
	if err != nil {
		return fmt.Errorf("failed to get peer id: %w", err)
	}

	// For chain verification, we call the behavior on the ContractActor (not the node)
	// The ContractActor will find Contract A (using contractConfig.DID) and Contract B (Org ↔ Provider)
	destination, err := actor.HandleFromPeerID(pid.String())
	if err != nil {
		return fmt.Errorf("failed to construct contract actor handle: %w", err)
	}

	req := contracts.ContractChainVerificationRequest{
		SolutionEnablerDID: hostDID.String(),
		ContractDID:        contractConfig.DID, // Contract DID from config
		OrchestratorDID:    orchestratorDID.String(),
		ProviderDID:        providerDID.String(),
	}

	reply, err := n.invokeBehaviour(destination, behaviors.ContractChainVerificationBehavior, req, invokeMessageTimeout)
	if err != nil {
		return fmt.Errorf("failed to invoke chain verification: %w", err)
	}

	var resp contracts.ContractChainVerificationResponse
	err = json.Unmarshal(reply.Message, &resp)
	if err != nil {
		return fmt.Errorf("failed to unmarshal chain verification response: %w", err)
	}

	if !resp.Valid {
		return fmt.Errorf("contract chain verification failed: %s", resp.Error)
	}

	log.Infow("contract chain verification successful",
		"labels", string(observability.LabelDeployment),
		"contract_key", contractKey,
		"organization_did", resp.OrganizationDID,
		"orchestrator_contract", resp.OrchestratorContract.ContractDID,
		"provider_contract", resp.ProviderContract.ContractDID)

	return nil
}

// gateway logic to decide a bid or not goes here
// we keep all the restrictions and contrains here as it is for normal bid
func (n *Node) handleBidRequest(msg actor.Envelope) {
	defer msg.Discard()

	// ignore bid request from self if broadcast
	// only accept self bid if own peer specified on ensemble
	if msg.IsBroadcast() &&
		n.actor.Handle().Address.HostID == msg.From.Address.HostID {
		return
	}

	log.Infow(
		"got a bid request from actor",
		"labels", string(observability.LabelDeployment),
		"from", msg.From.Address,
	)

	// if not a gateway check onboarded
	if !n.dmsConfig.General.ComputeGateway {
		if onboarded := n.onboarding.IsOnboarded(); !onboarded {
			log.Debugw(
				"node_not_onboarded_ignoring_bid_request",
				"labels", string(observability.LabelDeployment),
			)
			return
		}
	}

	var request jobtypes.EnsembleBidRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugw(
			"unmarshal_bid_request_error",
			"labels", string(observability.LabelDeployment),
			"error", err,
		)
		return
	}

	log.Infow(
		"bid_request",
		"labels", string(observability.LabelDeployment),
		"from", msg.From.Address,
		"orchestratorID", request.ID,
	)

	// metric
	if m := observability.BidReceived; m != nil {
		m.Add(n.ctx, 1, metric.WithAttributes(
			observability.AttrDID,
			attribute.String("orchestratorID", request.ID),
		))
	}

	if n.dmsConfig.Job.RequireContractsForDeployment {
		// contracts are global at ensemble level so they apply to all nodes
		if len(request.Request) > 0 {
			if len(request.Request[0].V1.Contracts) == 0 {
				log.Debugw(
					"bid_request_missing_contracts_for_deployment",
					"labels", string(observability.LabelDeployment),
					"ensemble_id", request.ID,
				)
				return
			}
		}
	}

	// contracts are global at ensemble level so they apply
	// to all nodes
	if len(request.Request) > 0 {
		if len(request.Request[0].V1.Contracts) > 0 {
			// Extract orchestrator DID from message envelope
			orchestratorDID := msg.From.DID
			err := n.verifyContract(request.Request[0].V1.Contracts, orchestratorDID)
			if err != nil {
				log.Errorw(
					"contract_verification_error",
					"labels", string(observability.LabelDeployment),
					"error", err,
				)
				return
			}

			log.Infow("contract_verification_success",
				"contracts", request.Request[0].V1.Contracts,
				"labels", string(observability.LabelDeployment),
			)
		} else {
			log.Debugw(
				"contracts_empty",
				"labels", string(observability.LabelDeployment),
			)
		}
	}

	machineResources, err := n.hardware.GetMachineResources()
	if err != nil {
		log.Debugw(
			"machine_resources_retrieval_error",
			"labels", string(observability.LabelDeployment),
			"error", err,
		)
		return
	}

	// randomize the order of bid request checks
	rand.Shuffle(len(request.Request), func(i, j int) {
		request.Request[i], request.Request[j] = request.Request[j], request.Request[i]
	})

	// find the first bid request that matches
	var toAnswer jobtypes.BidRequest
	var found bool

loop:
	for _, v := range request.Request {
		// check if it is a V1 request
		if v.V1 == nil {
			log.Debugw("bid_request_not_v1",
				"labels", string(observability.LabelDeployment))
			continue
		}

		answered := n.bidAnswered(request.ID, request.Nonce)
		if answered {
			log.Debugf("bid already answered: ensembleID: %s, nonce: %d", request.ID, request.Nonce)
			return
		}

		// check if we are excluded
		hostID := n.actor.Handle().Address.HostID
		for _, p := range request.PeerExclusion {
			if p == hostID {
				log.Debugw("bid_request_excluded_peer",
					"labels", string(observability.LabelDeployment),
					"hostID", hostID)
				continue loop
			}
		}

		constraints := v.V1.Location
		if !n.location().Satisfies(constraints) {
			log.Debugw("bid_request_location_constraints_not_satisfied",
				"labels", string(observability.LabelDeployment),
				"nodeID", v.V1.NodeID,
				"ourLocation", n.location(),
				"constraints", constraints,
			)
			continue loop
		}

		// if the desired executable is not found stop
		if !n.dmsConfig.General.ComputeGateway {
			for _, e := range v.V1.Executors {
				executor, err := n.getExecutor(e)
				if err != nil {
					log.Debugw("executor_unavailable",
						"labels", string(observability.LabelDeployment),
						"executor", e,
						"error", err)
					continue loop
				}

				if executor.executionType == jobtypes.ExecutorDocker {
					if v.V1.GeneralRequirements.PrivilegedDocker {
						if !n.dmsConfig.AllowPrivilegedDocker {
							log.Debugw("privileged_docker_not_allowed",
								"labels", string(observability.LabelDeployment))
							continue loop
						}
					}
				}
			}
		}

		if !n.dmsConfig.General.ComputeGateway {
			comparisonResult, err := machineResources.Compare(v.V1.Resources)
			if err != nil {
				log.Debugw("compare_machine_resources_error",
					"labels", string(observability.LabelDeployment),
					"error", err)
				continue loop
			}

			if comparisonResult != types.Better {
				log.Debugw("resource_not_better",
					"labels", string(observability.LabelDeployment),
					"comparisonResult", comparisonResult)
				continue
			}

		} else {
			// make this concurrent
			foundServer := int32(0)
			allProviders := n.serverProviderRegistry.All()
			log.Debugf("server providers %d", len(allProviders))

			var wg sync.WaitGroup
			for _, pp := range allProviders {
				wg.Add(1)
				go func(pp provider.Provider) {
					defer wg.Done()

					if atomic.LoadInt32(&foundServer) == 1 {
						return
					}

					plans, err := pp.ListPlans(n.ctx)
					if err != nil {
						return
					}

					_, err = pp.SelectMatchingPlan(plans, v.V1.Resources)
					if err != nil {
						return
					}
					atomic.StoreInt32(&foundServer, 1)
				}(pp)
			}

			wg.Wait()

			if atomic.LoadInt32(&foundServer) == 0 {
				log.Debug("couldn't find servers to provision")
				return
			}
		}

		found = true
		toAnswer = v
		break
	}

	if !found {
		log.Debugw("bid_requirements_not_satisfied",
			"labels", string(observability.LabelDeployment))
		return
	}

	if !n.dmsConfig.General.ComputeGateway {
		if err := n.allocator.CheckAvailability(toAnswer.V1.PublicPorts.Static, toAnswer.V1.PublicPorts.Dynamic, toAnswer.V1.Resources); err != nil {
			log.Debugw("no_resource_availability_for_bid",
				"labels", string(observability.LabelDeployment),
				"nodeID", toAnswer.V1.NodeID,
				"staticPorts", toAnswer.V1.PublicPorts.Static,
				"dynamicPorts", toAnswer.V1.PublicPorts.Dynamic,
				"resources", toAnswer.V1.Resources,
				"error", err)
			return
		}
	}

	log.Debugw("signing_bid_with_node_identity",
		"labels", string(observability.LabelDeployment),
		"DID", n.actor.Security().DID())

	provider, err := n.rootCap.Trust().GetProvider(n.actor.Security().DID())
	if err != nil {
		log.Debugw("provider_retrieval_error",
			"labels", string(observability.LabelDeployment),
			"error", err)
		return
	}
	log.Debugw("signing_bid_with_provider",
		"labels", string(observability.LabelDeployment),
		"providerDID", provider.DID())

	bid := jobtypes.Bid{
		V1: &jobtypes.BidV1{
			EnsembleID: request.ID,
			NodeID:     toAnswer.V1.NodeID,
			Peer:       n.hostID,
			PubAddress: n.publicIP.String(),
			Location:   n.location(),
			Handle:     n.actor.Handle(),
		},
	}

	// indicate if its a promise bid
	if n.dmsConfig.General.ComputeGateway {
		bid.V1.PromiseBid = true
	}

	if err := bid.Sign(provider); err != nil {
		log.Debugw("bid_signing_error",
			"labels", string(observability.LabelDeployment),
			"error", err)
		return
	}

	log.Infow("sending_bid_response",
		"labels", string(observability.LabelDeployment),
		"ensembleID", request.ID,
		"nodeID", toAnswer.V1.NodeID,
		"peerID", n.hostID,
		"nonce", request.Nonce)

	n.sendReply(msg, bid)
	n.storeBid(request.ID, request.Nonce, toAnswer)

	// metric
	if m := observability.BidAccepted; m != nil {
		m.Add(n.ctx, 1, metric.WithAttributes(
			observability.AttrDID,
			attribute.String("orchestratorID", request.ID),
		))
	}
}
