package provider

import "fmt"

// Factory is a constructor that builds a Provider from config.
type Factory func(cfg map[string]interface{}) (Provider, error)

// FactoryRegistry stores known provider constructors
type FactoryRegistry struct {
	GatewayDID string
	factories  map[string]Factory
}

// NewProviderFactoryRegistry creates an empty registry.
func NewProviderFactoryRegistry(gatewayDID string) *FactoryRegistry {
	return &FactoryRegistry{
		GatewayDID: gatewayDID,
		factories:  make(map[string]Factory),
	}
}

// Register adds a new provider factory.
func (r *FactoryRegistry) Register(name string, factory Factory) {
	r.factories[name] = factory
}

// Create instantiates a provider using a registered factory.
func (r *FactoryRegistry) Create(name string, cfg map[string]interface{}) (Provider, error) {
	f, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("no factory registered for provider type %q", name)
	}
	return f(cfg)
}

// List lists all known provider factory types.
func (r *FactoryRegistry) List() []string {
	names := make([]string, 0, len(r.factories))
	for n := range r.factories {
		names = append(names, n)
	}
	return names
}
