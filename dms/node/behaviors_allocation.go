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

	"github.com/google/uuid"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	"gitlab.com/nunet/device-management-service/dms/jobs"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/types"
)

func (n *Node) handleSubnetCreate(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error creating subnet: %s", err)
		n.sendReply(msg, orchestrator.SubnetCreateResponse{Error: err.Error()})
	}

	var request orchestrator.SubnetCreateRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling subnet create request: %s", err))
		return
	}

	resp := orchestrator.SubnetCreateResponse{}
	err := n.network.CreateSubnet(context.Background(), request.SubnetID, request.CIDR, request.RoutingTable)
	if err != nil {
		handleErr(err)
		return
	}

	resp.OK = true
	n.sendReply(msg, resp)
}

func (n *Node) handleSubnetDestroy(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error destroying subnet: %s", err)
		n.sendReply(msg, orchestrator.SubnetDestroyResponse{Error: err.Error()})
	}

	var request orchestrator.SubnetDestroyRequest
	resp := orchestrator.SubnetDestroyResponse{}
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

func (n *Node) handleSubnetJoin(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error subnet join: %s", err)
		n.sendReply(msg, orchestrator.SubnetJoinResponse{Error: err.Error()})
	}

	var request orchestrator.SubnetJoinRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(fmt.Errorf("error unmarshalling subnet join: %s", err))
		return
	}

	resp := orchestrator.SubnetJoinResponse{}
	err := n.network.AddSubnetPeer(request.SubnetID, request.PeerID, request.IP)
	if err != nil {
		handleErr(err)
		return
	}

	err = n.network.AddSubnetDNSRecords(request.SubnetID, request.Records)
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
		fmt.Sprintf(behaviors.AllocationLogsBehavior.DynamicTemplate, ensembleID): {
			fn: n.handleAllocationLogs,
		},
		fmt.Sprintf(behaviors.AllocationShutdownBehavior.DynamicTemplate, ensembleID): {
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

// createAllocation creates an allocation
func (n *Node) createAllocation(
	allocationID string,
	allocType jobtypes.AllocationType,
	job jobs.Job, supervisor actor.Handle,
) (*jobs.Allocation, error) {
	executor, err := createExecutor(context.Background(), n.fs, job.Execution.Type)
	if err != nil {
		return nil, fmt.Errorf("create executor: %w", err)
	}

	allocActor, err := n.actor.CreateChild(allocationID, supervisor)
	if err != nil {
		return nil, fmt.Errorf("create allocation actor: %w", err)
	}

	allocation, err := n.allocator.Allocate(
		context.Background(), allocationID,
		allocType, allocActor, supervisor,
		job, executor,
	)
	if err != nil {
		return nil, fmt.Errorf("allocate: %w", err)
	}

	return allocation, nil
}

func (n *Node) createAllocations(
	ensembleID string,
	allocations map[string]jobtypes.AllocationDeploymentConfig,
	supervisor actor.Handle,
) (map[string]actor.Handle, error) {
	if len(allocations) == 0 {
		log.Errorf("no allocations to create for ensembleID: %s", ensembleID)
		return nil, fmt.Errorf("no allocations to create for ensembleID: %s", ensembleID)
	}

	if supervisor.Empty() || supervisor.DID.Empty() {
		log.Errorf("invalid supervisor handle: %+v", supervisor)
		return nil, fmt.Errorf("invalid supervisor handle")
	}

	allocHandlesByName := make(map[string]actor.Handle, len(allocations))
	allocationIDs := make([]string, 0, len(allocations))
	for allocationName, allocationConfig := range allocations {
		allocationID := types.ConstructAllocationID(ensembleID, allocationName)
		// TODO: check if the allocation ID exists

		allocation, err := n.createAllocation(
			allocationID,
			allocationConfig.Type,
			jobs.Job{
				Resources:        allocationConfig.Resources,
				Execution:        allocationConfig.Execution,
				ProvisionScripts: allocationConfig.ProvisionScripts,
				Keys:             allocationConfig.Keys,
				Volume:           allocationConfig.Volumes,
			},
			supervisor,
		)
		if err != nil {
			return nil, fmt.Errorf("create allocation %s: %w", allocationID, err)
		}

		allocHandlesByName[allocationName] = allocation.Actor.Handle()
		allocationIDs = append(allocationIDs, allocation.ID)

		// node grants subnet create/destroy caps to the orchestrator
		if err := n.actor.Security().Grant(supervisor.DID, n.actor.Handle().DID, []ucan.Capability{
			ucan.Capability(fmt.Sprintf(behaviors.EnsembleNamespace, ensembleID)),
		}, grantAllocationCapsFreq); err != nil {
			return nil, fmt.Errorf("grant node caps: %w", err)
		}

		allocDID, err := did.FromID(allocation.Actor.Handle().ID)
		if err != nil {
			return nil, fmt.Errorf("deriving allocation did: %w", err)
		}
		if err := n.actor.Security().Grant(supervisor.DID, allocDID, []ucan.Capability{
			behaviors.AllocationNamespace,
		}, grantAllocationCapsFreq); err != nil {
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
					if err := n.actor.Security().Grant(supervisor.DID, n.actor.Handle().DID, []ucan.Capability{
						ucan.Capability(fmt.Sprintf(behaviors.EnsembleNamespace, ensembleID)),
					}, grantAllocationCapsFreq); err != nil {
						log.Warnf("grant node caps: %v", err)
					}

					// allocation grants subnet manage caps to the orchestrator
					if err := n.actor.Security().Grant(supervisor.DID, allocDID, []ucan.Capability{
						behaviors.AllocationNamespace,
					}, grantAllocationCapsFreq); err != nil {
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

// TODO (wrong nomenclature): handleAllocationDeployment -> handleEnsembleDeployment
func (n *Node) handleAllocationDeployment(msg actor.Envelope) {
	defer msg.Discard()

	handleErr := func(err error) {
		log.Errorf("Error handling allocation deployment: %s", err)
		n.sendReply(msg, jobtypes.AllocationDeploymentResponse{Error: err.Error()})
	}

	var request jobtypes.AllocationDeploymentRequest
	if err := json.Unmarshal(msg.Message, &request); err != nil {
		handleErr(err)
		return
	}

	resp := jobtypes.AllocationDeploymentResponse{}
	if err := n.addEnsembleBehaviors(request.EnsembleID); err != nil {
		handleErr(fmt.Errorf("failed to register dynamic behaviors: %s", err))
		return
	}

	allocations, err := n.createAllocations(
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

func (n *Node) handleAllocationLogs(msg actor.Envelope) {
	defer msg.Discard()
	log.Infof("behavior get logs invoked by: %+v", msg.From)

	handleErr := func(err error) {
		log.Errorf("error getting allocation logs: %s", err)
		n.sendReply(msg, orchestrator.AllocationLogsResponse{Error: err.Error()})
	}

	var resp orchestrator.AllocationLogsResponse
	ensembleID, err := ensembleIDFromBehavior(msg.Behavior)
	if err != nil {
		handleErr(fmt.Errorf("error getting ensemble ID from behavior %s: %s", msg.Behavior, err))
		return
	}

	var req orchestrator.AllocationLogsRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		handleErr(fmt.Errorf("allocation logs request: %w", types.ErrUnmarshal))
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

// AllocationsListResponse represents the response for the allocations list request
type AllocationsListResponse struct {
	Allocations []jobs.AllocationInfo `json:"allocations"`
	Error       string                `json:"error,omitempty"`
}

// handleAllocationsList returns information about all running allocations
func (n *Node) handleAllocationsList(msg actor.Envelope) {
	defer msg.Discard()

	resp := AllocationsListResponse{
		Allocations: []jobs.AllocationInfo{},
	}

	allocations := n.allocator.GetAllocations()

	for _, alloc := range allocations {
		resp.Allocations = append(resp.Allocations, alloc.Info())
	}

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
