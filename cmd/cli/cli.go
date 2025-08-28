// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cli

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/cli/passphrase"
	"gitlab.com/nunet/device-management-service/internal/config"
	env "gitlab.com/nunet/device-management-service/lib/env"
)

type Streams struct {
	In  io.Reader
	Out io.Writer
	Err io.Writer
}

func CmdStreams(cmd *cobra.Command) Streams {
	return Streams{
		In:  cmd.InOrStdin(),
		Out: cmd.OutOrStdout(),
		Err: cmd.ErrOrStderr(),
	}
}

type DmsCLI struct {
	env                env.EnvironmentProvider
	fs                 afero.Fs
	defaultConfig      *config.Config
	configLoader       *config.Loader
	passphraseProvider passphrase.Provider
	keystoreProvider   keystore.KeyStore
	clientFn           func(cfg *config.Config, sctx actor.SecurityContext) (client.DmsClient, error)
}

func (c *DmsCLI) Env() env.EnvironmentProvider {
	return c.env
}

func (c *DmsCLI) FS() afero.Fs {
	return c.fs
}

func (c *DmsCLI) ConfigLoader() *config.Loader {
	return c.configLoader
}

func (c *DmsCLI) Config() (*config.Config, error) {
	return c.configLoader.GetConfig()
}

func (c *DmsCLI) Passphrase(key string) (string, error) {
	return c.passphraseProvider.GetPassphrase(key)
}

func (c *DmsCLI) NewPassphrase(key string) (string, error) {
	return c.passphraseProvider.NewPassphrase(key)
}

func (c *DmsCLI) Keystore() keystore.KeyStore {
	return c.keystoreProvider
}

func (c *DmsCLI) NewClient(sctx actor.SecurityContext) (client.DmsClient, error) {
	cfg, err := c.Config()
	if err != nil {
		return nil, err
	}
	return c.clientFn(cfg, sctx)
}

func New(opts ...func(*DmsCLI)) *DmsCLI {
	cli := &DmsCLI{}

	for _, opt := range opts {
		opt(cli)
	}

	if cli.fs == nil {
		cli.fs = afero.NewOsFs()
	}

	if cli.configLoader == nil {
		cli.configLoader = config.NewLoader(config.WithFS(cli.fs))
	}

	if cli.defaultConfig != nil {
		cli.configLoader.SetConfig(*cli.defaultConfig)
	}

	if cli.env == nil {
		cli.env = env.NewOSEnvironment()
	}

	if cli.passphraseProvider == nil {
		cli.passphraseProvider = passphrase.DefaultProvider(cli.env)
	}

	if cli.clientFn == nil {
		cli.clientFn = func(cfg *config.Config, sctx actor.SecurityContext) (client.DmsClient, error) {
			return client.NewClient(client.Config{
				Host:      fmt.Sprintf("%s:%d", cfg.Rest.Addr, cfg.Rest.Port),
				APIPrefix: "/api",
				Version:   "v1",
			}, sctx)
		}
	}

	return cli
}

func WithEnv(env env.EnvironmentProvider) func(*DmsCLI) {
	return func(cli *DmsCLI) {
		cli.env = env
	}
}

func WithFS(fs afero.Fs) func(*DmsCLI) {
	return func(cli *DmsCLI) {
		cli.fs = fs
	}
}

func WithConfig(cfg *config.Config) func(*DmsCLI) {
	return func(cli *DmsCLI) {
		cli.defaultConfig = cfg
	}
}

func WithPassphraseProvider(pp passphrase.Provider) func(*DmsCLI) {
	return func(cli *DmsCLI) {
		cli.passphraseProvider = pp
	}
}

func WithKeystoreProvider(ks keystore.KeyStore) func(*DmsCLI) {
	return func(cli *DmsCLI) {
		cli.keystoreProvider = ks
	}
}

func WithClientFn(clientFn func(cfg *config.Config, sctx actor.SecurityContext) (client.DmsClient, error)) func(*DmsCLI) {
	return func(cli *DmsCLI) {
		cli.clientFn = clientFn
	}
}
