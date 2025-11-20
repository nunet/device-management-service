package provider

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/types"
)

func TestNewProviderRegistry(t *testing.T) {
	p1 := &mockProvider{name: "provider1"}
	p2 := &mockProvider{name: "provider2"}

	registry := NewProviderRegistry(p1, p2)
	list := registry.List()

	assert.Len(t, list, 2, "should register two providers")
	assert.Contains(t, list, "provider1")
	assert.Contains(t, list, "provider2")
}

func TestRegisterAndGet(t *testing.T) {
	registry := NewProviderRegistry()
	p1 := &mockProvider{name: "provider1"}

	registry.Register(p1)
	got, err := registry.Get("provider1")

	assert.NoError(t, err)
	assert.NotNil(t, got)
	assert.Equal(t, "provider1", got.Name())
}

func TestGetNotFound(t *testing.T) {
	registry := NewProviderRegistry()
	got, err := registry.Get("nonexistent")

	assert.Error(t, err)
	assert.Nil(t, got)
	assert.Contains(t, err.Error(), "provider \"nonexistent\" not found")
}

func TestList(t *testing.T) {
	registry := NewProviderRegistry()
	p1 := &mockProvider{name: "provider1"}
	p2 := &mockProvider{name: "provider2"}
	p3 := &mockProvider{name: "provider3"}

	registry.Register(p1)
	registry.Register(p2)
	registry.Register(p3)

	list := registry.List()

	assert.Len(t, list, 3)
	assert.ElementsMatch(t, list, []string{"provider1", "provider2", "provider3"})
}

func TestRegisterOverwrites(t *testing.T) {
	registry := NewProviderRegistry()
	p1 := &mockProvider{name: "provider1"}
	p2 := &mockProvider{name: "provider1"} // same name

	registry.Register(p1)
	registry.Register(p2)

	list := registry.List()
	assert.Len(t, list, 1, "should only keep one provider after overwrite")

	got, err := registry.Get("provider1")
	assert.NoError(t, err)
	assert.Same(t, p2, got, "should overwrite with latest provider instance")
}

type mockProvider struct {
	name string
}

func (m *mockProvider) Name() string { return m.name }

func (m *mockProvider) ListPlans(_ context.Context) ([]Plan, error) { return nil, nil }

func (m *mockProvider) ProvisionServer(_ context.Context, _ Plan, _, _, _ string) (*Server, error) {
	return nil, nil
}
func (m *mockProvider) DeleteServer(_ context.Context, _ string) error { return nil }

func (m *mockProvider) RestartServer(_ context.Context, _ string) error { return nil }

func (m *mockProvider) SelectMatchingPlan(_ []Plan, _ types.Resources) (*Plan, error) {
	return nil, nil
}

func (m *mockProvider) GetServerStatus(_ context.Context, _ string) (*Server, error) {
	return nil, nil
}
