// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

// keep as var instead of consts so that we change the values in tests
var (
	BidRequestTimeout           = 5 * time.Second
	CommitDeploymentTimeout     = 3 * time.Second
	VerifyEdgeConstraintTimeout = 5 * time.Second
	AllocationDeploymentTimeout = 5 * time.Second

	// Setting a big timeout as the user might have to
	// download large execution images
	AllocationStartTimeout    = 5 * time.Minute
	AllocationShutdownTimeout = 5 * time.Second

	MinEnsembleDeploymentTime = 15 * time.Second
	MinEnsembleUpdateTimeout  = 15 * time.Second

	SubnetCreateTimeout  = 2 * time.Minute
	SubnetDestroyTimeout = 30 * time.Second

	MaxBidMultiplier = 8
	MaxPermutations  = 1_000_000

	grantOrchestratorCapsFrequency = 5 * time.Minute
)

var (
	ErrProvisioningFailed   = errors.New("failed to provision the ensemble")
	ErrDeploymentFailed     = errors.New("failed to create deployment")
	ErrOrchestratorExists   = errors.New("orchestrator with ID already exists")
	ErrOrchestratorNotFound = errors.New("orchestrator with ID not found")
)

// Orchestrator is the interface for orchestrating deployments
type Orchestrator interface {
	Deploy(expiry time.Time) error
	Update(cfg jtypes.EnsembleConfig, expiry time.Time) error
	Shutdown() error
	Stop()
	GetAllocationLogs(allocationID string) (AllocationLogsResponse, error)
	WriteAllocationLogs(allocationID string, stdout, stderr []byte) (string, error)
	StatusChannel(ctx context.Context) <-chan jtypes.DeploymentStatus
	Status() jtypes.DeploymentStatus
	Manifest() jtypes.EnsembleManifest
	Config() jtypes.EnsembleConfig
	ID() string
	ActorPrivateKey() crypto.PrivKey
	DeploymentSnapshot() jtypes.DeploymentSnapshot
}

type BasicOrchestrator struct {
	lock   sync.Mutex
	ctx    context.Context
	cancel func()

	fs      afero.Afero
	workDir string
	actor   actor.Actor

	id             string
	cfg            jtypes.EnsembleConfig
	manifest       jtypes.EnsembleManifest
	subnetManifest SubnetManifest
	status         jtypes.DeploymentStatus

	deploymentSnapshot jtypes.DeploymentSnapshot
	supervisor         *Supervisor

	// Status subscribers
	statusSubscribers     map[chan jtypes.DeploymentStatus]struct{}
	statusSubscribersLock sync.RWMutex
}

var _ Orchestrator = (*BasicOrchestrator)(nil)

func NewOrchestrator(
	ctx context.Context,
	fs afero.Afero,
	workDir string,
	id string,
	oActor actor.Actor,
	cfg jtypes.EnsembleConfig,
) (*BasicOrchestrator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate ensemble configuration: %w", err)
	}

	subnet, err := newSubnetManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet manifest: %w", err)
	}

	o := &BasicOrchestrator{
		actor:             oActor,
		id:                id,
		cfg:               cfg,
		ctx:               ctx,
		fs:                fs,
		workDir:           workDir,
		subnetManifest:    subnet,
		supervisor:        NewSupervisor(ctx, oActor, id),
		statusSubscribers: make(map[chan jtypes.DeploymentStatus]struct{}),
	}

	orchestratorBehaviors := map[string]func(actor.Envelope){
		behaviors.NotifyTaskTerminationBehavior: o.handleTaskTermination,
	}

	for b, handler := range orchestratorBehaviors {
		err := o.actor.AddBehavior(b, handler)
		if err != nil {
			return nil, fmt.Errorf("add allocation start behavior to allocation actor: %w", err)
		}
	}

	return o, nil
}

