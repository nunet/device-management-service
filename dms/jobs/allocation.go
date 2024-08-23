package jobs

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/google/uuid"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/executor"
	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/executor/firecracker"
	"gitlab.com/nunet/device-management-service/types"
)

// Status holds the status of an allocation.
type Status struct {
	JobResources types.ExecutionResources
	Status       AllocationStatus
}

// AllocationDetails encapsulates the dependencies to the constructor.
type AllocationDetails struct {
	Job      Job
	NodeID   string
	SourceID string
}

// AllocationStatus is a representation of the execution status
type AllocationStatus string

const (
	pending AllocationStatus = "pending"
	running AllocationStatus = "running"
	stopped AllocationStatus = "stopped"
)

// Allocation represents an allocation
type Allocation struct {
	ID          string
	Job         Job
	status      AllocationStatus
	NodeID      string
	SourceID    string
	executionID string

	actor           *dms.BasicActor
	executor        executor.Executor
	resourceManager resources.Manager
}

// NewAllocation creates a new allocation given the actor.
func NewAllocation(actor *dms.BasicActor, details AllocationDetails, resourceManager resources.Manager) (*Allocation, error) {
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
		NodeID:          details.NodeID,
		SourceID:        details.SourceID,
		actor:           actor,
		executionID:     executorID.String(),
		resourceManager: resourceManager,
		status:          pending,
	}, nil
}

// Run creates the executor based on the execution engine configuration.
func (a *Allocation) Run(ctx context.Context) error {
	freeResources, err := a.resourceManager.UpdateFreeResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to get free resources: %w", err)
	}

	if !availableResources(a.Job.Resources, freeResources) {
		return fmt.Errorf("no available resources for job %s", a.Job.ID)
	}

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

	_, err = a.resourceManager.UpdateFreeResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to update resources after running allocation's executor: %w", err)
	}

	a.status = running

	return nil
}

// Stop stops the running executor
func (a *Allocation) Stop(ctx context.Context) error {
	if a.status != running {
		return errors.New("allocation is not running")
	}

	err := a.executor.Cancel(ctx, a.executionID)
	if err != nil {
		return fmt.Errorf("failed to stop execution: %w", err)
	}

	a.status = stopped

	_, err = a.resourceManager.UpdateFreeResources(ctx)
	if err != nil {
		return fmt.Errorf("failed to update resources after stoping allocation's executor: %w", err)
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

// StartActor starts the actor of the allocation.
func (a *Allocation) StartActor() error {
	err := a.actor.Start()
	if err != nil {
		return fmt.Errorf("failed to start allocation actor: %w", err)
	}

	return nil
}

// ProcessMessages processes actor messages.
func (a *Allocation) ProcessMessages() {
	for msg := range a.actor.Messages() {
		a.dispatchMethod(msg.Type, msg.Data)
	}
}

// SendMessage sends a message through the actor.
func (a *Allocation) SendMessage(ctx context.Context, destination *dms.ActorAddrInfo, m *dms.Message) error {
	return a.actor.SendMessage(ctx, destination, m)
}

func (a *Allocation) dispatchMethod(methodName string, args ...any) {
	handlerMethod := fmt.Sprintf("Handle%s", methodName)

	arguments := make([]reflect.Value, 0)
	for _, v := range args {
		arguments = append(arguments, reflect.ValueOf(v))
	}
	method := reflect.ValueOf(a).MethodByName(handlerMethod)
	if method.IsValid() {
		method.Call(arguments)
		return
	}

	// check if actor has the method
	actorMethod := reflect.ValueOf(a.actor).MethodByName(handlerMethod)
	if actorMethod.IsValid() {
		actorMethod.Call(arguments)
	}
}

func (a *Allocation) createExecutor(ctx context.Context, conf types.SpecConfig) error {
	if conf.Type == types.ExecutorTypeFirecracker {
		executor, err := firecracker.NewExecutor(ctx, a.executionID)
		if err != nil {
			return fmt.Errorf("firecracker executor: %w", err)
		}
		a.executor = executor
	} else if conf.Type == types.ExecutorTypeDocker {
		executor, err := docker.NewExecutor(ctx, a.executionID)
		if err != nil {
			return fmt.Errorf("docker executor: %w", err)
		}
		a.executor = executor
	}

	return nil
}

// TODO: ExecutionResources and FreeResources should be compatible
func availableResources(jobResources types.ExecutionResources, fr types.FreeResources) bool {
	return fr.RAM >= uint64(jobResources.Memory.Size) && fr.Disk >= jobResources.Disk.Size && fr.CPU > float64(jobResources.CPU.Cores)
}
