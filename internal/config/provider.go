package config

type Provider struct {
	config *Config
}

func NewProvider(opts ...func(*Provider)) *Provider {
	p := &Provider{}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *Provider) GetConfig() (*Config, error) {
	return p.config, nil
}

func WithConfig(config *Config) func(*Provider) {
	return func(p *Provider) {
		p.config = config
	}
}

func DefaultProvider() *Provider {
	return NewProvider(WithConfig(GetConfig()))
}