func (o *BasicOrchestrator) setStatus(status jtypes.DeploymentStatus) {
	o.lock.Lock()
	defer o.lock.Unlock()

	log.Infow("orchestrator_status_updated",
		"labels", []string{string(observability.LabelDeployment)},
		"status", status.String(),
		"orchestratorID", o.id)
	oldStatus := o.status
	o.status = status

	if oldStatus != status {
		// Notify all subscribers
		o.statusSubscribersLock.RLock()
		defer o.statusSubscribersLock.RUnlock()
		for ch := range o.statusSubscribers {
			select {
			case ch <- status:
			default:
				// Skip if channel is blocked
			}
		}
	}

	// If we've reached a terminal state, close all subscriber channels
	if status == jtypes.DeploymentStatusRunning ||
		status == jtypes.DeploymentStatusFailed ||
		status == jtypes.DeploymentStatusCompleted {
		for ch := range o.statusSubscribers {
			close(ch)
		}
		o.statusSubscribers = make(map[chan jtypes.DeploymentStatus]struct{})
	}
}

func (o *BasicOrchestrator) StatusChannel(ctx context.Context) <-chan jtypes.DeploymentStatus {
	ch := make(chan jtypes.DeploymentStatus, 1)

	// Send initial status
	select {
	case ch <- o.Status():
	case <-ctx.Done():
		close(ch)
		return ch
	}

	o.statusSubscribersLock.Lock()
	o.statusSubscribers[ch] = struct{}{}
	o.statusSubscribersLock.Unlock()

	// Clean up when context is done
	go func() {
		<-ctx.Done()
		o.statusSubscribersLock.Lock()
		delete(o.statusSubscribers, ch)
		o.statusSubscribersLock.Unlock()
		select {
		case <-ch:
		default:
			close(ch)
		}
	}()

	return ch
}

