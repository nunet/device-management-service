package utils

import (
	"context"
	"fmt"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

// These are the keys stored into the Context
// We could use a map directly, but empty struct
// has a better performance if it grows too large
type (
	nodesCtxKey     struct{}
	nodeMapCtxKey   struct{}
	ensembleCtxKey  struct{}
	manifestCtxKey  struct{}
	allocRespCtxKey struct{}
)

// TestCtx is a wrapper of Context
// It allows for some type safety and it's more elegant
type TestCtx struct {
	ctx context.Context
}

func NewTestCtx(ctx context.Context) *TestCtx {
	return &TestCtx{ctx: ctx}
}

// Call this method if needed to return a Context value
func (t *TestCtx) Unwrap() context.Context {
	return t.ctx
}

func (t *TestCtx) Nodes() ([]*Node, error) {
	nodes, ok := t.ctx.Value(nodesCtxKey{}).([]*Node)
	if !ok {
		return nil, fmt.Errorf("no nodes available on context")
	}
	return nodes, nil
}

func (t *TestCtx) WithNodes(n []*Node) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, nodesCtxKey{}, n),
	}
}

func (t *TestCtx) NodeMap() (map[string]*Node, error) {
	nodeMap, ok := t.ctx.Value(nodeMapCtxKey{}).(map[string]*Node)
	if !ok {
		return nil, fmt.Errorf("no node map available on context")
	}
	return nodeMap, nil
}

func (t *TestCtx) WithNodeMap(m map[string]*Node) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, nodeMapCtxKey{}, m),
	}
}

func (t *TestCtx) EnsembleID() (string, error) {
	id, ok := t.ctx.Value(ensembleCtxKey{}).(string)
	if !ok {
		return "", fmt.Errorf("no ensemble ID available on context")
	}
	return id, nil
}

func (t *TestCtx) WithEnsembleID(id string) *TestCtx {
	return &TestCtx{
		ctx: context.WithValue(t.ctx, ensembleCtxKey{}, id),
	}
}

func (t *TestCtx) Manifest() (*jobtypes.EnsembleManifest, error) {
	manifest, ok := t.ctx.Value(manifestCtxKey{}).(*jobtypes.EnsembleManifest)
	if !ok {
		return nil, fmt.Errorf("no ensemble ID available on context")
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
