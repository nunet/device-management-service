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
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/crypto"
	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/observability"
	"gitlab.com/nunet/device-management-service/types"
)

// Registry is an interface which acts as a source for orchestrators
type Registry interface {
	// NewOrchestrator creates a new orchestrator
	NewOrchestrator(
		ctx context.Context, fs afero.Afero, workDir string,
		id string, actor actor.Actor, cfg jtypes.EnsembleConfig,
		nodeIDGenerator types.NodeIDGenerator, allocationIDGenerator types.AllocationIDGenerator,
	) (Orchestrator, error)
	// RestoreDeployment restores deployments where the status is either provisioning, committing or running
	RestoreDeployment(
		ctx context.Context,
		actr actor.Actor, id string, cfg jtypes.EnsembleConfig,
		manifest jtypes.EnsembleManifest, status jtypes.DeploymentStatus,
		restoreInfo jtypes.DeploymentSnapshot,
		subnetManifest jtypes.SubnetManifest,
		allocationIDGenerator types.AllocationIDGenerator,
	) (Orchestrator, error)
	// Orchestrators returns a map of all orchestrators
	Orchestrators() map[string]Orchestrator
	// GetOrchestrator returns an orchestrator by ID
	GetOrchestrator(id string) (Orchestrator, error)
	// DeleteOrchestrator deletes an orchestrator by ID
	DeleteOrchestrator(id string)

	// Methods for deployment persistence
	// SaveOrchestrator persists orchestrator state to store
	SaveOrchestrator(orchestrator Orchestrator) error
	// GetAllDeployments retrieves all deployments from store
	GetAllDeployments() ([]*jtypes.OrchestratorView, error)
	// GetDeploymentsByStatus retrieves deployments filtered by status
	GetDeploymentsByStatus(status jtypes.DeploymentStatus) ([]*jtypes.OrchestratorView, error)
	// DeleteDeployment removes a specific deployment by orchestrator ID
	DeleteDeployment(orchestratorID string) error
	// GetDeployment retrieves a deployment from store by ID
	GetDeployment(orchestratorID string) (*jtypes.OrchestratorView, error)
}

// basicRegistry the default implementation of Registry
type basicRegistry struct {
	lock          sync.RWMutex
	orchestrators map[string]Orchestrator // map of orchestrators (in-memory cache)
	store         DeploymentStore         // persistent store
}

var _ Registry = (*basicRegistry)(nil)

// NewRegistry creates a new orchestrator registry
func NewRegistry(store DeploymentStore) Registry {
	return &basicRegistry{
		orchestrators: make(map[string]Orchestrator),
		store:         store,
	}
}

// NewOrchestrator creates a new orchestrator
// TODO-style: NewOrchestrator calls NewOrchestrator, that is confusing
func (f *basicRegistry) NewOrchestrator(
	ctx context.Context, fs afero.Afero, workDir string,
	id string, actor actor.Actor, cfg jtypes.EnsembleConfig,
	nodeIDGenerator types.NodeIDGenerator, allocationIDGenerator types.AllocationIDGenerator,
) (Orchestrator, error) {
	// check if orchestrator already exists in store
	if _, err := f.store.Get(id); err == nil {
		return nil, ErrOrchestratorExists
	}

	// NewOrchestrator creates a new orchestrator with a new context
	o, err := NewOrchestrator(ctx, fs, workDir, id, actor, cfg, nodeIDGenerator, allocationIDGenerator)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
	}

	// Initialize manifest with metadata so it's available when saved to store
	o.manifest = o.newManifest(cfg)

	// Save to store immediately
	if err := f.SaveOrchestrator(o); err != nil {
		return nil, fmt.Errorf("failed to save orchestrator: %w", err)
	}

	f.lock.Lock()
	defer f.lock.Unlock()
	f.orchestrators[id] = o

	// save deployment on status change, use the orchestrator's context
	f.saveDeploymentOnStatusChange(o.ctx, o)

	return o, nil
}

func (f *basicRegistry) saveDeploymentOnStatusChange(ctx context.Context, o Orchestrator) {
	go func() {
		prevStatus := o.Status()
		statusCh := o.StatusChannel(ctx)
		for {
			select {
			case <-ctx.Done():
				return
			case status := <-statusCh:
				if status == prevStatus {
					continue
				}
				prevStatus = status
				log.Debugw("status changed, saving deployment...",
					"labels", []string{string(observability.LabelDeployment)},
					"orchestratorID", o.ID(),
					"status", o.Status().String(),
				)
				if err := f.SaveOrchestrator(o); err != nil {
					log.Errorw("failed to save orchestrator on status change", "error", err)
				}
			}
		}
	}()
}

