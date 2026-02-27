// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package jobs

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/orchestrator"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/tokenomics/eventhandler"
	"gitlab.com/nunet/device-management-service/tokenomics/events"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

const (
	deleteLogsAfter = 30 * time.Minute

	// Liveness reporting configuration
	livenessReportInterval = 30 * time.Second
	livenessReportTimeout  = 2 * time.Minute
	livenessMaxRetries     = 3
)

// AllocationInfo gathers useful internal information for external callers
type AllocationInfo struct {
	ID           string                  `json:"id"`
	Type         jobtypes.AllocationType `json:"type"`
	Resources    types.Resources         `json:"resources"`
	Orchestrator string                  `json:"orchestrator"` // peerID
	Status       string                  `json:"status"`
	Executor     string                  `json:"executor"`
	ExecutionID  string                  `json:"execution_id"`
	UsingPorts   []int                   `json:"using_ports,omitempty"`
	CreatedAt    time.Time               `json:"created_at"`
	StartedAt    time.Time               `json:"started_at"`
}

// Status holds the status of an allocation.
type Status struct {
	JobResources types.Resources
	Status       AllocationStatus
}

// AllocationDetails encapsulates the dependencies to the constructor.
// TODO: rename and organize general dependencies of allocaiton
type AllocationDetails struct {
	Job      Job
	NodeID   string
	SourceID string
}

// TODO: remove this struct and move everything to AllocationDetails
type Job struct {
	Resources        types.Resources
	Execution        types.SpecConfig
	ProvisionScripts map[string][]byte
	Keys             []types.AllocationKey
	Volume           []types.VolumeConfig
}

// Allocation represents an allocation
// allocationLiveness contains state for push-based liveness reporting
type allocationLiveness struct {
	enabled        bool
	interval       time.Duration
	sequenceNumber int64
	cancel         context.CancelFunc
	lock           sync.Mutex
}

type Allocation struct {
	ID                 string
	allocType          jobtypes.AllocationType
	Actor              actor.Actor
	actorRunning       bool
	status             AllocationStatus
	nodeID             string
	sourceID           string
	computeProviderDID string
	deploymentID       string
	orchestrator       actor.Handle
	executor           types.Executor
	executionID        string
	Job                Job
	network            network.Network
	// TODO: create separated type for vpn info
	state struct {
		subnetIP    string
		gatewayIP   string
		portMapping map[int]int
	}
	resultsDir string

	workDir string
	lock    sync.Mutex
	fs      afero.Afero

	healthcheck func() error

	// selfRelease will use node's releaseAllocation mechanism
	selfRelease func() error

	Contracts            map[string]types.ContractConfig
	contractEventHandler *eventhandler.EventHandler
	contractStore        TailContractGetter

	createdAt time.Time
	startedAt time.Time

	// Liveness reporting state
	liveness allocationLiveness
}

func (a *Allocation) setStatus(ns AllocationStatus, msg string, notify bool) {
	a.lock.Lock()
	defer a.lock.Unlock()
	os := a.status
	a.status = ns

	if notify && os != ns {
		a.statusChangeNotify(os, ns, msg)
	}
}

// TailContractFinder is an interface for finding tail contracts associated with a head contract.
// This interface is used to avoid import cycles between jobs and store packages.
// It returns ContractConfig directly to avoid referencing contracts.Contract.
type TailContractGetter interface {
	FindTailContract(headContractConfig types.ContractConfig, computeProviderDID string) (*types.ContractConfig, error)
}

