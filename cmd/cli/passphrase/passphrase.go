// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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