// restoreDeployment restores deployments where the status is either provisioning, committing or running
// TODO: restore subnetManifest if necessary
func (f *basicRegistry) restoreDeployment(
	ctx context.Context,
	actr actor.Actor, id string,
	cfg jtypes.EnsembleConfig, manifest jtypes.EnsembleManifest,
	status jtypes.DeploymentStatus, restoreInfo jtypes.DeploymentSnapshot,
	subnetManifest jtypes.SubnetManifest,
	allocationIDGenerator types.AllocationIDGenerator,
) (Orchestrator, error) {
	log.Infow("restoring deployment", "id", id, "status", status)
	ctx, cancel := context.WithCancel(ctx)

	o := &BasicOrchestrator{
		id:                    id,
		actor:                 actr,
		cfg:                   cfg,
		status:                status,
		deploymentSnapshot:    restoreInfo,
		supervisor:            NewSupervisor(ctx, actr, id),
		manifest:              manifest,
		subnetManifest:        subnetManifest,
		ctx:                   ctx,
		cancel:                cancel,
		statusSubscribers:     make(map[chan jtypes.DeploymentStatus]struct{}),
		allocationIDGenerator: allocationIDGenerator,
	}

	// TODO: manifest.Empty()
	if manifest.ID == "" {
		o.manifest = o.NewManifest(cfg)
	}

	log.Infow("restored deployment", "id", id, "status", status)

	if o.status == jtypes.DeploymentStatusCommitting {
		log.Debugw("reverting deployment of old candidates and restarting deployment from the beginning",
			"labels", []string{string(observability.LabelDeployment)},
			"reason", "deployment was in committing state",
			"orchestratorID", id,
		)
		manifest := o.manifest.Clone()
		for nodeID, bid := range restoreInfo.Candidates {
			o.revertNodeDeployment(cfg, nodeID, bid.Handle())
		}

		// save deployment on status change, use the orchestrator's context
		f.saveDeploymentOnStatusChange(o.ctx, o)

		go o.monitorOnlyTaskManifest()
		go o.supervisor.Supervise(jtypes.NewManifestReader(o.manifest))

		return o, o.deploy(cfg, manifest, restoreInfo.Expiry)
	}

	if o.status == jtypes.DeploymentStatusProvisioning {
		log.Debugw("restoring deployment from manifest",
			"labels", []string{string(observability.LabelDeployment)},
			"orchestratorID", id,
		)

		// Log the manifest content to see what nodes are included
		log.Infow("manifest before provisioning restoration",
			"labels", []string{string(observability.LabelDeployment)},
			"orchestratorID", id,
			"manifestNodes", len(manifest.Nodes),
			"nodeIDs", func() []string {
				var nodeIDs []string
				for nodeID := range manifest.Nodes {
					nodeIDs = append(nodeIDs, nodeID)
				}
				return nodeIDs
			}(),
		)

		log.Infow("starting provisioning restoration with healthy provisioner",
			"labels", []string{string(observability.LabelDeployment)},
			"orchestratorID", id,
		)

		provisioner := NewProvisioner(ctx, cancel, actr, subnetManifest, o.allocationIDGenerator)
		manifestAfterProvision, err := provisioner.Provision(
			jtypes.NewEnsembleCfgReader(cfg),
			jtypes.NewManifestReader(manifest))
		if err != nil {
			log.Errorf("failed to provision network during restoration: %s", err)
			log.Infow("reverting deployment due to failed provisioning during restoration",
				"labels", []string{string(observability.LabelDeployment)},
				"orchestratorID", id,
			)
			o.revert(cfg, manifest)
			log.Infow("reverted deployment due to failed provisioning during restoration",
				"labels", []string{string(observability.LabelDeployment)},
				"orchestratorID", id,
			)

			log.Infow("deploying new manifest after failed provisioning during restoration",
				"labels", []string{string(observability.LabelDeployment)},
				"orchestratorID", id,
			)

			// save deployment on status change, use the orchestrator's context
			f.saveDeploymentOnStatusChange(o.ctx, o)

			go o.monitorOnlyTaskManifest()
			go o.supervisor.Supervise(jtypes.NewManifestReader(o.manifest))

			return o, o.deploy(cfg, o.newManifest(cfg), restoreInfo.Expiry)
		}

		log.Infow("provisioning restoration completed successfully",
			"labels", []string{string(observability.LabelDeployment)},
			"orchestratorID", id,
		)

		o.updateManifest(manifestAfterProvision)

		o.setStatus(jtypes.DeploymentStatusRunning)
	}

	if o.status == jtypes.DeploymentStatusRunning {
		if o.manifest.Subnet.Join {
			if _, ok := o.subnetManifest.IndexRoutingTable[orchSubnetName]; ok {
				handleError := func(err error) (Orchestrator, error) {
					log.Errorf("failed to join subnet: %s", err)
					o.revert(cfg, manifest)
					return o, o.deploy(cfg, o.newManifest(cfg), restoreInfo.Expiry)
				}

				provisioner := NewProvisioner(ctx, cancel, o.actor, o.subnetManifest, o.allocationIDGenerator)
				err := provisioner.createSubnet(o.manifest.ID, o.subnetManifest.RoutingTable, []actor.Handle{o.actor.Supervisor()})
				if err != nil {
					return handleError(err)
				}

				err = provisioner.orchestratorJoinSubnet(
					manifest.ID,
					o.subnetManifest.IndexRoutingTable,
					o.subnetManifest.RoutingTable,
					o.subnetManifest.DNSRecords,
				)
				if err != nil {
					return handleError(err)
				}
			}
		}
	}

	// save deployment on status change, use the orchestrator's context
	f.saveDeploymentOnStatusChange(o.ctx, o)

	go o.monitorOnlyTaskManifest()
	go o.supervisor.Supervise(jtypes.NewManifestReader(o.manifest))

	return o, nil
}