// NewAllocation creates a new allocation given the actor.
func NewAllocation(
	id string,
	allocType jobtypes.AllocationType,
	orchestrator actor.Handle,
	fs afero.Afero,
	workDir string,
	actor actor.Actor,
	details AllocationDetails,
	network network.Network,
	executor types.Executor,
	selfRelease func() error,
	contractEventHandler *eventhandler.EventHandler,
	contractStore TailContractGetter,
	deploymentID string,
) (*Allocation, error) {
	if network == nil {
		return nil, fmt.Errorf("network is nil")
	}

	if actor == nil {
		return nil, fmt.Errorf("actor is nil")
	}

	if executor == nil {
		return nil, fmt.Errorf("executor is nil")
	}

	executionID, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("create executor id: %w", err)
	}

	allocation := &Allocation{
		ID:           id,
		allocType:    allocType,
		fs:           fs,
		nodeID:       details.NodeID,
		sourceID:     details.SourceID,
		orchestrator: orchestrator,
		Job:          details.Job,
		Actor:        actor,
		executionID:  executionID.String(),
		workDir:      workDir,
		status:       AllocationPending,
		network:      network,
		executor:     executor,
		selfRelease:  selfRelease,
		state: struct {
			subnetIP    string
			gatewayIP   string
			portMapping map[int]int
		}{},
		createdAt:            time.Now(),
		contractEventHandler: contractEventHandler,
		contractStore:        contractStore,
		computeProviderDID:   actor.Parent().DID.URI,
		deploymentID:         deploymentID,
	}

	// Initialize liveness reporting state
	allocation.liveness.enabled = true // hard coded for now
	allocation.liveness.interval = livenessReportInterval

	log.Debugw("allocation_created",
		"labels", string(observability.LabelAllocation),
		"allocationID", allocation.ID,
		"allocDID", allocation.Actor.Handle().DID.String(),
		"executionID", allocation.executionID,
	)

	return allocation, nil
}

// GetPortMapping returns allocation's port mapping
func (a *Allocation) GetPortMapping() map[int]int {
	a.lock.Lock()
	defer a.lock.Unlock()

	ports := make(map[int]int)
	for i, v := range a.state.portMapping {
		ports[i] = v
	}

	return ports
}

// findTailContractsForHeadContract finds Tail Contracts associated with a Head Contract
// using the Head Contract config from the ensemble
func (a *Allocation) findTailContractForHeadContract(headContractConfig types.ContractConfig) (*types.ContractConfig, error) {
	if a.contractStore == nil {
		return nil, fmt.Errorf("contract store is not available")
	}

	tailContract, err := a.contractStore.FindTailContract(headContractConfig, a.computeProviderDID)
	if err != nil {
		return nil, fmt.Errorf("failed to find tail contracts for head contract %s: %w", headContractConfig.DID, err)
	}

	return tailContract, nil
}