func (o *BasicOrchestrator) Deploy(expiry time.Time) error {
	defer func() {
		if o.status != jtypes.DeploymentStatusRunning {
			o.setStatus(jtypes.DeploymentStatusFailed)
		}
	}()
	o.setStatus(jtypes.DeploymentStatusPreparing)

	log.Debugw("initializing manifest",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", o.id)
	o.manifest = o.newManifest(o.cfg)

	if err := o.deploy(o.cfg, o.manifest, expiry); err != nil {
		return fmt.Errorf("deploying ensemble: %w", err)
	}

	log.Infof("deployment successful, starting supervisor",
		"orchestratorID", o.id)
	go o.supervisor.Supervise(jtypes.NewManifestReader(o.manifest))
	return nil
}

func (o *BasicOrchestrator) newManifest(
	cfg jtypes.EnsembleConfig,
) jtypes.EnsembleManifest {
	log.Debugf("creating new manifest based on config %+v", cfg.V1)
	manifest := jtypes.EnsembleManifest{
		ID:           o.id,
		Orchestrator: o.actor.Handle(),
		Metadata:     cfg.V1.Metadata,
		Allocations:  make(map[string]jtypes.AllocationManifest),
		Nodes:        make(map[string]jtypes.NodeManifest),
		Contracts:    make(map[string]jtypes.ContractManifest),
	}

	for name, v := range cfg.Contracts() {
		manifest.Contracts[name] = jtypes.ContractManifest{
			ID:   name,
			DID:  v.DID,
			Host: v.Host,
		}
	}

	for name, alloc := range cfg.Allocations() {
		amf := jtypes.AllocationManifest{
			ID:          types.ConstructAllocationID(o.id, name),
			DNSName:     alloc.DNSName + ".internal",
			Healthcheck: alloc.HealthCheck,
			Status:      jtypes.AllocationPending,
			Ports:       make(map[int]int),
			Type:        alloc.Type,
		}
		manifest.Allocations[name] = amf
	}
	for name, node := range cfg.Nodes() {
		nmf := jtypes.NodeManifest{
			ID:          name,
			Allocations: node.Allocations,
			Peer:        node.Peer,
		}
		manifest.Nodes[name] = nmf
	}

	manifest.Subnet = cfg.V1.Subnet

	return manifest
}

// TODO (dynamic ensemble PR): documentation on how updates
// and revert handle manifest changes
//
// IMPORTANT: when passing the manifest and config down the stack,
// use the readers (`jobs/types/readers.go`) to guarantee the immutability
// of these objects. (that is not to solve race condition problems but
// to manage the state of the orchestrator in a safer way)
func (o *BasicOrchestrator) deploy(
	cfg jtypes.EnsembleConfig,
	partialManifest jtypes.EnsembleManifest,
	expiry time.Time,
) error {
	o.deploymentSnapshot.Expiry = expiry

deploy:
	for time.Now().Before(expiry) {
		o.setStatus(jtypes.DeploymentStatusPreparing)

		// delete old state of candidates if any
		for c := range o.deploymentSnapshot.Candidates {
			o.lock.Lock()
			delete(o.deploymentSnapshot.Candidates, c)
			o.lock.Unlock()
		}

		// 1. bid
		bidCoordinator, err := NewBidCoordinator(o.id, o.actor)
		if err != nil {
			return fmt.Errorf("failed to create bidder: %w", err)
		}

		candidateDeployment, err := bidCoordinator.bid(jtypes.NewEnsembleCfgReader(cfg), o.deploymentSnapshot.Candidates, expiry)
		if err != nil {
			if errors.Is(err, ErrCandidateNotFound) {
				log.Warnf("candidate deployment not found, redeploying: %v", err)
				continue deploy
			}

			log.Errorf("failed to bid: %v", err)
			return fmt.Errorf("failed to bid: %w", err)
		}

		// 2. Commit the deployment
		o.deploymentSnapshot.Candidates = candidateDeployment
		o.setStatus(jtypes.DeploymentStatusCommitting)

		committer := NewCommitter(o.ctx, o.id, o.actor)

		manifestAfterCommit, err := committer.commit(
			jtypes.NewEnsembleCfgReader(cfg),
			jtypes.NewManifestReader(partialManifest),
			candidateDeployment,
		)
		if err != nil {
			log.Warnw("failed to commit deployment",
				"labels", []string{string(observability.LabelDeployment)},
				"orchestratorID", o.id,
				"error", err)

			for nodeName, n := range manifestAfterCommit.Nodes {
				o.revertNodeDeployment(cfg, nodeName, n.Handle)
			}
			continue deploy
		}

		o.updateManifest(manifestAfterCommit)

		mnfJSON, err := manifestAfterCommit.JSON()
		if err != nil {
			return fmt.Errorf("failed to marshal manifest: %w", err)
		}
		log.Debugf("manifest after commit:\n", string(mnfJSON))

		// 3. provision the network and start the allocations
		o.setStatus(jtypes.DeploymentStatusProvisioning)

		provisioner := NewProvisioner(o.ctx, o.cancel, o.actor, o.subnetManifest)
		manifestAfterProvision, err := provisioner.Provision(
			jtypes.NewEnsembleCfgReader(cfg),
			jtypes.NewManifestReader(manifestAfterCommit))
		if err != nil {
			log.Errorw("provisioning failed",
				"labels", []string{string(observability.LabelDeployment)},
				"error", err,
				"orchestratorID", o.id)

			o.lock.Lock()
			o.revert(cfg, manifestAfterCommit)
			o.lock.Unlock()
			continue deploy
		}

		go o.monitorOnlyTaskManifest()
		o.updateManifest(manifestAfterProvision)

		o.lock.Lock()
		o.ctx, o.cancel = context.WithCancel(context.Background())
		o.lock.Unlock()

		log.Infof("deployment successful")
		o.setStatus(jtypes.DeploymentStatusRunning)

		return nil
	}

	// we failed to create the deployment in time
	log.Errorw("deployment creation timed out",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", o.id)
	return ErrDeploymentFailed
}

// Stop stops the orchestrator
func (o *BasicOrchestrator) Stop() {
	// TODO

	err := o.actor.Stop()
	if err != nil {
		log.Warnf("error stopping orchestrator's actor: %s", err)
	}
}

type AllocationLogsRequest struct {
	AllocName string
}

type AllocationLogsResponse struct {
	Stdout []byte
	Stderr []byte
	Error  string
}

func (o *BasicOrchestrator) GetAllocationLogs(name string) (AllocationLogsResponse, error) {
	var allocNodeHandle actor.Handle
	var logsResp AllocationLogsResponse
	for _, n := range o.manifest.Nodes {
		if ok := utils.SliceContains(n.Allocations, name); ok {
			allocNodeHandle = n.Handle
			break
		}
	}

	if allocNodeHandle.Empty() {
		return logsResp,
			fmt.Errorf(
				"node not found for allocation %s of ensemble %s",
				name, o.id,
			)
	}

	msg, err := actor.Message(
		o.actor.Handle(),
		allocNodeHandle,
		fmt.Sprintf(behaviors.AllocationLogsBehavior.DynamicTemplate, o.manifest.ID),
		AllocationLogsRequest{
			AllocName: name,
		},
		actor.WithMessageExpiry(uint64(time.Now().Add(2*time.Minute).UnixNano())),
	)
	if err != nil {
		return logsResp, fmt.Errorf("creating get logs message: %w", err)
	}

	replyCh, err := o.actor.Invoke(msg)
	if err != nil {
		return logsResp, fmt.Errorf("invoking get logs message: %w", err)
	}

	var reply actor.Envelope
	select {
	case reply = <-replyCh:
	case <-time.After(2 * time.Minute):
		return logsResp, fmt.Errorf("timeout getting logs for %s: %w", name, ErrDeploymentFailed)
	}

	defer reply.Discard()

	if err := json.Unmarshal(reply.Message, &logsResp); err != nil {
		return logsResp, fmt.Errorf("unmarshalling get logs response: %w", err)
	}

	if logsResp.Error != "" {
		return logsResp, fmt.Errorf("replied with error getting logs for %s: %s", name, logsResp.Error)
	}

	return logsResp, nil
}

func (o *BasicOrchestrator) Status() jtypes.DeploymentStatus {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.status
}

func (o *BasicOrchestrator) Manifest() jtypes.EnsembleManifest {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.manifest.Clone()
}

func (o *BasicOrchestrator) ManifestNodesPeerIDs() []string {
	o.lock.Lock()
	defer o.lock.Unlock()

	ids := make([]string, len(o.manifest.Nodes))
	for _, n := range o.manifest.Nodes {
		ids = append(ids, n.Peer)
	}

	return ids
}

func (o *BasicOrchestrator) Config() jtypes.EnsembleConfig {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.cfg.Clone()
}

func (o *BasicOrchestrator) ID() string {
	return o.id
}

func (o *BasicOrchestrator) ActorPrivateKey() crypto.PrivKey {
	return o.actor.Security().PrivKey()
}

func (o *BasicOrchestrator) DeploymentSnapshot() jtypes.DeploymentSnapshot {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.deploymentSnapshot
}

func (o *BasicOrchestrator) updateManifest(m jtypes.EnsembleManifest) {
	o.lock.Lock()
	// cloning since the orchestrator original manifest state
	// might inherit map references of partial updates
	o.manifest = m.Clone()
	o.lock.Unlock()
}

// monitorOnlyTaskManifest will be responsible for tearing down
// the orchestrator after all tasks are terminated when
// the ensemble is composed *ONLY* by tasks
func (o *BasicOrchestrator) monitorOnlyTaskManifest() {
	if !isOnlyTaskManifest(o.manifest) {
		return
	}

	ticker := time.NewTicker(monitorOnlyTaskManifestInterval)
	defer ticker.Stop()
	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			o.lock.Lock()
			allTerminated := true
			for name := range o.manifest.Allocations {
				if !o.manifest.IsTerminatedTask(name) {
					allTerminated = false
					break
				}
			}
			o.lock.Unlock()

			if !allTerminated {
				continue
			}

			log.Infof("All tasks are terminated, shutting down orchestrator.")
			o.setStatus(jtypes.DeploymentStatusCompleted)
			o.cancel()
			return
		}
	}
}

func isOnlyTaskManifest(m jtypes.EnsembleManifest) bool {
	for _, a := range m.Allocations {
		if a.Type != jtypes.AllocationTypeTask {
			return false
		}
	}
	return true
}
