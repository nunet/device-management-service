package utils

import (
	"context"
	"fmt"
)

type (
	nodesCtxKey    struct{}
	nodeMapCtxKey  struct{}
	ensembleCtxKey struct{}
)

type TestCtx struct {
	ctx context.Context
}

func NewTestCtx(ctx context.Context) *TestCtx {
	return &TestCtx{ctx: ctx}
}

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