// Run creates the executor based on the execution engine configuration.
func (a *Allocation) Run(
	ctx context.Context, subnetIP string,
	gatewayIP string, portMapping map[int]int,
) error {
	a.lock.Lock()
	defer func() {
		a.lock.Unlock()
		a.setStatus(AllocationRunning, "allocation started", true)
	}()

	if a.status == AllocationRunning {
		log.Warnw("allocation_already_running",
			"labels", string(observability.LabelAllocation),
			"allocationID", a.ID)
		// TODO: Should we return error instead?
		return nil
	}

	var err error
	a.resultsDir = filepath.Join(a.workDir, "jobs", a.ID)
	err = a.fs.MkdirAll(a.resultsDir, 0o700)
	if err != nil {
		return fmt.Errorf("create results directory: %w", err)
	}

	executionRequest := &types.ExecutionRequest{
		JobID:               a.ID,
		ExecutionID:         a.executionID,
		EngineSpec:          &a.Job.Execution,
		Resources:           &a.Job.Resources,
		ProvisionScripts:    a.Job.ProvisionScripts,
		Keys:                a.Job.Keys,
		ResultsDir:          a.resultsDir,
		PersistLogsDuration: deleteLogsAfter,
		GatewayIP:           gatewayIP,
	}

	// prepare the directories on host
	if len(a.Job.Volume) > 0 {
		executionRequest.Inputs = make([]*types.StorageVolumeExecutor, 0)

		for _, v := range a.Job.Volume {
			src := ""
			if v.Type == "glusterfs" {
				src = filepath.Join(a.workDir, "volumes", a.ID, v.Name)
			} else {
				src = v.Src
			}

			target := v.MountDestination
			if target == "" {
				target = "/" + v.Name
			}

			executionRequest.Inputs = append(executionRequest.Inputs, &types.StorageVolumeExecutor{
				Type:     "bind",
				Source:   src,
				Target:   target,
				ReadOnly: v.ReadOnly,
			})
		}
	}

	for hostPort, executorPort := range portMapping {
		executionRequest.PortsToBind = append(
			executionRequest.PortsToBind,
			types.PortsToBind{
				IP:           subnetIP,
				HostPort:     hostPort,
				ExecutorPort: executorPort,
			},
		)
	}

	err = a.executor.Start(ctx, executionRequest)
	if err != nil {
		return fmt.Errorf("start executor: %w", err)
	}

	// Update status (lock already held from function start)
	a.startedAt = time.Now()

	var headContractConfig types.ContractConfig
	// Find Head Contract config from ensemble contracts
	for _, contractConfig := range a.Contracts {
		headContractConfig = contractConfig
		break
	}
	headContractDID := headContractConfig.DID

	// Find Tail Contracts associated with Head Contract
	var contractsToNotify []types.ContractConfig
	if headContractDID != "" {
		// Contract chain scenario: use Tail Contracts
		tailContract, err := a.findTailContractForHeadContract(headContractConfig)
		if err != nil {
			log.Warnw("failed to find tail contracts, falling back to ensemble contracts",
				"head_contract_did", headContractDID,
				"error", err)
			// Convert map to slice for fallback
			for _, v := range a.Contracts {
				contractsToNotify = append(contractsToNotify, v)
			}
		} else {
			contractsToNotify = []types.ContractConfig{*tailContract}
		}
	} else {
		// P2P scenario: use contracts from ensemble
		for _, v := range a.Contracts {
			contractsToNotify = append(contractsToNotify, v)
		}
	}

	// Send events to Tail Contracts (or P2P contracts)
	for _, v := range contractsToNotify {
		evt := events.StartAllocation{
			EventBase: events.EventBase{Type: events.StartAllocationEvent},
			AllocationBase: events.AllocationBase{
				AllocationID:       a.ID,
				DeploymentID:       a.deploymentID,
				ComputeProviderDID: a.computeProviderDID,
				HeadContractDID:    headContractDID, // Include Head Contract DID in payload
			},
			Resources: a.Job.Resources,
		}
		a.contractEventHandler.Push(eventhandler.Event{
			ContractHostDID: v.Host,
			ContractDID:     v.DID,
			Payload:         evt,
		})
	}

	// NEW: Log the resources we've assigned for this run
	log.Infow("allocation_run_started",
		"labels", string(observability.LabelAllocation),
		"allocationID", a.ID,
		"cpuCoresAssigned", a.Job.Resources.CPU.Cores,
		"ramGBAssigned", a.Job.Resources.RAM.SizeInGB(),
		"gpuCountAssigned", len(a.Job.Resources.GPUs),
	)

	if a.allocType == jobtypes.AllocationTypeTask {
		go a.handleExecutionExit(ctx)
	} else {
		// Start periodic liveness reporting for service allocations
		a.startLivenessReporting(ctx)
	}

	return nil
}

// handleExecutionExit handles the exit of an execution
//
// TODO: retry policy for transient and long-running allocations
func (a *Allocation) handleExecutionExit(ctx context.Context) {
	resChan, errChan := a.executor.Wait(ctx, a.executionID)

	var result *types.ExecutionResult
	var err error

	select {
	case result = <-resChan:
	case err = <-errChan:
	case <-ctx.Done():
		err = ctx.Err()
	}

	a.handleTransience(result, err)
}

