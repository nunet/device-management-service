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
	"fmt"
	"path/filepath"
	"strings"
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

type HealthCheckResponse struct {
	OK    bool
	Error string
}

const (
	Pending    AllocationStatus = "pending"
	Running    AllocationStatus = "running"
	Stopped    AllocationStatus = "stopped"
	Completed  AllocationStatus = "completed"
	Terminated AllocationStatus = "terminated"

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

	Actor     actor.Actor
	executor  executor.Executor
	dmsConfig config.Config

	network network.Network

	actorRunning bool

	Job        Job
	resultsDir string

	healthcheck func() error

	state struct {
		subnetIP    string
		portMapping map[int]int
	}
}

// NewAllocation creates a new allocation given the actor.
func NewAllocation(
	id string,
	fs afero.Afero,
	dmsConfig config.Config,
	actor actor.Actor,
	details AllocationDetails,
	network network.Network,
) (*Allocation, error) {
	uuid, err := uuid.NewUUID()
	if err != nil {
		return nil, fmt.Errorf("failed to create executor id: %w", err)
	}

	return &Allocation{
		ID:          id,
		fs:          fs,
		nodeID:      details.NodeID,
		sourceID:    details.SourceID,
		Job:         details.Job,
		Actor:       actor,
		executionID: uuid.String(),
		dmsConfig:   dmsConfig,
		status:      Pending,
		network:     network,
		state: struct {
			subnetIP    string
			portMapping map[int]int
		}{},
	}, nil
}

