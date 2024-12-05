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
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/executor"
	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/network"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	pending   AllocationStatus = "pending"
	running   AllocationStatus = "running"
	stopped   AllocationStatus = "stopped"
	completed AllocationStatus = "completed"

	deleteLogsAfter = 30 * time.Minute
)

// AllocationStatus is a representation of the execution status
type AllocationStatus string

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
	ID               string
	AllocationID     string
	Resources        types.Resources
	Execution        types.SpecConfig
	ProvisionScripts map[string][]byte
}

// Allocation represents an allocation
type Allocation struct {
	ID string

	mx sync.Mutex
	fs afero.Afero

	status      AllocationStatus
	nodeID      string
	sourceID    string
	executionID string

	Actor           actor.Actor
	executor        executor.Executor
	resourceManager types.ResourceManager
	dmsConfig       config.Config

	network network.Network

	actorRunning bool

	Job        Job
	resultsDir string
}

// NewAllocation creates a new allocation given the actor.
func NewAllocation(
	fs afero.Afero,
	dmsConfig config.Config,
	actor actor.Actor,
	details AllocationDetails,
	resourceManager types.ResourceManager,
	network network.Network,
) (*Allocation, error) {
	if resourceManager == nil {
		return nil, errors.New("resource manager is nil")
	}

	id, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to generate uuid for allocation: %w", err)
	}

	executorID, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to create executor id: %w", err)
	}

	return &Allocation{
		ID:              id.String(),
		fs:              fs,
		nodeID:          details.NodeID,
		sourceID:        details.SourceID,
		Job:             details.Job,
		Actor:           actor,
		executionID:     executorID.String(),
		resourceManager: resourceManager,
		dmsConfig:       dmsConfig,
		status:          pending,
		network:         network,
	}, nil
}

// Run creates the executor based on th e execution engine configuration.
func (a *Allocation) Run(ctx context.Context) error {
	a.mx.Lock()
	defer a.mx.Unlock()

	if a.status == running {
		log.Warnf("allocation %s is already running", a.ID)
		return nil
	}

	var err error

	// if executor is nil create it
	if a.executor == nil {
		err = a.createExecutor(ctx, a.Job.Execution)
		if err != nil {
			return fmt.Errorf("failed to create executor: %w", err)
		}
	}

	a.resultsDir = filepath.Join(a.dmsConfig.WorkDir, "jobs", a.Job.ID)
	err = a.fs.MkdirAll(a.resultsDir, 0o700)
	if err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	err = a.executor.Start(ctx, &types.ExecutionRequest{
		JobID:            a.Job.ID,
		ExecutionID:      a.executionID,
		EngineSpec:       &a.Job.Execution,
		Resources:        &a.Job.Resources,
		ProvisionScripts: a.Job.ProvisionScripts,
		// TODO add the following
		Inputs:              []*types.StorageVolumeExecutor{}, // Question: what are those?
		Outputs:             []*types.StorageVolumeExecutor{},
		ResultsDir:          a.resultsDir,
		PersistLogsDuration: deleteLogsAfter,
	})
	if err != nil {
		return fmt.Errorf("failed to start executor: %w", err)
	}

	a.status = running

	go a.monitorExecutor(ctx)

	return nil
}

func (a *Allocation) monitorExecutor(ctx context.Context) {
	resChan, errChan := a.executor.Wait(ctx, a.executionID)

	var finalResult *types.ExecutionResult
	var finalError error

	for {
		select {
		case res, ok := <-resChan:
			if !ok {
				resChan = nil // no more reads
			} else {
				finalResult = res
			}

		case err, ok := <-errChan:
			if !ok {
				errChan = nil // no more reads
			} else {
				finalError = err
			}
		}

		if resChan == nil && errChan == nil {
			break
		}
	}

	a.mx.Lock()
	if finalError != nil {
		a.status = stopped
		log.Warnf("execution failed: %v", finalError)
	}

	if finalResult != nil {
		a.status = completed
	}
	a.mx.Unlock()

	// deallocate resources after everything is done.
	err := a.resourceManager.DeallocateResources(ctx, a.Job.AllocationID)
	if err != nil {
		log.Errorf("failed to deallocate resources for %s: %v", a.Job.AllocationID, err)
	}
}

