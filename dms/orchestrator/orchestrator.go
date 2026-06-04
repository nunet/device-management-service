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
	"strings"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/afero"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/dms/behaviors"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/tokenomics/eventhandler"
	"gitlab.com/nunet/device-management-service/tokenomics/events"
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
	AllocationShutdownTimeout = 15 * time.Second

	MinEnsembleDeploymentTime = 15 * time.Second
	MinEnsembleUpdateTimeout  = 15 * time.Second

	SubnetCreateTimeout  = 2 * time.Minute
	SubnetDestroyTimeout = 10 * time.Second

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
	SubnetManifest() jtypes.SubnetManifest
	Config() jtypes.EnsembleConfig
	ID() string
	ActorPrivateKey() crypto.PrivKey
	Actor() actor.Actor
	DeploymentSnapshot() jtypes.DeploymentSnapshot
	AllocationInfo() map[string]jtypes.AllocationInfo
	UpdateAllocationStatus()
	Done() <-chan struct{}
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
	subnetManifest jtypes.SubnetManifest
	status         jtypes.DeploymentStatus
	allocs         map[string]jtypes.AllocationInfo

	deploymentSnapshot jtypes.DeploymentSnapshot
	supervisor         *Supervisor

	// ID generators
	nodeIDGenerator       types.NodeIDGenerator
	allocationIDGenerator types.AllocationIDGenerator

	// Status subscribers
	statusSubscribers     map[chan jtypes.DeploymentStatus]struct{}
	statusSubscribersLock sync.RWMutex

	contractEventHandler *eventhandler.EventHandler
	contracts            map[string]types.ContractConfig
}

var _ Orchestrator = (*BasicOrchestrator)(nil)

func NewOrchestrator(
	ctx context.Context,
	fs afero.Afero,
	workDir string,
	id string,
	oActor actor.Actor,
	cfg jtypes.EnsembleConfig,
	nodeIDGenerator types.NodeIDGenerator,
	allocationIDGenerator types.AllocationIDGenerator,
	contractEventHandler *eventhandler.EventHandler,
	contracts map[string]types.ContractConfig,
) (*BasicOrchestrator, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("failed to validate ensemble configuration: %w", err)
	}

	// Validate generators at instantiation time
	validator := types.NewDefaultGeneratorValidator()

	if err := validator.ValidateNodeIDGenerator(nodeIDGenerator); err != nil {
		return nil, fmt.Errorf("invalid node ID generator: %w", err)
	}

	if err := validator.ValidateAllocationIDGenerator(allocationIDGenerator); err != nil {
		return nil, fmt.Errorf("invalid allocation ID generator: %w", err)
	}

	subnet, err := newSubnetManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet manifest: %w", err)
	}

	childCtx, childCancel := context.WithCancel(ctx)
	o := &BasicOrchestrator{
		actor:                 oActor,
		id:                    id,
		cfg:                   cfg,
		ctx:                   childCtx,
		cancel:                childCancel,
		fs:                    fs,
		workDir:               workDir,
		subnetManifest:        subnet,
		allocs:                make(map[string]jtypes.AllocationInfo),
		supervisor:            NewSupervisor(childCtx, oActor, id),
		nodeIDGenerator:       nodeIDGenerator,
		allocationIDGenerator: allocationIDGenerator,
		statusSubscribers:     make(map[chan jtypes.DeploymentStatus]struct{}),
		contractEventHandler:  contractEventHandler,
		contracts:             contracts,
	}
	o.supervisor.SetAllocationStatusUpdater(o.updateAllocationStatusFromSupervisor)

	err = o.RegisterBehaviors()
	if err != nil {
		return nil, fmt.Errorf("failed to register behaviors: %w", err)
	}

	return o, nil
}