// Run creates the executor based on th e execution engine configuration.
func (a *Allocation) Run(ctx context.Context, subnetIP string, portMapping map[int]int) error {
	a.mx.Lock()
	defer a.mx.Unlock()

	if a.status == Running {
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

	a.resultsDir = filepath.Join(a.dmsConfig.WorkDir, "jobs", a.ID)
	err = a.fs.MkdirAll(a.resultsDir, 0o700)
	if err != nil {
		return fmt.Errorf("failed to create results directory: %w", err)
	}

	executionRequest := &types.ExecutionRequest{
		JobID:            a.ID,
		ExecutionID:      a.executionID,
		EngineSpec:       &a.Job.Execution,
		Resources:        &a.Job.Resources,
		ProvisionScripts: a.Job.ProvisionScripts,
		// TODO add the following
		Inputs:              []*types.StorageVolumeExecutor{}, // Question: what are those?
		Outputs:             []*types.StorageVolumeExecutor{},
		ResultsDir:          a.resultsDir,
		PersistLogsDuration: deleteLogsAfter,
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
		return fmt.Errorf("failed to start executor: %w", err)
	}

	a.status = Running

	return nil
}

// Cancel stops the running executor
func (a *Allocation) stopExecution(ctx context.Context) error {
	a.mx.Lock()
	defer a.mx.Unlock()

	if a.status != Running {
		return nil
	}

	if err := a.executor.Cancel(ctx, a.executionID); err != nil {
		return fmt.Errorf("failed to stop execution: %w", err)
	}
	log.Debugf("stopped execution: %s", a.executionID)

	a.status = Stopped

	return nil
}

func (a *Allocation) Cleanup() error {
	if err := a.executor.Remove(a.executionID, AllocationShutdownTimeout); err != nil {
		return fmt.Errorf("failed to remove execution: %w", err)
	}
	log.Debugf("removed execution: %s", a.executionID)
	return nil
}

// Terminate stops the allocation and cleans up after
func (a *Allocation) Terminate(ctx context.Context) error {
	if a.status != Stopped && a.status != Completed {
		err := a.Stop(ctx)
		if err != nil {
			log.Warnf("failed to stop allocation: %s", err)
			return fmt.Errorf("failed to stop allocation: %w", err)
		}
	}

	a.mx.Lock()
	defer a.mx.Unlock()

	if err := a.Cleanup(); err != nil {
		// TODO: exec.Remove should return a defined custom error
		//       container already removed is not an error
		log.Warnf("failed to cleanup allocation: %s", err)
		return fmt.Errorf("failed to cleanup allocation: %w", err)
	}

	a.status = Terminated

	return nil
}

// StopActor stops the allocation actor
func (a *Allocation) stopActor() error {
	a.mx.Lock()
	defer a.mx.Unlock()

	if a.actorRunning {
		if err := a.Actor.Stop(); err != nil {
			log.Warnf("error stopping allocation actor: %s", err)
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
		return fmt.Errorf("failed to stop actor: %w", err)
	}

	err = a.stopExecution(ctx)
	if err != nil {
		return fmt.Errorf("failed to stop execution: %w", err)
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

	behaviors := map[string]func(actor.Envelope){
		AllocationStartBehavior:   a.handleAllocationStart,
		AllocationRestartBehavior: a.handleAllocationRestart,

		SubnetAddPeerBehavior:         a.handleSubnetAddPeer,
		SubnetRemovePeerBehavior:      a.handleSubnetRemovePeer,
		SubnetAcceptPeerBehavior:      a.handleSubnetAcceptPeer,
		SubnetMapPortBehavior:         a.handleSubnetMapPort,
		SubnetUnmapPortBehavior:       a.handleSubnetUnmapPort,
		SubnetDNSAddRecordsBehavior:   a.handleSubnetDNSAddRecords,
		SubnetDNSRemoveRecordBehavior: a.handleSubnetDNSRemoveRecord,
		RegisterHealthcheckBehavior:   a.handleRegisterHealthcheck,
		actor.HealthCheckBehavior:     a.handleHealthcheck,
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

func (a *Allocation) ExecutionID() string {
	return a.executionID
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

	if err := a.Run(ctx, a.state.subnetIP, a.state.portMapping); err != nil {
		_ = a.Stop(ctx)
		return err
	}

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

	var req AllocationStartRequest
	if err := json.Unmarshal(msg.Message, &req); err != nil {
		log.Errorf("error unmarshalling allocation start request: %s", err)
		return
	}

	var resp AllocationStartResponse
	// TODO: context should cancel when the actor is stopped to stop monitor
	if err := a.Run(context.TODO(), req.SubnetIP, req.PortMapping); err != nil {
		err = fmt.Errorf("failed to run allocation: %w", err)
		log.Error(err)

		resp.Error = err.Error()
		resp.OK = false
		a.sendReply(msg, resp)
		return
	}

	log.Info("Running allocation's job: ", a.ID)

	a.state.subnetIP = req.SubnetIP
	a.state.portMapping = req.PortMapping

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleAllocationRestart(msg actor.Envelope) {
	defer msg.Discard()

	resp := AllocationRestartResponse{}
	if err := a.Restart(context.TODO()); err != nil { // TODO: fix context.TODO()
		err = fmt.Errorf("failed to restart allocation: %w", err)
		log.Error(err)

		resp.Error = err.Error()
		resp.OK = false
		a.sendReply(msg, resp)
		return
	}

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

	log.Debugf("added peer: %q to subnet: %q", request.PeerID, request.SubnetID)

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

	log.Debugf("accepted peer: %q to subnet: %q", request.PeerID, request.SubnetID)

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleSubnetMapPort(msg actor.Envelope) {
	defer msg.Discard()
	log.Debugf("behavior handleSubnetMapPort invoked by: %+v", msg.From)

	var request SubnetMapPortRequest
	resp := SubnetMapPortResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		log.Debugf("error unmarshalling subnet map port request: %s", err)
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.MapPort(request.SubnetID, request.Protocol, request.SourceIP, request.SourcePort, request.DestIP, request.DestPort)
	if err != nil {
		log.Debugf("error mapping port: %s", err)
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	log.Debugf("mapped port: %d to subnet: %q", request.SourcePort, request.SubnetID)

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleSubnetDNSAddRecords(msg actor.Envelope) {
	defer msg.Discard()

	var request SubnetDNSAddRecordsRequest
	resp := SubnetDNSAddRecordsResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	err := a.network.AddSubnetDNSRecords(request.SubnetID, request.Records)
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	log.Debugf("added dns record(s): %q to subnet: %q", request.Records, request.SubnetID)

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

	log.Debugf("unmapped port: %d from subnet: %q", request.SourcePort, request.SubnetID)

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

	log.Debugf("removed dns record: %q from subnet: %q", request.DomainName, request.SubnetID)

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

	log.Debugf("removed peer: %q from subnet: %q", request.PeerID, request.SubnetID)

	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleRegisterHealthcheck(msg actor.Envelope) {
	defer msg.Discard()

	var request RegisterHealthcheckRequest
	resp := RegisterHealthcheckResponse{}

	if err := json.Unmarshal(msg.Message, &request); err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	healthcheck, err := types.NewHealthCheck(request.HealthCheck, func(mf types.HealthCheckManifest) error {
		exitCode, stdout, stderr, err := a.executor.Exec(context.TODO(), a.executionID, mf.Exec)

		log.Debugf("health check command: %s\nstdout: %s\nstderr: %s", mf.Exec, stdout, stderr)
		if err != nil {
			log.Warnf("health check command failed: %s", err)
			return fmt.Errorf("health check command failed: %w", err)
		}

		if exitCode != 0 {
			log.Warnf("health check command failed with exit code: %d", exitCode)
			return fmt.Errorf("health check command failed with exit code %d", exitCode)
		}

		if !strings.Contains(stdout+stderr, mf.Response.Value) {
			log.Warnf("health check command err: %s", stderr)
			return fmt.Errorf("unexpected health check command output: %s\nstderr: %s", stdout, stderr)
		}

		log.Debugf("health check command succeeded")
		return nil
	})
	if err != nil {
		resp.Error = err.Error()
		a.sendReply(msg, resp)
		return
	}

	a.SetHealthCheck(healthcheck)
	resp.OK = true
	a.sendReply(msg, resp)
}

func (a *Allocation) handleHealthcheck(msg actor.Envelope) {
	defer msg.Discard()

	a.mx.Lock()
	healthcheck := a.healthcheck
	a.mx.Unlock()

	var resp HealthCheckResponse
	if healthcheck != nil {
		if err := healthcheck(); err != nil {
			resp.Error = err.Error()
		} else {
			resp.OK = true
		}
	} else {
		resp.OK = true
	}

	reply, err := actor.ReplyTo(msg, resp)
	if err != nil {
		log.Warnf("failed to create healthcheck reply: %s", err)
		return
	}
	if err := a.Actor.Send(reply); err != nil {
		log.Warnf("failed to send healthcheck reply: %s", err)
	}
}

func (a *Allocation) SetHealthCheck(f func() error) {
	a.mx.Lock()
	defer a.mx.Unlock()

	a.healthcheck = f
}
