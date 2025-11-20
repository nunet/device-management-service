package provider

import (
	"fmt"
	"sync"
)

// Registry manages a collection of providers by name.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

// NewProviderRegistry
func NewProviderRegistry(initialProviders ...Provider) *Registry {
	r := &Registry{
		providers: make(map[string]Provider),
	}
	for _, p := range initialProviders {
		r.Register(p)
	}
	return r
}

// Register adds a provider to the registry.
func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.providers[p.Name()] = p
}

// Get retrieves a provider by name.
func (r *Registry) Get(name string) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("provider %q not found", name)
	}
	return p, nil
}

// List returns the names of all registered providers.
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for n := range r.providers {
		names = append(names, n)
	}
	return names
}

// All returns all registered providers.
func (r *Registry) All() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pvs := make([]Provider, 0, len(r.providers))
	for _, v := range r.providers {
		pvs = append(pvs, v)
	}
	return pvs
}
