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
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/executor"
	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	pending   AllocationStatus = "pending"
	running   AllocationStatus = "running"
	stopped   AllocationStatus = "stopped"
	completed AllocationStatus = "completed"
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
	Resources        types.Resources
	Execution        types.SpecConfig
	ProvisionScripts map[string][]byte
}

// Allocation represents an allocation
type Allocation struct {
	ID string

	mx sync.Mutex

	status      AllocationStatus
	nodeID      string
	sourceID    string
	executionID string

	Actor           actor.Actor
	executor        executor.Executor
	resourceManager types.ResourceManager

	actorRunning bool

	Job Job
}

// NewAllocation creates a new allocation given the actor.
func NewAllocation(actor actor.Actor, details AllocationDetails, resourceManager types.ResourceManager) (*Allocation, error) {
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
		nodeID:          details.NodeID,
		sourceID:        details.SourceID,
		Job:             details.Job,
		Actor:           actor,
		executionID:     executorID.String(),
		resourceManager: resourceManager,
		status:          pending,
	}, nil
}

// Run creates the executor based on the execution engine configuration.
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

	err = a.executor.Start(ctx, &types.ExecutionRequest{
		JobID:            a.Job.ID,
		ExecutionID:      a.executionID,
		EngineSpec:       &a.Job.Execution,
		Resources:        &a.Job.Resources,
		ProvisionScripts: a.Job.ProvisionScripts,
		// TODO add the following
		Inputs:     []*types.StorageVolumeExecutor{}, // Question: what are those?
		Outputs:    []*types.StorageVolumeExecutor{},
		ResultsDir: "",
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
	err := a.resourceManager.DeallocateResources(ctx, a.Job.ID)
	if err != nil {
		log.Errorf("failed to deallocate resources for %s: %v", a.Job.ID, err)
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

	// add allocation behaviors
	err := a.Actor.AddBehavior(AllocationStartBehavior, a.handleAllocationStart)
	if err != nil {
		return fmt.Errorf("failed to add allocation start behavior to allocation actor: %w", err)
	}

	// start actor
	if a.actorRunning {
		return nil
	}

	err = a.Actor.Start()
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
		exec, err := docker.NewExecutor(ctx, id)
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
	log.Infof("behavior allocation start from: %+v", msg.From)
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