// handleTransience handles the exit of an execution for transient allocations.
//
// TODO: retry policy (meanwhile, we'll teardown everything in case of error)
func (a *Allocation) handleTransience(r *types.ExecutionResult, err error) {
	notifyOrchestrator := func(req behaviors.TaskTerminationNotification) {
		req.AllocationID = a.ID
		req.Status = string(a.status)

		// send logs if existent
		if r != nil {
			if len(r.STDOUT) > 0 {
				req.Stdout = []byte(r.STDOUT)
			}
			if len(r.STDERR) > 0 {
				req.Stderr = []byte(r.STDERR)
			}
		}

		msg, err := actor.Message(
			a.Actor.Handle(),
			a.orchestrator,
			behaviors.NotifyTaskTerminationBehavior,
			req,
			actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
		)
		if err != nil {
			log.Errorf("error creating task termination notification: %s", err)
		}

		err = a.Actor.Send(msg)
		if err != nil {
			log.Errorf("error notifying orchestrator: %s", err)
		}
	}

	if err != nil {
		log.Warnf("execution failed: %v", err)
		a.setStatus(AllocationFailed, "execution failed", true)

		exitCode := 0
		if r != nil {
			exitCode = r.ExitCode
		}

		notifyOrchestrator(behaviors.TaskTerminationNotification{
			Error: behaviors.TerminationError{
				ExitCode: exitCode,
				Err:      fmt.Sprintf("general execution failure: %v", err),
			},
		})
	} else if r != nil {
		switch {
		case r.ExitCode != 0:
			log.Infof("execution exited with exit code: %d", r.ExitCode)
			a.setStatus(AllocationFailed, fmt.Sprintf("execution exited with exit code: %d", r.ExitCode), true)

			notifyOrchestrator(behaviors.TaskTerminationNotification{
				Error: behaviors.TerminationError{
					ExitCode: r.ExitCode,
					Err:      fmt.Sprintf("execution exit code != 0, exit code: %d", r.ExitCode),
				},
			})
		case r.ExitCode == 0 && !r.Killed:
			log.Infof("task execution successfully completed")
			a.setStatus(AllocationCompleted, "task execution successfully completed", true)
			notifyOrchestrator(behaviors.TaskTerminationNotification{})
		case r.ExitCode == 0 && r.Killed:
			log.Infof("execution possibly killed")
			a.setStatus(AllocationFailed, "execution possibly killed", true)
			notifyOrchestrator(behaviors.TaskTerminationNotification{
				Error: behaviors.TerminationError{
					ExitCode: r.ExitCode,
					Err:      "execution possibly killed",
					Killed:   true,
				},
			})
		}
	}

	log.Debugf("self releasing: %s", a.ID)
	err = a.selfRelease()
	if err != nil {
		log.Errorf("error releasing self: %s", err)
	}

	if len(a.Contracts) == 0 {
		log.Errorf("no contracts  (handleTranscience)",
			"labels", string(observability.LabelAllocation),
			"allocationID", a.ID)
		return
	}

	var headContractConfig types.ContractConfig
	for _, contractConfig := range a.Contracts {
		headContractConfig = contractConfig
		break
	}
	headContractDID := headContractConfig.DID

	// Find Tail Contracts associated with Head Contract
	var contractsToNotify []types.ContractConfig
	if headContractDID != "" {
		// Contract chain scenario: use Tail Contracts
		tailContract, err := a.findTailContractForHeadContract(headContractConfig)
		if err != nil {
			log.Warnw("failed to find tail contracts, falling back to ensemble contracts",
				"head_contract_did", headContractDID,
				"error", err)
			// Convert map to slice for fallback
			for _, v := range a.Contracts {
				contractsToNotify = append(contractsToNotify, v)
			}
		} else {
			contractsToNotify = []types.ContractConfig{*tailContract}
		}
	} else {
		// P2P scenario: use contracts from ensemble
		for _, v := range a.Contracts {
			contractsToNotify = append(contractsToNotify, v)
		}
	}

	// Send events to Tail Contracts (or P2P contracts)
	for _, v := range contractsToNotify {
		evt := events.CompleteAllocation{
			EventBase: events.EventBase{Type: events.CompleteAllocationEvent},
			AllocationBase: events.AllocationBase{
				AllocationID:       a.ID,
				DeploymentID:       a.deploymentID,
				ComputeProviderDID: a.computeProviderDID,
				HeadContractDID:    headContractDID, // Include Head Contract DID in payload
			},
		}
		a.contractEventHandler.Push(eventhandler.Event{
			ContractHostDID: v.Host,
			ContractDID:     v.DID,
			Payload:         evt,
		})
	}
}

