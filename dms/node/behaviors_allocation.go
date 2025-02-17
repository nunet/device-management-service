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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"

	"github.com/google/uuid"
	"gitlab.com/nunet/device-management-service/executor/docker"

	"gitlab.com/nunet/device-management-service/lib/crypto"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/types"
)

func (n *Node) handleSubnetCreate(msg actor.Envelope) {
	defer msg.Discard()

	var request jobs.SubnetCreateRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		return
	}

	resp := jobs.SubnetCreateResponse{}
	err := n.network.CreateSubnet(context.Background(), request.SubnetID, request.RoutingTable)
	if err != nil {
		resp.Error = err.Error()
		n.sendReply(msg, resp)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) handleSubnetDestroy(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error destroying subnet: %s", err)
		n.sendReply(msg, jobs.SubnetDestroyResponse{Error: err.Error()})
	}

	var request jobs.SubnetDestroyRequest
	resp := jobs.SubnetDestroyResponse{}
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling subnet destroy: %s", err))
		return
	}

	err := n.network.DestroySubnet(request.SubnetID)
	if err != nil {
		handleErr(err)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) addEnsembleBehaviors(ensembleID string) error {
	dmsBehaviors := map[string]struct {
		fn   func(actor.Envelope)
		opts []actor.BehaviorOption
	}{
		fmt.Sprintf(behaviors.SubnetCreateBehavior.DynamicTemplate, ensembleID): {
			fn: n.handleSubnetCreate,
		},
		fmt.Sprintf(behaviors.SubnetDestroyBehavior.DynamicTemplate, ensembleID): {
			fn: n.handleSubnetDestroy,
		},
		fmt.Sprintf(behaviors.AllocationLogsBehavior, ensembleID): {
			fn: n.handleAllocationLogs,
		},
		fmt.Sprintf(behaviors.AllocationShutdownBehavior, ensembleID): {
			fn: n.handleAllocationShutdown,
		},
	}
	for behavior, handler := range dmsBehaviors {
		if err := n.actor.AddBehavior(behavior, handler.fn, handler.opts...); err != nil {
			return fmt.Errorf("adding %s behavior: %w", behavior, err)
		}
	}
	return nil
}

func (n *Node) grantCaps(orchestrator did.DID, aud did.DID, caps []ucan.Capability) error {
	tokens, err := n.rootCap.Grant(
		ucan.Delegate,
		orchestrator,
		aud,
		[]string{},
		actor.MakeExpiry(grantAllocationCapsFreq),
		1,
		caps,
	)
	if err != nil {
		return fmt.Errorf("create granting token for audience %s caps: %w", aud, err)
	}

	err = n.rootCap.AddRoots([]did.DID{}, tokens, ucan.TokenList{}, ucan.TokenList{})
	if err != nil {
		return fmt.Errorf("add roots for audience %s: %w", aud, err)
	}

	return nil
}

// createAllocation creates an allocation
func (n *Node) createAllocation(allocationID string, job jobs.Job, supervisor actor.Handle) (*jobs.Allocation, error) {
	priv, _, err := crypto.GenerateKeyPair(crypto.Ed25519)
	if err != nil {
		return nil, fmt.Errorf("generate keypair for allocation job %s: %w", allocationID, err)
	}

	allocActor, err := n.createChildActor(priv, allocationID, supervisor)
	if err != nil {
		return nil, fmt.Errorf("create allocation actor: %w", err)
	}

	executor, err := createExecutor(context.Background(), n.fs, job.Execution.Type)
	if err != nil {
		return nil, fmt.Errorf("create executor: %w", err)
	}

	allocation, err := n.allocator.Allocate(context.Background(), allocationID, allocActor, job, executor)
	if err != nil {
		return nil, fmt.Errorf("allocate resources: %w", err)
	}

	return allocation, nil
}

func (n *Node) createAllocations(
	orchestrator did.DID,
	ensembleID string,
	allocations map[string]jobs.AllocationDeploymentConfig,
	supervisor actor.Handle,
) (map[string]actor.Handle, error) {
	allocHandlesByName := make(map[string]actor.Handle, len(allocations))
	allocationIDs := make([]string, 0, len(allocations))
	for allocationName, allocationConfig := range allocations {
		allocationID := types.ConstructAllocationID(ensembleID, allocationName)
		// TODO: check if the allocation ID exists

		allocation, err := n.createAllocation(
			allocationID,
			jobs.Job{
				Resources:        allocationConfig.Resources,
				Execution:        allocationConfig.Execution,
				ProvisionScripts: allocationConfig.ProvisionScripts,
				Volume:           allocationConfig.Volume,
			},
			supervisor,
		)
		if err != nil {
			return nil, fmt.Errorf("create allocation %s: %w", allocationID, err)
		}

		allocHandlesByName[allocationName] = allocation.Actor.Handle()
		allocationIDs = append(allocationIDs, allocation.ID)

		// node grants subnet create/destroy caps to the orchestrator
		if err := n.grantCaps(orchestrator, n.actor.Handle().DID, []ucan.Capability{
			ucan.Capability(fmt.Sprintf(behaviors.EnsembleNamespace, ensembleID)),
		}); err != nil {
			return nil, fmt.Errorf("grant node caps: %w", err)
		}

		// allocation grants subnet manage caps to the orchestrator
		allocDID, err := did.FromID(allocation.Actor.Handle().ID)
		if err != nil {
			return nil, fmt.Errorf("get did from id: %w", err)
		}

		if err := n.grantCaps(orchestrator, allocDID, []ucan.Capability{
			behaviors.AllocationNamespace,
		}); err != nil {
			return nil, fmt.Errorf("grant allocation caps: %w", err)
		}

		// refresh allocation caps grants periodically
		go func() {
			ticker := time.NewTicker(grantAllocationCapsFreq)
			defer ticker.Stop()

			for allocation.Status(context.TODO()).Status != jobs.AllocationStopped {
				select {
				case <-n.ctx.Done():
					return
				case <-ticker.C:
					// node grants subnet create/destroy caps to the orchestrator
					if err := n.grantCaps(orchestrator, n.actor.Handle().DID, []ucan.Capability{
						ucan.Capability(fmt.Sprintf(behaviors.EnsembleNamespace, ensembleID)),
					}); err != nil {
						log.Warnf("grant node caps: %v", err)
					}

					// allocation grants subnet manage caps to the orchestrator
					if err := n.grantCaps(orchestrator, allocDID, []ucan.Capability{
						behaviors.AllocationNamespace,
					}); err != nil {
						log.Warnf("grant allocation caps: %v", err)
					}
				}
			}
		}()
	}

	// Start monitoring allocations
	go n.monitorEnsembleAllocations(ensembleID, allocationIDs)

	log.Infof("Finished createAllocations for ensembleID: %s", ensembleID)
	return allocHandlesByName, nil
}

