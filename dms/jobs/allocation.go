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
type AllocationDetails struct {
	Job      Job
	NodeID   string
	SourceID string
}

type Job struct {
	Resources        types.Resources
	Execution        types.SpecConfig
	ProvisionScripts map[string][]byte
	Volume           *types.VolumeConfig
}

// Allocation represents an allocation
type Allocation struct {
	ID           string
	Actor        actor.Actor
	actorRunning bool
	status       AllocationStatus
	nodeID       string
	sourceID     string
	executor     types.Executor
	executionID  string
	Job          Job
	network      network.Network
	state        struct {
		subnetIP    string
		gatewayIP   string
		portMapping map[int]int
	}
	resultsDir string

	workDir string
	lock    sync.Mutex
	fs      afero.Afero

	healthcheck func() error
}

// NewAllocation creates a new allocation given the actor.
func NewAllocation(
	id string,
	fs afero.Afero,
	workDir string,
	actor actor.Actor,
	details AllocationDetails,
	network network.Network,
	executor types.Executor,
) (*Allocation, error) {
	if network == nil {
		return nil, fmt.Errorf("network is nil")
	}

	executionID, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("create executor id: %w", err)
	}

	allocation := &Allocation{
		ID:          id,
		fs:          fs,
		nodeID:      details.NodeID,
		sourceID:    details.SourceID,
		Job:         details.Job,
		Actor:       actor,
		executionID: executionID.String(),
		workDir:     workDir,
		status:      AllocationPending,
		network:     network,
		executor:    executor,
		state: struct {
			subnetIP    string
			gatewayIP   string
			portMapping map[int]int
		}{},
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
func (a *Allocation) Run(ctx context.Context, subnetIP string, gatewayIP string, portMapping map[int]int) error {
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

	return nil
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
func (a *Allocation) Terminate(ctx context.Context) error {
	if a.status != AllocationStopped && a.status != AllocationCompleted {
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
