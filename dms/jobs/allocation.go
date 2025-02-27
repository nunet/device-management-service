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
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	deleteLogsAfter = 30 * time.Minute
)

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
	Volume           *types.VolumeConfig
}

// Allocation represents an allocation
type Allocation struct {
	ID           string
	allocType    jobtypes.AllocationType
	Actor        actor.Actor
	actorRunning bool
	status       AllocationStatus
	nodeID       string
	sourceID     string
	orchestrator actor.Handle
	executor     types.Executor
	executionID  string
	Job          Job
	network      network.Network
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
) (*Allocation, error) {
	// TODO: add check for nil values
	if network == nil {
		return nil, fmt.Errorf("network is nil")
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
		state: struct {
			subnetIP    string
			gatewayIP   string
			portMapping map[int]int
		}{},
		selfRelease: selfRelease,
	}
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

// Run creates the executor based on th e execution engine configuration.
func (a *Allocation) Run(
	ctx context.Context, subnetIP string,
	gatewayIP string, portMapping map[int]int,
) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.status == AllocationRunning {
		log.Warnf("allocation %s is already running", a.ID)
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
		ResultsDir:          a.resultsDir,
		PersistLogsDuration: deleteLogsAfter,
		GatewayIP:           gatewayIP,
	}

	if a.Job.Volume != nil {
		executionRequest.Inputs = []*types.StorageVolumeExecutor{
			{
				Type:     "bind",
				Source:   filepath.Join(a.workDir, "volumes", a.ID, a.Job.Volume.Name),
				Target:   "/" + a.Job.Volume.Name, // its important to prepend with / as target is expected to be an absolute path
				ReadOnly: false,
			},
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

	a.status = AllocationRunning

	if a.allocType == jobtypes.AllocationTypeTask {
		go a.handleExecutionExit(ctx)
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
	notifyOrchestrator := func(req TaskTerminationNotification) {
		req.AllocationID = a.ID
		req.Status = a.status

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

	a.lock.Lock()
	if err != nil {
		log.Warnf("execution failed: %v", err)
		a.status = AllocationFailed

		exitCode := 0
		if r != nil {
			exitCode = r.ExitCode
		}

		notifyOrchestrator(TaskTerminationNotification{
			Error: TerminationError{
				ExitCode: exitCode,
				Err:      fmt.Errorf("general execution failure: %w", err),
			},
		})
	} else if r != nil {
		// TODO: use switch-cases
		if r.ExitCode != 0 { //nolint
			log.Infof("execution exited with exit code: %d", r.ExitCode)
			a.status = AllocationFailed

			notifyOrchestrator(TaskTerminationNotification{
				Error: TerminationError{
					ExitCode: r.ExitCode,
					Err:      fmt.Errorf("execution exit code != 0, exit code: %d", r.ExitCode),
				},
			})
		} else if r.ExitCode == 0 && !r.Killed {
			log.Infof("execution successfully completed")
			a.status = AllocationCompleted
			notifyOrchestrator(TaskTerminationNotification{})
		} else if r.ExitCode == 0 && r.Killed {
			log.Infof("execution possibly killed")
			a.status = AllocationFailed
			notifyOrchestrator(TaskTerminationNotification{
				Error: TerminationError{
					ExitCode: r.ExitCode,
					Err:      fmt.Errorf("execution possibly killed"),
					Killed:   true,
				},
			})
		}
	}

	a.lock.Unlock()
	log.Debugf("self releasing: %s", a.ID)
	err = a.selfRelease()
	if err != nil {
		log.Errorf("error releasing self: %s", err)
	}
}

// Cancel stops the running executor
func (a *Allocation) stopExecution(ctx context.Context) error {
	a.lock.Lock()
	defer a.lock.Unlock()

	log.Debugf("stopping execution for alloc: %s", a.ID)

	if a.status != AllocationRunning {
		return nil
	}

	a.status = AllocationStopped

	if a.executor == nil {
		return nil
	}

	if err := a.executor.Cancel(ctx, a.executionID); err != nil {
		a.status = AllocationFailed
		return fmt.Errorf("stop execution: %w", err)
	}

	log.Debugf("stopped execution for alloc: %s", a.ID)

	return nil
}

func (a *Allocation) Cleanup() error {
	if a.executor == nil {
		return nil
	}

	if err := a.executor.Remove(a.executionID, AllocationShutdownTimeout); err != nil {
		return fmt.Errorf("failed to remove execution: %w", err)
	}
	log.Debugf("removed execution: %s", a.executionID)
	return nil
}

// Terminate stops the allocation and cleans up after
// TODO: shouldn't act on a best effort basis? meaning,
// it won't return errors right away but try to clean up
// all the other steps
func (a *Allocation) Terminate(ctx context.Context) error {
	status := a.Status(ctx)
	if status.Status != AllocationStopped && status.Status != AllocationCompleted {
		err := a.Stop(ctx)
		if err != nil {
			log.Warnf("failed to stop allocation: %s", err)
			return fmt.Errorf("failed to stop allocation: %w", err)
		}
	}

	a.lock.Lock()
	defer a.lock.Unlock()

	if err := a.Cleanup(); err != nil {
		// TODO: exec.Remove should return a defined custom error
		//       container already removed is not an error
		log.Warnf("failed to cleanup allocation: %s", err)
	}

	a.status = AllocationTerminated

	return nil
}

// StopActor stops the allocation actor
func (a *Allocation) stopActor() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	if a.actorRunning {
		if err := a.Actor.Stop(); err != nil {
			log.Warnf("stopping allocation actor: %s", err)
		}
		log.Debugf("stopped allocation actor: %s", a.ID)
		a.actorRunning = false
	}
	return nil
}

// Stop stops the running executor and the allocation actor
// TODO: shouldn't act on a best effort basis? meaning,
// it won't return errors right away but try to clean up
// all the other steps
func (a *Allocation) Stop(ctx context.Context) error {
	err := a.stopActor()
	if err != nil {
		return fmt.Errorf("stop actor: %w", err)
	}

	err = a.stopExecution(ctx)
	if err != nil {
		a.lock.Lock()
		a.status = AllocationFailed
		a.lock.Unlock()
		return fmt.Errorf("stop execution: %w", err)
	}

	a.lock.Lock()
	a.status = AllocationStopped
	a.lock.Unlock()

	return nil
}

// Status returns information about the allocated/usage of resources and execution status of workload.
func (a *Allocation) Status(_ context.Context) Status {
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

	allocationBehaviors := map[string]func(actor.Envelope){
		behaviors.AllocationStartBehavior:       a.handleAllocationStart,
		behaviors.AllocationRestartBehavior:     a.handleAllocationRestart,
		behaviors.SubnetAddPeerBehavior:         a.handleSubnetAddPeer,
		behaviors.SubnetRemovePeerBehavior:      a.handleSubnetRemovePeer,
		behaviors.SubnetAcceptPeerBehavior:      a.handleSubnetAcceptPeer,
		behaviors.SubnetMapPortBehavior:         a.handleSubnetMapPort,
		behaviors.SubnetUnmapPortBehavior:       a.handleSubnetUnmapPort,
		behaviors.SubnetDNSAddRecordsBehavior:   a.handleSubnetDNSAddRecords,
		behaviors.SubnetDNSRemoveRecordBehavior: a.handleSubnetDNSRemoveRecord,
		behaviors.RegisterHealthcheckBehavior:   a.handleRegisterHealthcheck,
		actor.HealthCheckBehavior:               a.handleHealthcheck,
	}

	// add allocation behaviours to actor
	for behavior, handler := range allocationBehaviors {
		err := a.Actor.AddBehavior(behavior, handler)
		if err != nil {
			return fmt.Errorf("add allocation start behavior to allocation actor: %w", err)
		}
	}

	// start actor
	if a.actorRunning {
		return nil
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
