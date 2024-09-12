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
	"gitlab.com/nunet/device-management-service/executor/firecracker"
	"gitlab.com/nunet/device-management-service/types"
)

const (
	pending AllocationStatus = "pending"
	running AllocationStatus = "running"
	stopped AllocationStatus = "stopped"
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

// Allocation represents an allocation
type Allocation struct {
	ID string

	mx sync.Mutex

	status      AllocationStatus
	nodeID      string
	sourceID    string
	executionID string

	Actor           *actor.BasicActor
	executor        executor.Executor
	resourceManager types.ResourceManager

	actorRunning bool

	Job Job
}

// NewAllocation creates a new allocation given the actor.
func NewAllocation(actor *actor.BasicActor, details AllocationDetails, resourceManager types.ResourceManager) (*Allocation, error) {
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
		Job:             details.Job,
		nodeID:          details.NodeID,
		sourceID:        details.SourceID,
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
		return nil
	}

	resourceAllocation := types.ResourceAllocation{JobID: a.Job.ID, Resources: a.Job.Resources}
	err := a.resourceManager.AllocateResources(ctx, resourceAllocation)
	if err != nil {
		return fmt.Errorf("failed to allocate resources: %w", err)
	}

	defer func() {
		if a.status != running {
			// If not running, ensure deallocation of resources
			err = a.resourceManager.DeallocateResources(ctx, a.Job.ID)
		}
	}()

	// if executor is nil create it
	if a.executor == nil {
		err = a.createExecutor(ctx, a.Job.Execution)
		if err != nil {
			return fmt.Errorf("failed to create executor: %w", err)
		}
	}

	err = a.executor.Start(ctx, &types.ExecutionRequest{
		JobID:       a.Job.ID,
		ExecutionID: a.executionID,
		EngineSpec:  &a.Job.Execution,
		Resources:   &a.Job.Resources,
		// TODO add the following
		Inputs:     []*types.StorageVolumeExecutor{},
		Outputs:    []*types.StorageVolumeExecutor{},
		ResultsDir: "",
	})
	if err != nil {
		return fmt.Errorf("failed to start executor: %w", err)
	}

	a.status = running
	return nil
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

	if err := a.resourceManager.DeallocateResources(ctx, a.Job.ID); err != nil {
		return fmt.Errorf("failed to deallocate resources: %w", err)
	}

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

func (a *Allocation) createExecutor(ctx context.Context, conf types.SpecConfig) error {
	if conf.Type == string(types.ExecutorTypeFirecracker) {
		executor, err := firecracker.NewExecutor(ctx, a.executionID)
		if err != nil {
			return fmt.Errorf("firecracker executor: %w", err)
		}
		a.executor = executor
	} else if conf.Type == string(types.ExecutorTypeDocker) {
		executor, err := docker.NewExecutor(ctx, a.executionID)
		if err != nil {
			return fmt.Errorf("docker executor: %w", err)
		}
		a.executor = executor
	}

	return nil
}
