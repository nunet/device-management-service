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

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/tokenomics/eventhandler"
	"gitlab.com/nunet/device-management-service/types"
)

type MockOrchestrator struct {
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

	deploymentSnapshot jtypes.DeploymentSnapshot
	supervisor         *Supervisor

	nonce uint64
}

func NewMockOrchestrator(
	ctx context.Context,
	fs afero.Afero,
	workDir string,
	id string,
	oActor actor.Actor,
	cfg jtypes.EnsembleConfig,
) (*MockOrchestrator, error) {
	mo := &MockOrchestrator{
		actor:              oActor,
		id:                 id,
		cfg:                cfg,
		ctx:                ctx,
		fs:                 fs,
		workDir:            workDir,
		subnetManifest:     jtypes.SubnetManifest{},
		deploymentSnapshot: jtypes.DeploymentSnapshot{},
		nonce:              0,
		supervisor:         NewSupervisor(ctx, oActor, id),
	}
	mo.ctx, mo.cancel = context.WithCancel(ctx)

	return mo, nil
}

func (m *MockOrchestrator) Deploy(_ time.Time) error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.manifest = m.newManifest(m.cfg)
	return nil
}

func (m *MockOrchestrator) newManifest(
	cfg jtypes.EnsembleConfig,
) jtypes.EnsembleManifest {
	manifest := jtypes.EnsembleManifest{
		ID:           m.id,
		Orchestrator: m.actor.Handle(),
		Metadata:     cfg.V1.Metadata,
		Allocations:  make(map[string]jtypes.AllocationManifest),
		Nodes:        make(map[string]jtypes.NodeManifest),
	}

	for name, alloc := range cfg.Allocations() {
		amf := jtypes.AllocationManifest{
			ID:          types.NewAllocationID(m.id, "mock-node", name).String(),
			Type:        alloc.Type,
			NodeID:      "mock-node",
			DNSName:     alloc.DNSName + ".internal",
			Healthcheck: alloc.HealthCheck,
			Status:      jtypes.AllocationPending,
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

func (m *MockOrchestrator) Shutdown() error {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.status = jtypes.DeploymentStatusCompleted
	return nil
}

func (m *MockOrchestrator) Stop() {
	m.lock.Lock()
	defer m.lock.Unlock()
	m.status = jtypes.DeploymentStatusCompleted
}

// helper to set status when testing
func (m *MockOrchestrator) SetStatus(status jtypes.DeploymentStatus) {
	m.lock.Lock()
	defer m.lock.Unlock()

	m.status = status
}

func (m *MockOrchestrator) Status() jtypes.DeploymentStatus {
	m.lock.Lock()
	defer m.lock.Unlock()
	return m.status
}

func (m *MockOrchestrator) Manifest() jtypes.EnsembleManifest {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.manifest.Clone()
}

func (m *MockOrchestrator) SubnetManifest() jtypes.SubnetManifest {
	m.lock.Lock()
	defer m.lock.Unlock()

	return m.subnetManifest
}

func (m *MockOrchestrator) Config() jtypes.EnsembleConfig {
	return jtypes.EnsembleConfig{}
}

func (m *MockOrchestrator) ID() string {
	return m.id
}

func (m *MockOrchestrator) ActorPrivateKey() crypto.PrivKey {
	return m.actor.Security().PrivKey()
}

func (m *MockOrchestrator) DeploymentSnapshot() jtypes.DeploymentSnapshot {
	return jtypes.DeploymentSnapshot{}
}

func (m *MockOrchestrator) GetAllocationLogs(_ string) (AllocationLogsResponse, error) {
	return AllocationLogsResponse{}, nil
}

func (m *MockOrchestrator) WriteAllocationLogs(_ string, _, _ []byte) (string, error) {
	return "", nil
}

func (m *MockOrchestrator) Update(_ jtypes.EnsembleConfig, _ time.Time) error {
	return nil
}

func (m *MockOrchestrator) StatusChannel(_ context.Context) <-chan jtypes.DeploymentStatus {
	return make(chan jtypes.DeploymentStatus)
}

func (m *MockOrchestrator) AllocationInfo() map[string]jtypes.AllocationInfo {
	return make(map[string]jtypes.AllocationInfo)
}

func (m *MockOrchestrator) Done() <-chan struct{} {
	return nil
}

type MockOrchestratorRegistry struct {
	lock          sync.RWMutex
	orchestrators map[string]Orchestrator // map of orchestrators
}

var _ Registry = (*MockOrchestratorRegistry)(nil)

// NewRegistry creates a new orchestrator registry
func NewMockOrchestratorRegistry() *MockOrchestratorRegistry {
	return &MockOrchestratorRegistry{
		orchestrators: make(map[string]Orchestrator),
	}
}

func (m *MockOrchestratorRegistry) NewOrchestrator(
	ctx context.Context, fs afero.Afero, workDir string,
	id string, actor actor.Actor, cfg jtypes.EnsembleConfig,
	_ types.NodeIDGenerator, _ types.AllocationIDGenerator,
	_ *eventhandler.EventHandler, _ map[string]types.ContractConfig,
) (Orchestrator, error) {
	m.lock.RLock()
	if _, ok := m.orchestrators[id]; ok {
		m.lock.RUnlock()
		return nil, ErrOrchestratorExists
	}
	m.lock.RUnlock()

	mo, err := NewMockOrchestrator(ctx, fs, workDir, id, actor, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create orchestrator: %w", err)
	}

	m.lock.Lock()
	defer m.lock.Unlock()
	m.orchestrators[id] = mo
	return mo, nil
}

func (m *MockOrchestratorRegistry) RestoreDeployment(
	_ context.Context, _ afero.Afero,
	_ actor.Actor, _ string, _ jtypes.EnsembleConfig,
	_ jtypes.EnsembleManifest, _ jtypes.DeploymentStatus,
	_ jtypes.DeploymentSnapshot,
	_ jtypes.SubnetManifest,
	_ types.AllocationIDGenerator,
) (Orchestrator, error) {
	return nil, nil
}

func (m *MockOrchestratorRegistry) Orchestrators() map[string]Orchestrator {
	m.lock.RLock()
	defer m.lock.RUnlock()

	orchestrators := make(map[string]Orchestrator, len(m.orchestrators))
	for id, o := range m.orchestrators {
		orchestrators[id] = o
	}

	return orchestrators
}

func (m *MockOrchestratorRegistry) GetOrchestrator(id string) (Orchestrator, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	if o, ok := m.orchestrators[id]; ok {
		return o, nil
	}
	return nil, ErrOrchestratorNotFound
}

func (m *MockOrchestratorRegistry) DeleteOrchestrator(_ string) {}

// Methods for deployment persistence (mock implementations)
func (m *MockOrchestratorRegistry) SaveOrchestrator(_ Orchestrator) error {
	return nil
}

func (m *MockOrchestratorRegistry) GetAllDeployments() ([]*jtypes.OrchestratorView, error) {
	return nil, nil
}

func (m *MockOrchestratorRegistry) GetDeploymentsByStatus(_ jtypes.DeploymentStatus) ([]*jtypes.OrchestratorView, error) {
	return nil, nil
}

func (m *MockOrchestratorRegistry) QueryDeployments(_ DeploymentQuery) ([]*jtypes.OrchestratorView, int, error) {
	return nil, 0, nil
}

func (m *MockOrchestratorRegistry) DeleteDeployment(_ string) error {
	return nil
}

func (m *MockOrchestratorRegistry) GetDeployment(orchestratorID string) (*jtypes.OrchestratorView, error) {
	m.lock.RLock()
	defer m.lock.RUnlock()

	orch, ok := m.orchestrators[orchestratorID]
	if !ok {
		return nil, ErrOrchestratorNotFound
	}

	// Convert orchestrator to OrchestratorView
	privKey := orch.ActorPrivateKey()
	privKeyBytes, err := crypto.PrivateKeyToBytes(privKey)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal private key: %w", err)
	}

	view := &jtypes.OrchestratorView{
		OrchestratorID:     orch.ID(),
		Cfg:                orch.Config(),
		Manifest:           orch.Manifest(),
		SubnetManifest:     orch.SubnetManifest(),
		Status:             orch.Status(),
		DeploymentSnapshot: orch.DeploymentSnapshot(),
		PrivKey:            privKeyBytes,
	}

	return view, nil
}