// RestoreDeployment creates an orchestrator and attempts to restore its deployment
func (f *basicRegistry) RestoreDeployment(
	ctx context.Context,
	actr actor.Actor, id string, cfg jtypes.EnsembleConfig,
	manifest jtypes.EnsembleManifest, status jtypes.DeploymentStatus, restoreInfo jtypes.DeploymentSnapshot,
	subnetManifest jtypes.SubnetManifest,
	allocationIDGenerator types.AllocationIDGenerator,
) (Orchestrator, error) {
	// check if orchestrator already exists
	f.lock.RLock()
	if _, ok := f.orchestrators[id]; ok {
		f.lock.RUnlock()
		return nil, ErrOrchestratorExists
	}
	f.lock.RUnlock()

	o, err := f.restoreDeployment(ctx, actr, id, cfg, manifest, status, restoreInfo, subnetManifest, allocationIDGenerator)
	if err != nil {
		return nil, fmt.Errorf("failed to restore deployment: %w", err)
	}

	f.lock.Lock()
	defer f.lock.Unlock()
	f.orchestrators[id] = o

	return o, nil
}

// Orchestrators returns a map of all orchestrators
func (f *basicRegistry) Orchestrators() map[string]Orchestrator {
	f.lock.RLock()
	defer f.lock.RUnlock()

	orchestrators := make(map[string]Orchestrator, len(f.orchestrators))
	for id, o := range f.orchestrators {
		orchestrators[id] = o
	}

	return orchestrators
}

// GetOrchestrator returns an orchestrator by ID
func (f *basicRegistry) GetOrchestrator(id string) (Orchestrator, error) {
	f.lock.RLock()
	defer f.lock.RUnlock()

	o, ok := f.orchestrators[id]
	if !ok {
		return nil, ErrOrchestratorNotFound
	}

	return o, nil
}

// DeleteOrchestrator deletes an orchestrator by ID
func (f *basicRegistry) DeleteOrchestrator(id string) {
	f.lock.Lock()
	defer f.lock.Unlock()

	// Remove from memory
	delete(f.orchestrators, id)

	// Remove from store
	if err := f.store.Delete(id); err != nil {
		log.Warnf("failed to delete orchestrator from store: %s", err)
		// Log warning but don't fail - the orchestrator is already removed from memory
		// This could happen if the deployment was already deleted or doesn't exist
	}

	log.Infow("deleted orchestrator from store", "id", id)
}

// SaveOrchestrator persists orchestrator state to store
func (f *basicRegistry) SaveOrchestrator(orchestrator Orchestrator) error {
	pvkey := orchestrator.ActorPrivateKey()
	pkRaw, err := crypto.MarshalPrivateKey(pvkey)
	if err != nil {
		return fmt.Errorf("convert priv key to raw: %w", err)
	}
	view := &jtypes.OrchestratorView{
		BaseDBModel: types.BaseDBModel{
			ID:        orchestrator.ID(),
			CreatedAt: time.Now(),
		},
		OrchestratorID:     orchestrator.ID(),
		Cfg:                orchestrator.Config(),
		Manifest:           orchestrator.Manifest(),
		SubnetManifest:     orchestrator.SubnetManifest(),
		Status:             orchestrator.Status(),
		DeploymentSnapshot: orchestrator.DeploymentSnapshot(),
		PrivKey:            pkRaw,
	}
	return f.store.Upsert(view)
}

// GetAllDeployments retrieves all deployments from store
func (f *basicRegistry) GetAllDeployments() ([]*jtypes.OrchestratorView, error) {
	deployments, err := f.store.GetAll(nil)
	return deployments, err
}

// GetDeploymentsByStatus retrieves deployments filtered by status
func (f *basicRegistry) GetDeploymentsByStatus(status jtypes.DeploymentStatus) ([]*jtypes.OrchestratorView, error) {
	return f.store.GetAll(&status)
}

// DeleteDeployment removes a specific deployment by orchestrator ID
func (f *basicRegistry) DeleteDeployment(orchestratorID string) error {
	return f.store.Delete(orchestratorID)
}

// GetDeployment retrieves a deployment from store by ID
func (f *basicRegistry) GetDeployment(orchestratorID string) (*jtypes.OrchestratorView, error) {
	return f.store.Get(orchestratorID)
}