func (n *Node) handleAllocationDeployment(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error handling allocation deployment: %s", err)
		n.sendReply(msg, jobs.AllocationDeploymentResponse{Error: err.Error()})
	}

	var request jobs.AllocationDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(err)
		return
	}

	resp := jobs.AllocationDeploymentResponse{}
	if err := n.addEnsembleBehaviors(request.EnsembleID); err != nil {
		handleErr(fmt.Errorf("failed to register dynamic behaviors: %s", err))
		return
	}

	allocations, err := n.createAllocations(
		msg.From.DID,
		request.EnsembleID,
		request.Allocations,
		msg.From,
	)
	if err != nil {
		handleErr(err)
		return
	}

	resp.OK = true
	resp.Allocations = allocations
	n.sendReply(msg, resp)
}

type AllocationShutdownRequest struct {
	AllocationID string
}

type AllocationShutdownResponse struct {
	OK    bool
	Error string
}

func (n *Node) handleAllocationShutdown(msg actor.Envelope) {
	log.Debugf("handling allocation shutdown request from %s", msg.From.DID)
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error handling allocation shutdown request: %s", err)
		n.sendReply(msg, AllocationShutdownResponse{Error: err.Error()})
	}

	var request AllocationShutdownRequest
	resp := AllocationShutdownResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling allocation shutdown request: %s", err))
		return
	}

	err := n.allocator.Release(context.Background(), request.AllocationID)
	if err != nil {
		handleErr(err)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

func ensembleIDFromBehavior(b string) (string, error) {
	parts := strings.Split(b, "/")
	if len(parts) > 3 {
		return parts[3], nil
	}
	return "", fmt.Errorf("invalid ensemble behavior: %s", b)
}

// TODO: instead of having a behavior to get logs from a FINISHED allocation,
// we may allow only getting logs from ONGOING allocations.
// For finite allocations, it could send the logs back to the orchestrator
// if specified on the ensemble.
func (n *Node) handleAllocationLogs(msg actor.Envelope) {
	defer msg.Discard()
	log.Infof("behavior get logs invoked by: %+v", msg.From)

	handleErr := func(err error) {
		log.Errorf("error getting allocation logs: %s", err)
		n.sendReply(msg, jobs.AllocationLogsResponse{Error: err.Error()})
	}

	var resp jobs.AllocationLogsResponse
	ensembleID, err := ensembleIDFromBehavior(msg.Behavior)
	if err != nil {
		handleErr(fmt.Errorf("error getting ensemble ID from behavior %s: %s", msg.Behavior, err))
		return
	}

	var req jobs.AllocationLogsRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("error unmarshalling allocation logs request: %s", err))
		return
	}

	allocID := types.ConstructAllocationID(ensembleID, req.AllocName)
	resultsDir := filepath.Join(n.dmsConfig.WorkDir, "jobs", allocID)

	stdout, err := n.fs.ReadFile(filepath.Join(resultsDir, "stdout.log"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			log.Warnf("stdout file for allocation %s does not exist (ensemble: %s)", req.AllocName, ensembleID)
		} else {
			handleErr(fmt.Errorf("failed to read results file: %s", err))
			return
		}
	}

	stderr, err := n.fs.ReadFile(filepath.Join(resultsDir, "stderr.log"))
	if err != nil {
		if err == os.ErrNotExist {
			log.Debugf("stderr file for allocation %s does not exist (ensemble: %s)", req.AllocName, ensembleID)
		} else {
			handleErr(fmt.Errorf("failed to read results file: %s", err))
			return
		}
	}

	if len(stdout) == 0 && len(stderr) == 0 {
		handleErr(
			fmt.Errorf("stdout and stderr files for allocation %s are empty (ensemble: %s)",
				req.AllocName, ensembleID),
		)
		return
	}

	log.Info("sending logs for allocation: ", allocID)

	resp.Stdout = stdout
	resp.Stderr = stderr
	n.sendReply(msg, resp)
}

func createExecutor(ctx context.Context, fs afero.Afero, executionType string) (types.Executor, error) {
	switch executionType {
	case types.ExecutorTypeDocker.String():
		id := uuid.New().String()
		exec, err := docker.NewExecutor(ctx, fs, id)
		if err != nil {
			return nil, fmt.Errorf("create executor: %w", err)
		}
		return exec, nil
	default:
		return nil, fmt.Errorf("unsupported executor type: %s", executionType)
	}
}