// Cancel stops the running executor
func (a *Allocation) stopExecution(ctx context.Context) error {
	log.Debugw("allocation_stopping_execution",
		"labels", string(observability.LabelAllocation),
		"allocationID", a.ID)

	if a.Status().Status != AllocationRunning {
		return nil
	}

	if a.executor == nil {
		return nil
	}

	if err := a.executor.Cancel(ctx, a.executionID); err != nil {
		a.setStatus(AllocationFailed, fmt.Sprintf("error stopping executor: %v", err), true)
		return fmt.Errorf("stop execution: %w", err)
	}

	a.setStatus(AllocationStopped, "allocation stopped", true)
	log.Debugw("allocation_stopped_execution",
		"labels", string(observability.LabelAllocation),
		"allocationID", a.ID)
	return nil
}

func (a *Allocation) Cleanup() error {
	if a.executor == nil {
		return nil
	}

	if err := a.executor.Remove(a.executionID, orchestrator.AllocationShutdownTimeout); err != nil {
		return fmt.Errorf("failed to remove execution: %w", err)
	}

	log.Debugw("allocation_removed_execution",
		"labels", string(observability.LabelAllocation),
		"executionID", a.executionID)
	return nil
}

// Terminate stops the allocation and cleans up after
// TODO: shouldn't act on a best effort basis? meaning,
// it won't return errors right away but try to clean up
// all the other steps
func (a *Allocation) Terminate(ctx context.Context) error {
	var headContractConfig types.ContractConfig
	for _, contractConfig := range a.Contracts {
		headContractConfig = contractConfig
		break
	}
	headContractDID := headContractConfig.DID

	// Find Tail Contracts associated with Head Contract
	var contractsToNotify []types.ContractConfig
	if headContractDID != "" {
		// Contract chain scenario: use Tail Contracts
		tailContract, err := a.findTailContractForHeadContract(headContractConfig)
		if err != nil {
			log.Warnw("failed to find tail contracts, falling back to ensemble contracts",
				"head_contract_did", headContractDID,
				"error", err)
			// Convert map to slice for fallback
			for _, v := range a.Contracts {
				contractsToNotify = append(contractsToNotify, v)
			}
		} else {
			contractsToNotify = []types.ContractConfig{*tailContract}
		}
	} else {
		// P2P scenario: use contracts from ensemble
		for _, v := range a.Contracts {
			contractsToNotify = append(contractsToNotify, v)
		}
	}

	// Send events to Tail Contracts (or P2P contracts)
	for _, v := range contractsToNotify {
		evt := events.StopAllocation{
			EventBase: events.EventBase{Type: events.StopAllocationEvent},
			AllocationBase: events.AllocationBase{
				AllocationID:       a.ID,
				DeploymentID:       a.deploymentID,
				ComputeProviderDID: a.computeProviderDID,
				HeadContractDID:    headContractDID, // Include Head Contract DID in payload
			},
		}
		a.contractEventHandler.Push(eventhandler.Event{
			ContractHostDID: v.Host,
			ContractDID:     v.DID,
			Payload:         evt,
		})
	}

	status := a.Status().Status
	if status != AllocationStopped && status != AllocationCompleted {
		err := a.Stop(ctx)
		if err != nil {
			log.Warnw("allocation_failed_to_stop",
				"labels", string(observability.LabelAllocation),
				"error", err,
				"allocationID", a.ID)
			return fmt.Errorf("failed to stop allocation: %w", err)
		}

		// terminated status only if had to stop
		a.setStatus(AllocationTerminated, "allocation terminated", true)
	}

	if err := a.Cleanup(); err != nil {
		log.Warnw("allocation_failed_to_cleanup",
			"labels", string(observability.LabelAllocation),
			"error", err,
			"allocationID", a.ID)
	}

	return nil
}

// StopActor stops the allocation actor
func (a *Allocation) stopActor() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.actorRunning {
		if err := a.Actor.Stop(); err != nil {
			log.Warnw("allocation_actor_stop_failure",
				"labels", string(observability.LabelAllocation),
				"error", err,
				"allocationID", a.ID)
		}
		log.Debugw("allocation_actor_stopped",
			"labels", string(observability.LabelAllocation),
			"allocationID", a.ID)
		a.actorRunning = false
	}
	return nil
}

