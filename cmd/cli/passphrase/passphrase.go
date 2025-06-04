package passphrase

import (
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/lib/env"
	"gitlab.com/nunet/device-management-service/utils"
)

type Provider interface {
	GetPassphrase(key string) (string, error)
	NewPassphrase(key string) (string, error)
}

type envPassphraseProvider struct {
	env env.EnvironmentProvider
}

func (e *envPassphraseProvider) GetPassphrase(_ string) (string, error) {
	passphrase := e.env.Getenv(node.DMSPassphraseEnv)
	if passphrase == "" {
		return "", ErrPassphraseNotFound
	}
	return passphrase, nil
}

func (e *envPassphraseProvider) NewPassphrase(key string) (string, error) {
	return e.GetPassphrase(key)
}

type promptPassphraseProvider struct{}

func (p *promptPassphraseProvider) GetPassphrase(_ string) (string, error) {
	return utils.PromptForPassphrase(false)
}

func (p *promptPassphraseProvider) NewPassphrase(_ string) (string, error) {
	return utils.PromptForPassphrase(true)
}

type DefaultPassphraseProvider struct {
	providers []Provider
}

func (d *DefaultPassphraseProvider) GetPassphrase(key string) (string, error) {
	for _, provider := range d.providers {
		passphrase, err := provider.GetPassphrase(key)
		if err == nil {
			return passphrase, nil
		}
	}
	return "", ErrPassphraseNotFound
}

func (d *DefaultPassphraseProvider) NewPassphrase(key string) (string, error) {
	for _, provider := range d.providers {
		passphrase, err := provider.NewPassphrase(key)
		if err == nil {
			return passphrase, nil
		}
	}
	return "", ErrNewPassphraseFailed
}

func DefaultProvider(env env.EnvironmentProvider) Provider {
	return &DefaultPassphraseProvider{
		providers: []Provider{
			&envPassphraseProvider{env: env},
			&promptPassphraseProvider{},
		},
	}
}
