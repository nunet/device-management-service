// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"context"
	"fmt"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

// TODO: Deprecate keys in favor of struct fields
// These are the keys stored into the Context
// We could use a map directly, but empty struct
// has a better performance if it grows too large
type (
	nodesCtxKey          struct{}
	nodeMapCtxKey        struct{}
	ensembleIDCtxKey     struct{}
	ensembleFileCtxKey   struct{}
	manifestCtxKey       struct{}
	allocRespCtxKey      struct{}
	contractInfoKey      struct{}
	deploymentsCtxKey    struct{}
	connectionAttemptKey struct{}
	natRoutersKey        struct{}
	relayAddressKey      struct{}
	tokenMapKey          struct{}
	orgMapCtxKey         struct{}
)

// TODO: Define TestCase struct
// TestCtx is a wrapper of Context
// It allows for some type safety and it's more elegant
type TestCtx struct {
	ctx context.Context
}

// TODO: Temporary wrapper for contract
type ContractData struct {
	HostDID string
	DID     string
}

func NewTestCtx(ctx context.Context) *TestCtx {
	return &TestCtx{ctx: ctx}
}

// Call this method if needed to return a Context value
func (t *TestCtx) Unwrap() context.Context {
	return t.ctx
}

func (t *TestCtx) Nodes() (map[string]*Node, error) {
	nodes, ok := t.ctx.Value(nodesCtxKey{}).(map[string]*Node)
	if !ok {
		return nil, fmt.Errorf("no nodes available on context")
	}
	return nodes, nil
}

func (t *TestCtx) WithNodes(n map[string]*Node) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, nodesCtxKey{}, n),
	}
}

// TODO: Deprecate in favor of `Nodes`
func (t *TestCtx) NodeMap() (map[string]*Instance, error) {
	nodeMap, ok := t.ctx.Value(nodeMapCtxKey{}).(map[string]*Instance)
	if !ok {
		return nil, fmt.Errorf("no node map available on context")
	}
	return nodeMap, nil
}

// TODO: Deprecate in favor of `WithNodes`
func (t *TestCtx) WithNodeMap(m map[string]*Instance) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, nodeMapCtxKey{}, m),
	}
}

func (t *TestCtx) EnsembleID() (string, error) {
	id, ok := t.ctx.Value(ensembleIDCtxKey{}).(string)
	if !ok {
		return "", fmt.Errorf("no ensemble ID available on context")
	}
	return id, nil
}

func (t *TestCtx) WithEnsembleID(id string) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, ensembleIDCtxKey{}, id),
	}
}

func (t *TestCtx) EnsembleFile() (string, error) {
	path, ok := t.ctx.Value(ensembleFileCtxKey{}).(string)
	if !ok {
		return "", fmt.Errorf("no ensemble file available on context")
	}
	return path, nil
}

func (t *TestCtx) WithEnsembleFile(path string) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, ensembleFileCtxKey{}, path),
	}
}

func (t *TestCtx) Manifest() (*jobtypes.EnsembleManifest, error) {
	manifest, ok := t.ctx.Value(manifestCtxKey{}).(*jobtypes.EnsembleManifest)
	if !ok {
		return nil, fmt.Errorf("no manifest available on context")
	}
	return manifest, nil
}

func (t *TestCtx) WithManifest(m *jobtypes.EnsembleManifest) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, manifestCtxKey{}, m),
	}
}

func (t *TestCtx) AllocationResponses() ([]string, error) {
	cfg, ok := t.ctx.Value(allocRespCtxKey{}).([]string)
	if !ok {
		return []string{}, fmt.Errorf("no allocation response available on context")
	}
	return cfg, nil
}

func (t *TestCtx) WithAllocationResponses(r []string) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, allocRespCtxKey{}, r),
	}
}

func (t *TestCtx) Contract() (ContractData, error) {
	data, ok := t.ctx.Value(contractInfoKey{}).(ContractData)
	if !ok {
		return ContractData{}, fmt.Errorf("no contract DID available on context")
	}
	return data, nil
}

func (t *TestCtx) WithContract(c ContractData) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, contractInfoKey{}, c),
	}
}

func (t *TestCtx) Deployments() (map[string]string, error) {
	cfg, ok := t.ctx.Value(deploymentsCtxKey{}).(map[string]string)
	if !ok {
		return nil, fmt.Errorf("no allocation response available on context")
	}
	return cfg, nil
}

func (t *TestCtx) WithDeployments(d map[string]string) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, deploymentsCtxKey{}, d),
	}
}

func (t *TestCtx) ConnectionAttempt() (map[string]interface{}, error) {
	attempt, ok := t.ctx.Value(connectionAttemptKey{}).(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("no connection attempt available on context")
	}
	return attempt, nil
}

func (t *TestCtx) WithConnectionAttempt(attempt map[string]interface{}) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, connectionAttemptKey{}, attempt),
	}
}

func (t *TestCtx) NATRouters() ([]*NATRouterContainer, error) {
	routers, ok := t.ctx.Value(natRoutersKey{}).([]*NATRouterContainer)
	if !ok {
		return nil, fmt.Errorf("no NAT routers available on context")
	}
	return routers, nil
}

func (t *TestCtx) WithNATRouters(routers []*NATRouterContainer) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, natRoutersKey{}, routers),
	}
}

func (t *TestCtx) RelayAddress() (string, error) {
	addr, ok := t.ctx.Value(relayAddressKey{}).(string)
	if !ok {
		return "", fmt.Errorf("no relay address available on context")
	}
	return addr, nil
}

func (t *TestCtx) WithRelayAddress(addr string) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, relayAddressKey{}, addr),
	}
}

func (t *TestCtx) TokenMap() (map[string]string, error) {
	tokenMap, ok := t.ctx.Value(tokenMapKey{}).(map[string]string)
	if !ok {
		return nil, fmt.Errorf("no token map available on context")
	}
	return tokenMap, nil
}

func (t *TestCtx) WithTokenMap(m map[string]string) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, tokenMapKey{}, m),
	}
}

func (t *TestCtx) OrganizationMap() (map[string]*Context, error) {
	orgMap, ok := t.ctx.Value(orgMapCtxKey{}).(map[string]*Context)
	if !ok {
		return nil, fmt.Errorf("no organization map available on context")
	}
	return orgMap, nil
}

func (t *TestCtx) WithOrganizationMap(m map[string]*Context) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, orgMapCtxKey{}, m),
	}
}