// Stop stops the running executor and the allocation actor
// TODO: shouldn't act on a best effort basis? meaning,
// it won't return errors right away but try to clean up
// all the other steps
func (a *Allocation) Stop(ctx context.Context) error {
	// Stop liveness reporting first
	a.stopLivenessReporting()

	err := a.stopActor()
	if err != nil {
		return fmt.Errorf("stop actor: %w", err)
	}

	err = a.stopExecution(ctx)
	if err != nil {
		a.setStatus(AllocationFailed, "failed to stop execution", true)
		return fmt.Errorf("stop execution: %w", err)
	}

	a.setStatus(AllocationStopped, "allocation stopped", true)

	return nil
}

// Status returns information about the allocated/usage of resources and execution status of workload.
func (a *Allocation) Status() Status {
	a.lock.Lock()
	defer a.lock.Unlock()

	return Status{
		JobResources: a.Job.Resources,
		Status:       a.status,
	}
}

// Start the actor of the allocation.
func (a *Allocation) Start() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	// start actor
	if a.actorRunning {
		return nil
	}

	allocationBehaviors := map[string]func(actor.Envelope){
		behaviors.AllocationStartBehavior:        a.handleAllocationStart,
		behaviors.AllocationRestartBehavior:      a.handleAllocationRestart,
		behaviors.AllocationStatsBehavior:        a.handleAllocationStats,
		behaviors.SubnetAddPeerBehavior:          a.handleSubnetAddPeer,
		behaviors.SubnetRemovePeersBehavior:      a.handleSubnetRemovePeers,
		behaviors.SubnetAcceptPeersBehavior:      a.handleSubnetAcceptPeers,
		behaviors.SubnetMapPortBehavior:          a.handleSubnetMapPort,
		behaviors.SubnetUnmapPortBehavior:        a.handleSubnetUnmapPort,
		behaviors.SubnetDNSAddRecordsBehavior:    a.handleSubnetDNSAddRecords,
		behaviors.SubnetDNSRemoveRecordsBehavior: a.handleSubnetDNSRemoveRecords,
		behaviors.RegisterHealthcheckBehavior:    a.handleRegisterHealthcheck,
		actor.HealthCheckBehavior:                a.handleHealthcheck,
	}

	// add allocation behaviours to actor
	for behavior, handler := range allocationBehaviors {
		err := a.Actor.AddBehavior(behavior, handler)
		if err != nil {
			return fmt.Errorf("add allocation start behavior to allocation actor: %w", err)
		}
	}

	err := a.Actor.Start()
	if err != nil {
		return fmt.Errorf("start allocation actor: %w", err)
	}

	a.actorRunning = true
	return nil
}

func (a *Allocation) Restart(ctx context.Context) error {
	if a.state.subnetIP == "" {
		// if you get this error, did you start the allocation properly before restart?
		return fmt.Errorf("allocation: state is empty, no subnet ip is provided")
	}

	if err := a.Stop(ctx); err != nil {
		return err
	}

	if err := a.Start(); err != nil {
		return err
	}

	if err := a.Run(ctx, a.state.subnetIP, a.state.gatewayIP, a.state.portMapping); err != nil {
		_ = a.Stop(ctx)
		return fmt.Errorf("run allocation: %w", err)
	}

	return nil
}

// TODO: make send reply a helper func from actor pkg
func (a *Allocation) sendReply(msg actor.Envelope, payload interface{}) {
	var opt []actor.MessageOption
	if msg.IsBroadcast() {
		opt = append(opt, actor.WithMessageSource(a.Actor.Handle()))
	}

	reply, err := actor.ReplyTo(msg, payload, opt...)
	if err != nil {
		log.Debugf("creating reply: %s", err)
		return
	}

	if err := a.Actor.Send(reply); err != nil {
		log.Debugf("sending reply: %s", err)
	}
}

func (a *Allocation) SetHealthCheck(f func() error) {
	a.lock.Lock()
	defer a.lock.Unlock()

	a.healthcheck = f
}

