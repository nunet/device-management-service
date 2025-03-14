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
	"gitlab.com/nunet/device-management-service/dms/node/geolocation"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

const (
	BidRequestTimeout           = 5 * time.Second
	VerifyEdgeConstraintTimeout = 5 * time.Second
	CommitDeploymentTimeout     = 3 * time.Second
	AllocationDeploymentTimeout = 5 * time.Second

	// Setting a big timeout as the user might have to
	// download large execution images
	AllocationStartTimeout    = 5 * time.Minute
	AllocationShutdownTimeout = 5 * time.Second

	MinEnsembleDeploymentTime = 15 * time.Second

	SubnetCreateTimeout  = 2 * time.Minute
	SubnetDestroyTimeout = 30 * time.Second

	MaxBidMultiplier = 8
	MaxPermutations  = 1_000_000

	grantOrchestratorCapsFrequency = 5 * time.Minute

	orchSubnetName = "orchestrator"
)

var (
	ErrProvisioningFailed   = errors.New("failed to provision the ensemble")
	ErrDeploymentFailed     = errors.New("failed to create deployment")
	ErrOrchestratorExists   = errors.New("orchestrator with ID already exists")
	ErrOrchestratorNotFound = errors.New("orchestrator with ID not found")
)

// Orchestrator manages the lifecycle of an ensemble deployment
type Orchestrator interface {
	Deploy(expiry time.Time) error
	Shutdown()
	Stop()
	Status() jtypes.DeploymentStatus
	Manifest() jtypes.EnsembleManifest
	Config() jtypes.EnsembleConfig
	ID() string
	ActorPrivateKey() crypto.PrivKey
	DeploymentSnapshot() jtypes.DeploymentSnapshot
	GetAllocationLogs(name string) (AllocationLogsResponse, error)
	WriteAllocationLogs(name string, stdout, stderr []byte) (string, error)
}

// TODO: use immutable data structures (there are libraries for that), specially
// for EnsembleManifest and EnsembleConfig
type BasicOrchestrator struct {
	lock   sync.Mutex
	ctx    context.Context
	cancel func()

	fs      afero.Afero
	workDir string
	actor   actor.Actor
	geo     *geolocation.GeoLocator

	id             string
	cfg            jtypes.EnsembleConfig
	manifest       jtypes.EnsembleManifest
	subnetManifest SubnetManifest
	status         jtypes.DeploymentStatus

	deploymentSnapshot jtypes.DeploymentSnapshot
	supervisor         *Supervisor

	nonce uint64
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

	geo, err := geolocation.NewGeoLocator()
	if err != nil {
		return nil, fmt.Errorf("failed to create geolocator: %w", err)
	}

	subnet, err := newSubnetManifest()
	if err != nil {
		return nil, fmt.Errorf("failed to create subnet manifest: %w", err)
	}

	o := &BasicOrchestrator{
		actor:          oActor,
		geo:            geo,
		id:             id,
		cfg:            cfg,
		ctx:            ctx,
		fs:             fs,
		workDir:        workDir,
		subnetManifest: subnet,
		supervisor:     NewSupervisor(ctx, oActor, id),
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
		"status", jtypes.DeploymentStatusString(status),
		"orchestratorID", o.id)
	o.status = status
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
	go o.supervisor.Supervise(o.manifest)
	return nil
}

func (o *BasicOrchestrator) newManifest(
	cfg jtypes.EnsembleConfig,
) jtypes.EnsembleManifest {
	manifest := jtypes.EnsembleManifest{
		ID:           o.id,
		Orchestrator: o.actor.Handle(),
		Allocations:  make(map[string]jtypes.AllocationManifest),
		Nodes:        make(map[string]jtypes.NodeManifest),
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
// TODO: provision/commit should not update o.manifest by themselves
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
		candidateDeployment, err := o.bid(cfg, expiry)
		if err != nil {
			if errors.Is(err, ErrCandidateNotFound) {
				log.Warnf("candidate deployment not found, redeploying: %v", err)
				continue deploy
			}

			log.Errorf("failed to bid: %v", err)
		}

		// 2. Commit the deployment
		o.deploymentSnapshot.Candidates = candidateDeployment
		manifest, err := o.commit(cfg, partialManifest, candidateDeployment)
		if err != nil {
			log.Warnw("failed to commit deployment",
				"labels", []string{string(observability.LabelDeployment)},
				"orchestratorID", o.id,
				"error", err)
			continue deploy
		}

		// 3. provision the network and start the allocations
		if err := o.provision(cfg, manifest); err != nil {
			log.Errorw("provisioning failed",
				"labels", []string{string(observability.LabelDeployment)},
				"error", err,
				"orchestratorID", o.id)

			o.lock.Lock()
			o.revert(cfg, manifest)
			o.lock.Unlock()
			continue deploy
		}

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
		fmt.Sprintf(behaviors.AllocationLogsBehavior, o.manifest.ID),
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
