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

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/observability"
)

// Registry is an interface which acts as a source for orchestrators
type Registry interface {
	// NewOrchestrator creates a new orchestrator
	NewOrchestrator(
		ctx context.Context, fs afero.Afero, workDir string,
		id string, actor actor.Actor, cfg jtypes.EnsembleConfig,
	) (Orchestrator, error)
	// RestoreDeployment restores deployments where the status is either provisioning, committing or running
	RestoreDeployment(
		actr actor.Actor, id string, cfg jtypes.EnsembleConfig,
		manifest jtypes.EnsembleManifest, status jtypes.DeploymentStatus,
		restoreInfo jtypes.DeploymentSnapshot,
	) (Orchestrator, error)
	// Orchestrators returns a map of all orchestrators
	Orchestrators() map[string]Orchestrator
	// GetOrchestrator returns an orchestrator by ID
	GetOrchestrator(id string) (Orchestrator, error)
	// DeleteOrchestrator deletes an orchestrator by ID
	DeleteOrchestrator(id string)
}

// basicRegistry the default implementation of Registry
type basicRegistry struct {
	lock          sync.RWMutex
	orchestrators map[string]Orchestrator // map of orchestrators
}

var _ Registry = (*basicRegistry)(nil)

// NewRegistry creates a new orchestrator registry
func NewRegistry() Registry {
	return &basicRegistry{
		orchestrators: make(map[string]Orchestrator),
	}
}

// NewOrchestrator creates a new orchestrator
// TODO-style: NewOrchestrator calls NewOrchestrator, that is confusing
func (f *basicRegistry) NewOrchestrator(
	ctx context.Context, fs afero.Afero, workDir string,
	id string, actor actor.Actor, cfg jtypes.EnsembleConfig,
) (Orchestrator, error) {
	// check if orchestrator already exists
	f.lock.RLock()
	if _, ok := f.orchestrators[id]; ok {
		f.lock.RUnlock()
		return nil, ErrOrchestratorExists
	}
	f.lock.RUnlock()

	o, err := NewOrchestrator(ctx, fs, workDir, id, actor, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
	}

	f.lock.Lock()
	defer f.lock.Unlock()
	f.orchestrators[id] = o

	return o, nil
}

// restoreDeployment restores deployments where the status is either provisioning, committing or running
// TODO: restore subnetManifest if necessary
func restoreDeployment(
	actr actor.Actor, id string,
	cfg jtypes.EnsembleConfig, manifest jtypes.EnsembleManifest,
	status jtypes.DeploymentStatus, restoreInfo jtypes.DeploymentSnapshot,
) (Orchestrator, error) {
	subnet, err := newSubnetManifest()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	o := &BasicOrchestrator{
		id:                 id,
		actor:              actr,
		cfg:                cfg,
		status:             status,
		deploymentSnapshot: restoreInfo,
		supervisor:         NewSupervisor(context.TODO(), actr, id),
		subnetManifest:     subnet,
		ctx:                ctx,
		cancel:             cancel,
	}

	// TODO: manifest.Empty()
	if manifest.ID == "" {
		o.manifest = o.newManifest(cfg)
	}

	if o.status == jtypes.DeploymentStatusCommitting {
		log.Debugw("reverting deployment of old candidates and restarting deployment from the beginning",
			"labels", []string{string(observability.LabelDeployment)},
			"reason", "deployment was in committing state",
			"orchestratorID", id,
		)
		for nodeID, bid := range restoreInfo.Candidates {
			o.revertNodeDeployment(cfg, nodeID, bid.Handle())
		}

		return o, o.deploy(cfg, o.manifest, restoreInfo.Expiry)
	}

	if o.status == jtypes.DeploymentStatusProvisioning {
		log.Debugw("restoring deployment from manifest",
			"labels", []string{string(observability.LabelDeployment)},
			"orchestratorID", id,
		)
		provisioner := NewProvisioner(ctx, cancel, actr, subnet)
		manifestAfterProvision, err := provisioner.Provision(
			jtypes.NewEnsembleCfgReader(cfg),
			jtypes.NewManifestReader(manifest))
		if err != nil {
			log.Errorf("failed to provision network: %s", err)
			o.lock.Lock()
			o.revert(cfg, manifest)
			o.lock.Unlock()
			return o, o.deploy(cfg, o.newManifest(cfg), restoreInfo.Expiry)
		}

		o.updateManifest(manifestAfterProvision)

		o.setStatus(jtypes.DeploymentStatusRunning)
	}

	allocations := make(map[string]actor.Handle, len(manifest.Allocations))
	for _, allocation := range manifest.Allocations {
		allocations[allocation.ID] = allocation.Handle
	}
	go o.supervisor.Supervise(jtypes.NewManifestReader(o.manifest))

	return o, nil
}

// RestoreDeployment creates an orchestrator and attempts to restore its deployment
func (f *basicRegistry) RestoreDeployment(
	actr actor.Actor, id string, cfg jtypes.EnsembleConfig,
	manifest jtypes.EnsembleManifest, status jtypes.DeploymentStatus, restoreInfo jtypes.DeploymentSnapshot,
) (Orchestrator, error) {
	// check if orchestrator already exists
	f.lock.RLock()
	if _, ok := f.orchestrators[id]; ok {
		f.lock.RUnlock()
		return nil, ErrOrchestratorExists
	}
	f.lock.RUnlock()

	o, err := restoreDeployment(actr, id, cfg, manifest, status, restoreInfo)
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

	delete(f.orchestrators, id)
}