func (a *Allocation) Info() AllocationInfo {
	a.lock.Lock()
	defer a.lock.Unlock()

	return AllocationInfo{
		ID:           a.ID,
		Type:         a.allocType,
		Orchestrator: a.orchestrator.Address.HostID,
		Resources:    a.Job.Resources,
		Status:       string(a.status),
		Executor:     a.Job.Execution.Type,
		ExecutionID:  a.ID,
		UsingPorts:   utils.MapKeysToSlice(a.state.portMapping),
		CreatedAt:    a.createdAt,
		StartedAt:    a.startedAt,
	}
}

// startLivenessReporting starts periodic push-based liveness reporting
// Only for service allocations - tasks use handleTransience
func (a *Allocation) startLivenessReporting(ctx context.Context) {
	if a.allocType == jobtypes.AllocationTypeTask {
		return // Tasks already push via handleTransience
	}

	if !a.liveness.enabled {
		log.Debugw("push_liveness_disabled",
			"labels", string(observability.LabelAllocation),
			"allocationID", a.ID)
		return
	}

	livenessCtx, cancel := context.WithCancel(ctx)
	a.liveness.lock.Lock()
	a.liveness.cancel = cancel
	a.liveness.lock.Unlock()

	log.Infow("starting_push_liveness_reporting",
		"labels", string(observability.LabelAllocation),
		"allocationID", a.ID,
		"interval", a.liveness.interval,
		"note", "passive collection only, pull checks remain authoritative")

	go func() {
		// Send initial heartbeat immediately
		if err := a.sendLivenessReport(livenessCtx); err != nil {
			log.Debugw("initial_liveness_report_failed",
				"labels", string(observability.LabelAllocation),
				"allocationID", a.ID,
				"error", err)
		}

		ticker := time.NewTicker(a.liveness.interval)
		defer ticker.Stop()

		for {
			select {
			case <-livenessCtx.Done():
				log.Debugw("stopping_liveness_reporting",
					"labels", string(observability.LabelAllocation),
					"allocationID", a.ID)
				return
			case <-ticker.C:
				if err := a.sendLivenessReport(livenessCtx); err != nil {
					log.Debugw("liveness_report_failed",
						"labels", string(observability.LabelAllocation),
						"allocationID", a.ID,
						"error", err)
					// Continue trying - don't stop on failure
				}
			}
		}
	}()
}

// sendLivenessReport sends a single liveness notification
func (a *Allocation) sendLivenessReport(ctx context.Context) error {
	currentStatus := a.Status()

	// Increment sequence number
	a.liveness.lock.Lock()
	a.liveness.sequenceNumber++
	seqNum := a.liveness.sequenceNumber
	a.liveness.lock.Unlock()

	// Perform self health check
	health := a.performSelfHealthCheck(ctx)

	// Optionally gather resource usage
	var resourceUsage *jobtypes.AllocationResourceUsage
	if usage, err := a.gatherResourceUsage(ctx); err == nil {
		resourceUsage = usage
	}

	notification := jobtypes.AllocationLivenessNotification{
		AllocationID:   a.ID,
		Status:         string(currentStatus.Status),
		Timestamp:      time.Now().Unix(),
		SequenceNumber: seqNum,
		Health:         health,
		ResourceUsage:  resourceUsage,
		Version:        "0.1",
	}

	return a.sendToOrchestratorWithRetry(
		ctx,
		behaviors.NotifyAllocationLivenessBehavior,
		notification,
		livenessMaxRetries,
	)
}