func (o *BasicOrchestrator) updateAllocationStatusFromSupervisor(allocationID string, status jtypes.AllocationStatus) {
	o.lock.Lock()
	if info, ok := o.allocs[allocationID]; ok {
		info.Status = status
		info.Timestamp = time.Now().Unix()
		o.allocs[allocationID] = info
	}
	o.lock.Unlock()

	allocID, err := types.ParseAllocationID(allocationID)
	if err != nil {
		log.Debugf("failed to parse allocation ID %s: %v", allocationID, err)
		return
	}

	manifest := o.Manifest()
	if alloc, ok := manifest.Allocations[allocID.ManifestKey()]; ok {
		alloc.Status = status
		manifest.Allocations[allocID.ManifestKey()] = alloc
		o.updateManifest(manifest)
	}
}

func (o *BasicOrchestrator) SetStatus(status jtypes.DeploymentStatus) {
	o.setStatus(status)
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
	if status == jtypes.DeploymentStatusFailed ||
		status == jtypes.DeploymentStatusCompleted {
		for ch := range o.statusSubscribers {
			close(ch)
		}
		o.statusSubscribers = make(map[chan jtypes.DeploymentStatus]struct{})
	}

	// metrics
	if m := observability.DeploymentStatus; m != nil {
		m.Add(o.ctx, 1, metric.WithAttributes(
			observability.AttrDID,
			attribute.String("orchestratorID", o.id),
			attribute.String("status", status.String()),
		))
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

	log.Infow("initializing manifest",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", o.id)

	o.manifest = o.newManifest(o.cfg)

	if err := o.deploy(o.cfg, o.manifest, expiry); err != nil {
		return fmt.Errorf("deploying ensemble: %w", err)
	}

	for _, a := range o.Manifest().Allocations {
		err := o.grantOrchestratorCaps(a.Handle.DID)
		if err != nil {
			return fmt.Errorf("failed to grant orchestrator capabilities to allocations: %w", err)
		}
	}

	log.Infow("deployment successful, starting supervisor",
		"labels", []string{string(observability.LabelDeployment)},
		"orchestratorID", o.id)
	go o.supervisor.Supervise(jtypes.NewManifestReader(o.manifest))

	for _, v := range o.contracts {
		evt := events.DeploymentStart{
			EventBase:       events.EventBase{Type: events.DeploymentStartEvent},
			DeploymentID:    o.manifest.ID,
			OrchestratorID:  o.id,
			HeadContractDID: v.DID, // treat contrat as if head of contract chain, won't be taken into consideration in billing if contract is p2p
		}
		o.contractEventHandler.Push(eventhandler.Event{
			ContractHostDID: v.Host,
			ContractDID:     v.DID,
			Payload:         evt,
		})
	}
	return nil
}

func (o *BasicOrchestrator) NewManifest(
	cfg jtypes.EnsembleConfig,
) jtypes.EnsembleManifest {
	return o.newManifest(cfg)
}

func (o *BasicOrchestrator) newManifest(
	cfg jtypes.EnsembleConfig,
) jtypes.EnsembleManifest {
	log.Debugw("creating new manifest",
		"labels", []string{string(observability.LabelDeployment)},
		"config", cfg.V1)
	manifest := jtypes.EnsembleManifest{
		ID:           o.id,
		Orchestrator: o.actor.Handle(),
		Metadata:     cfg.V1.Metadata,
		Allocations:  make(map[string]jtypes.AllocationManifest),
		Nodes:        make(map[string]jtypes.NodeManifest),
		Contracts:    make(map[string]jtypes.ContractManifest),
		Subnet:       cfg.V1.Subnet,
	}

	for name, v := range cfg.Contracts() {
		manifest.Contracts[name] = jtypes.ContractManifest{
			ID:   name,
			DID:  v.DID,
			Host: v.Host,
		}
	}

	for name, node := range cfg.NodesWithGenerator(o.nodeIDGenerator) {
		nodeAllocations := make([]string, 0)
		for _, allocName := range node.Allocations {
			_, ok := cfg.Allocation(allocName)
			if !ok {
				log.Errorw("allocation not found in ensemble config, skipping",
					"labels", []string{string(observability.LabelAllocation)},
					"allocation", allocName)
				continue
			}

			// Generate manifest key using generator
			allocKey, err := o.allocationIDGenerator.GenerateManifestKey(name, allocName)
			if err != nil {
				log.Errorf("failed to generate manifest key for %s.%s: %v", name, allocName, err)
				continue
			}
			nodeAllocations = append(nodeAllocations, allocKey)
		}

		standbyNodes := make([]string, 0)
		if node.Redundancy > 0 {
			for i := 1; i <= node.Redundancy; i++ {
				standbyNodeID, err := o.nodeIDGenerator.GenerateStandbyNodeID(name, i)
				if err != nil {
					log.Errorf("failed to generate standby node ID for %s-%d: %v", name, i, err)
					continue
				}
				standbyNodes = append(standbyNodes, standbyNodeID)
			}
		}

		// Create primary node entry
		nmf := jtypes.NodeManifest{
			ID:           name,
			Allocations:  nodeAllocations,
			Peer:         node.Peer,
			StandbyNodes: standbyNodes,
		}
		manifest.Nodes[name] = nmf
	}

	// Now create allocation entries
	for nodeID, nodeManifest := range manifest.Nodes {
		for _, allocKey := range nodeManifest.Allocations {
			parts := strings.Split(allocKey, ".")
			if len(parts) != 2 {
				log.Errorf("invalid allocation key format: %s, skipping", allocKey)
				continue
			}
			configAllocName := parts[1]
			alloc, ok := cfg.Allocation(configAllocName)
			if !ok {
				log.Errorf("allocation %s not found in ensemble config, skipping", configAllocName)
				continue
			}

			isStandby := nodeManifest.RedundancyRole == jtypes.RoleStandby

			// Generate full allocation ID using generator
			fullAllocID, err := o.allocationIDGenerator.GenerateFullAllocationID(o.id, nodeID, configAllocName)
			if err != nil {
				log.Errorf("failed to generate full allocation ID for %s.%s: %v", nodeID, configAllocName, err)
				continue
			}

			amf := jtypes.AllocationManifest{
				ID:              fullAllocID,
				Type:            alloc.Type,
				NodeID:          nodeID,
				DNSName:         alloc.DNSName + ".internal",
				Healthcheck:     alloc.HealthCheck,
				Status:          jtypes.AllocationPending,
				Ports:           make(map[int]int),
				RedundancyGroup: configAllocName,
				IsStandby:       isStandby,
			}
			manifest.Allocations[allocKey] = amf
		}
	}

	return manifest
}

func (o *BasicOrchestrator) invokeBehaviour(destination actor.Handle, behavior string, payload any, timeout time.Duration) (actor.Envelope, error) {
	msg, err := actor.Message(
		o.actor.Handle(),
		destination,
		behavior,
		payload,
		actor.WithMessageExpiry(actor.MakeExpiry(timeout)),
	)
	if err != nil {
		return actor.Envelope{}, fmt.Errorf("failed to create contract actor message: %w", err)
	}

	replyCh, err := o.actor.Invoke(msg)
	if err != nil {
		return actor.Envelope{}, fmt.Errorf("failed to invoke message: %w", err)
	}

	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	var reply actor.Envelope
	select {
	case reply = <-replyCh:
		defer reply.Discard()

		return reply, nil

	case <-ticker.C:
		return actor.Envelope{}, errors.New("failed to receive reply due to timeout")
	}
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

		for key, v := range candidateDeployment {
			if v.V1.PromiseBid {
				// wait for provisioning
				pb := jtypes.PromiseBidRequest{
					Bid: v,
				}
				envelope, err := o.invokeBehaviour(v.V1.Handle, behaviors.PromiseBidToBidBehavior, pb, time.Minute*5)
				if err != nil {
					log.Errorf("failed to convert promise bid: %v", err)
					return fmt.Errorf("failed to convert promise bid: %w", err)
				}

				var newBid jtypes.ConvertedPromiseBidResponse
				err = json.Unmarshal(envelope.Message, &newBid)
				if err != nil {
					log.Errorf("failed to unmarshal new bid: %v", err)
					return fmt.Errorf("failed to unmarshal new bid: %w", err)
				}

				// replace the current bid with the new bid
				candidateDeployment[key] = newBid.Bid
			}
		}

		// 2. Commit the deployment
		o.deploymentSnapshot.Candidates = candidateDeployment
		o.setStatus(jtypes.DeploymentStatusCommitting)

		committer := NewCommitter(o.ctx, o.id, o.actor, o.allocationIDGenerator, o.nodeIDGenerator)

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

		log.Debugw("manifest after commit",
			"labels", []string{string(observability.LabelDeployment)},
			"manifest", manifestAfterCommit)

		// 3. provision the network and start the allocations
		o.setStatus(jtypes.DeploymentStatusProvisioning)

		provisioner := NewProvisioner(o.ctx, o.cancel, o.actor, o.subnetManifest, o.allocationIDGenerator)
		manifestAfterProvision, err := provisioner.Provision(
			jtypes.NewEnsembleCfgReader(cfg),
			jtypes.NewManifestReader(manifestAfterCommit))
		if err != nil {
			log.Errorw("provisioning failed",
				"labels", []string{string(observability.LabelDeployment)},
				"error", err,
				"orchestratorID", o.id)

			o.revert(cfg, manifestAfterCommit)
			continue deploy
		}

		go o.monitorOnlyTaskManifest()
		o.updateManifest(manifestAfterProvision)

		log.Infof("deployment successful")
		o.setStatus(jtypes.DeploymentStatusRunning)

		var allocated types.Resources
		for idx, a := range o.Manifest().Allocations {
			res := o.Config().V1.Allocations[a.ID].Resources
			o.allocs[a.ID] = jtypes.AllocationInfo{
				AllocationID:   a.ID,
				HeartbeatSeq:   0,
				Status:         a.Status,
				HasHealthCheck: len(a.Healthcheck.Exec) != 0 && a.Healthcheck.Type != "",
				ResourceLimit:  res,
				DNSName:        a.DNSName,
				IP:             o.SubnetManifest().IndexRoutingTable[idx],
				ResourceUsage:  jtypes.AllocationResourceUsage{},
				Timestamp:      time.Now().Unix(),
			}

			_ = allocated.RAM.Add(res.RAM)
			_ = allocated.Disk.Add(res.Disk)
			_ = allocated.CPU.Add(res.CPU)
			_ = allocated.GPUs.Add(res.GPUs)
		}

		// metric
		if m := observability.DeploymentSuccess; m != nil {
			m.Add(o.ctx, 1, metric.WithAttributes(
				observability.AttrDID,
				// attribute.Int("allocations", len(o.Manifest().Allocations)),
			))

			if m := observability.DeploySuccessAllocations; m != nil {
				m.Record(o.ctx, int64(len(o.Manifest().Allocations)), metric.WithAttributes(
					observability.AttrDID,
					attribute.String("orchestratorID", o.id),
				))
			}
			if m := observability.DeploySuccessCPUCoresAssigned; m != nil {
				m.Record(o.ctx, float64(allocated.CPU.Cores), metric.WithAttributes(
					observability.AttrDID,
					attribute.String("orchestratorID", o.id),
				))
			}
			if m := observability.DeploySuccessRAMGBAssigned; m != nil {
				m.Record(o.ctx, int64(allocated.RAM.SizeInGB()), metric.WithAttributes(
					observability.AttrDID,
					attribute.String("orchestratorID", o.id),
				))
			}
			if m := observability.DeploySuccessDiskMBAssigned; m != nil {
				m.Record(o.ctx, float64(allocated.Disk.Size/(1024.0*1024.0)), metric.WithAttributes(
					observability.AttrDID,
					attribute.String("orchestratorID", o.id),
				))
			}
			if m := observability.DeploySuccessGPUCountAssigned; m != nil {
				m.Record(o.ctx, int64(len(allocated.GPUs)), metric.WithAttributes(
					observability.AttrDID,
					attribute.String("orchestratorID", o.id),
				))
			}
		}

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

	o.cancel()

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

func (o *BasicOrchestrator) SubnetManifest() jtypes.SubnetManifest {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.subnetManifest
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
	o.UpdateAllocationStatus()
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

func (o *BasicOrchestrator) AllocationInfo() map[string]jtypes.AllocationInfo {
	o.lock.Lock()
	defer o.lock.Unlock()

	allocsCopy := make(map[string]jtypes.AllocationInfo, len(o.allocs))
	for k, v := range o.allocs {
		allocsCopy[k] = v
	}
	return allocsCopy
}

func (o *BasicOrchestrator) UpdateAllocationStatus() {
	manifest := o.Manifest()
	if manifest.Allocations == nil {
		return
	}

	o.lock.Lock()
	defer o.lock.Unlock()

	for _, a := range manifest.Allocations {
		if _, ok := o.allocs[a.ID]; !ok {
			continue
		}

		allocInfo := o.allocs[a.ID]
		allocInfo.Status = a.Status
		o.allocs[a.ID] = allocInfo
	}
}

func (o *BasicOrchestrator) RegisterBehaviors() error {
	orchestratorBehaviors := map[string]func(actor.Envelope){
		fmt.Sprintf(
			behaviors.NotifyTaskTerminationBehavior,
			o.id): o.handleTaskTermination,
		fmt.Sprintf(
			behaviors.NotifyAllocationLivenessBehavior,
			o.id): o.handleAllocationLiveness,
		fmt.Sprintf(
			behaviors.NotifyAllocationStatusBehavior,
			o.id): o.handleAllocationStatusUpdate,
		fmt.Sprintf(
			behaviors.DeploymentStateBehavior,
			o.id): o.handleDeploymentState,
	}

	for b, handler := range orchestratorBehaviors {
		err := o.actor.AddBehavior(b, handler)
		if err != nil {
			return fmt.Errorf("add behavior %s to orchestrator actor: %w", b, err)
		}
	}
	return nil
}

func (o *BasicOrchestrator) grantOrchestratorCaps(alloc did.DID) error {
	log.Infow("granting alloc capabilities",
		"orchestratorID", o.id,
		"allocationDID", alloc.String(),
	)
	oDID, err := did.FromID(o.actor.Handle().ID)
	if err != nil {
		return fmt.Errorf("failed to parse orchestrator DID: %w", err)
	}

	err = o.actor.Security().Grant(
		alloc,
		oDID,
		[]ucan.Capability{
			ucan.Capability(fmt.Sprintf(behaviors.OrchestratorEnsembleNamespace, o.ID())),
		},
		grantOrchestratorCapsFrequency,
	)
	if err != nil {
		return fmt.Errorf(
			"granting orchestrator caps to alloc %s: %w",
			alloc.String(), err)
	}

	// TODO: create helper func to periodically grant caps as
	// it's being used here and on createAllocations()
	go func() {
		ticker := time.NewTicker(grantOrchestratorCapsFrequency)
		defer ticker.Stop()

		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			err := o.actor.Security().Grant(
				alloc,
				o.actor.Handle().DID,
				[]ucan.Capability{
					ucan.Capability(
						fmt.Sprintf(behaviors.OrchestratorEnsembleNamespace, o.ID())),
				},
				grantOrchestratorCapsFrequency,
			)
			if err != nil {
				log.Errorf(
					"periodic grant orchestrator caps to alloc %s: %w",
					alloc.String(), err)
			}
			return
		}
	}()
	return nil
}

func (o *BasicOrchestrator) Done() <-chan struct{} {
	return o.ctx.Done()
}

func (o *BasicOrchestrator) sendReply(msg actor.Envelope, payload interface{}) {
	var opt []actor.MessageOption
	if msg.IsBroadcast() {
		opt = append(opt, actor.WithMessageSource(o.actor.Handle()))
	}

	reply, err := actor.ReplyTo(msg, payload, opt...)
	if err != nil {
		log.Debugf("creating reply: %s", err)
		return
	}

	if err := o.actor.Send(reply); err != nil {
		log.Debugf("sending reply: %s", err)
	}
}

func (o *BasicOrchestrator) Actor() actor.Actor {
	return o.actor
}