// Stop stops the running executor
func (a *Allocation) Stop(ctx context.Context) error {
	a.mx.Lock()
	defer a.mx.Unlock()

	defer func() {
		if a.actorRunning {
			if err := a.Actor.Stop(); err != nil {
				log.Warnf("error stopping allocation actor: %s", err)
			}
			a.actorRunning = false
		}
	}()

	if a.status != running {
		return nil
	}

	if err := a.executor.Cancel(ctx, a.executionID); err != nil {
		return fmt.Errorf("failed to stop execution: %w", err)
	}

	a.status = stopped

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
	a.mx.Lock()
	defer a.mx.Unlock()

	behaviors := map[string]func(actor.Envelope){
		AllocationStartBehavior:   a.handleAllocationStart,
		AllocationGetLogsBehavior: a.handleAllocationGetLogs,

		SubnetAddPeerBehavior:         a.handleSubnetAddPeer,
		SubnetRemovePeerBehavior:      a.handleSubnetRemovePeer,
		SubnetAcceptPeerBehavior:      a.handleSubnetAcceptPeer,
		SubnetMapPortBehavior:         a.handleSubnetMapPort,
		SubnetUnmapPortBehavior:       a.handleSubnetUnmapPort,
		SubnetDNSAddRecordBehavior:    a.handleSubnetDNSAddRecord,
		SubnetDNSRemoveRecordBehavior: a.handleSubnetDNSRemoveRecord,
	}

	// add allocation behaviors
	for behavior, handler := range behaviors {
		err := a.Actor.AddBehavior(behavior, handler)
		if err != nil {
			return fmt.Errorf("failed to add allocation start behavior to allocation actor: %w", err)
		}
	}

	// start actor
	if a.actorRunning {
		return nil
	}

	err := a.Actor.Start()
	if err != nil {
		return fmt.Errorf("failed to start allocation actor: %w", err)
	}

	a.actorRunning = true

	return nil
}

func (a *Allocation) createExecutor(ctx context.Context, execution types.SpecConfig) error {
	switch execution.Type {
	case types.ExecutorTypeDocker.String():
		id := uuid.New().String()
		exec, err := docker.NewExecutor(ctx, a.fs, id)
		if err != nil {
			return fmt.Errorf("failed to create executor: %w", err)
		}
		a.executor = exec
	default:
		return fmt.Errorf("unsupported executor type: %s", execution.Type)
	}

	return nil
}

func (a *Allocation) handleAllocationStart(msg actor.Envelope) {
	log.Infof("behavior allocation start invoked by: %+v", msg.From)
	defer msg.Discard()

	var resp AllocationStartResponse

	if err := a.Run(context.TODO()); err != nil {
		err = fmt.Errorf("failed to run allocation: %w", err)
		log.Error(err)

		resp.Error = err.Error()
		resp.OK = false
		a.sendReply(msg, resp)
		return
	}

	log.Info("Running allocation's job: ", a.ID)

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleAllocationGetLogs(msg actor.Envelope) {
	log.Infof("behavior get logs invoked by: %+v", msg.From)
	defer msg.Discard()

	var resp AllocationGetLogsResponse

	data, err := a.fs.ReadFile(filepath.Join(a.resultsDir, "stdout.log"))
	if err != nil {
		resp.Error = fmt.Sprintf("failed to read results file: %s", err.Error())
		a.sendReply(msg, resp)
		return
	}

	resp.Data = data
	a.sendReply(msg, resp)
}

// TODO: make send reply a helper func from actor pkg
func (a *Allocation) sendReply(msg actor.Envelope, payload interface{}) {
	var opt []actor.MessageOption
	if msg.IsBroadcast() {
		opt = append(opt, actor.WithMessageSource(a.Actor.Handle()))
	}

	reply, err := actor.ReplyTo(msg, payload, opt...)
	if err != nil {
		log.Debugf("error creating reply: %s", err)
		return
	}

	if err := a.Actor.Send(reply); err != nil {
		log.Debugf("error sending  reply: %s", err)
	}
}

func (a *Allocation) handleSubnetAddPeer(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetAddPeerRequest
	resp := SubnetAddPeerResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.AddSubnetPeer(request.SubnetID, request.PeerID, request.IP)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleSubnetAcceptPeer(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetAcceptPeerRequest
	resp := SubnetAcceptPeerResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.AcceptSubnetPeer(request.SubnetID, request.PeerID, request.IP)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleSubnetMapPort(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetMapPortRequest
	resp := SubnetMapPortResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.MapPort(request.SubnetID, request.Protocol, request.SourceIP, request.SourcePort, request.DestIP, request.DestPort)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleSubnetDNSAddRecord(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetDNSAddRecordRequest
	resp := SubnetDNSAddRecordResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.AddSubnetDNSRecord(request.SubnetID, request.DomainName, request.IP)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleSubnetUnmapPort(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetUnmapPortRequest
	resp := SubnetUnmapPortResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.UnmapPort(
		request.SubnetID, request.Protocol, request.SourceIP, request.SourcePort, request.DestIP, request.DestPort,
	)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleSubnetDNSRemoveRecord(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetDNSRemoveRecordRequest
	resp := SubnetDNSRemoveRecordResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.RemoveSubnetDNSRecord(request.SubnetID, request.DomainName)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleSubnetRemovePeer(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetRemovePeerRequest
	resp := SubnetRemovePeerResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.RemoveSubnetPeer(request.SubnetID, request.PeerID, request.IP)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	resp.OK = true
	a.sendReply(msg, resp)
}