// performSelfHealthCheck runs registered healthcheck (if any)
func (a *Allocation) performSelfHealthCheck(ctx context.Context) jobtypes.HealthStatus {
	a.lock.Lock()
	healthcheck := a.healthcheck
	a.lock.Unlock()

	if healthcheck == nil {
		return jobtypes.HealthStatus{
			Healthy:       true,
			LastCheckTime: time.Now().Unix(),
			CheckType:     jobtypes.HealthCheckTypeNone,
			Message:       "no healthcheck configured",
		}
	}

	// Run with timeout
	checkCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	errChan := make(chan error, 1)
	go func() {
		errChan <- healthcheck()
		close(errChan)
	}()

	select {
	case err := <-errChan:
		if err != nil {
			return jobtypes.HealthStatus{
				Healthy:       false,
				LastCheckTime: time.Now().Unix(),
				CheckType:     jobtypes.HealthCheckTypeSelf,
				Message:       fmt.Sprintf("healthcheck failed: %v", err),
			}
		}
		return jobtypes.HealthStatus{
			Healthy:       true,
			LastCheckTime: time.Now().Unix(),
			CheckType:     jobtypes.HealthCheckTypeSelf,
			Message:       "healthcheck passed",
		}
	case <-checkCtx.Done():
		return jobtypes.HealthStatus{
			Healthy:       false,
			LastCheckTime: time.Now().Unix(),
			CheckType:     jobtypes.HealthCheckTypeSelf,
			Message:       "healthcheck timeout",
		}
	}
}

// gatherResourceUsage collects resource metrics
func (a *Allocation) gatherResourceUsage(ctx context.Context) (*jobtypes.AllocationResourceUsage, error) {
	// zero usage if allocation not running
	if a.Status().Status != jobtypes.AllocationRunning {
		return &jobtypes.AllocationResourceUsage{}, nil
	}

	stats, err := a.executor.Stats(ctx, a.executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve allocation stats: %w", err)
	}

	if stats == nil {
		return nil, fmt.Errorf("allocation stats are nil")
	}

	resrcUsage := jobtypes.AllocationResourceUsage{
		CPUUsagePercent:  stats.CPUUsage.Percent,
		MemoryUsedBytes:  stats.Memory.Usage,
		MemoryLimitBytes: a.Job.Resources.RAM.Size,
		NetworkRxBytes:   stats.Network.RxBytes,
		NetworkTxBytes:   stats.Network.TxBytes,
	}
	return &resrcUsage, nil
}

// sendToOrchestratorWithRetry sends with exponential backoff
func (a *Allocation) sendToOrchestratorWithRetry(
	ctx context.Context,
	behavior string,
	payload interface{},
	maxRetries int,
) error {
	var lastErr error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff: 2^(attempt-1) seconds (1s, 2s, 4s, 8s...)
			backoff := time.Duration(math.Pow(2, float64(attempt-1))) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}

		msg, err := actor.Message(
			a.Actor.Handle(),
			a.orchestrator,
			behavior,
			payload,
			actor.WithMessageExpiry(uint64(time.Now().Add(livenessReportTimeout).UnixNano())),
		)
		if err != nil {
			lastErr = fmt.Errorf("create message: %w", err)
			continue
		}

		if err := a.Actor.Send(msg); err != nil {
			lastErr = fmt.Errorf("send attempt %d: %w", attempt+1, err)
			continue
		}

		return nil // Success
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries+1, lastErr)
}

// statusChangeNotify sends immediate notification when status changes
func (a *Allocation) statusChangeNotify(oldStatus, newStatus AllocationStatus, reason string) {
	if !a.liveness.enabled {
		return
	}

	update := jobtypes.AllocationStatusUpdate{
		AllocationID: a.ID,
		OldStatus:    string(oldStatus),
		NewStatus:    string(newStatus),
		Timestamp:    time.Now().Unix(),
		Reason:       reason,
	}

	msg, err := actor.Message(
		a.Actor.Handle(),
		a.orchestrator,
		behaviors.NotifyAllocationStatusBehavior,
		update,
		actor.WithMessageExpiry(uint64(time.Now().Add(livenessReportTimeout).UnixNano())),
	)
	if err != nil {
		log.Debugf("failed to create status update message: %v", err)
		return
	}

	if err := a.Actor.Send(msg); err != nil {
		log.Debugf("failed to send status update: %v", err)
	}
}

// stopLivenessReporting stops the liveness reporting goroutine
func (a *Allocation) stopLivenessReporting() {
	a.liveness.lock.Lock()
	defer a.liveness.lock.Unlock()

	if a.liveness.cancel != nil {
		a.liveness.cancel()
		a.liveness.cancel = nil
	}
}
