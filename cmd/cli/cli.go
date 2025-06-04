package cli

import (
	"fmt"
	"io"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

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
	cfgProvider        *config.Provider
	passphraseProvider passphrase.Provider
	clientFn           func(cfg *config.Config, sctx actor.SecurityContext) (client.DmsClient, error)
}

func (c *DmsCLI) Env() env.EnvironmentProvider {
	return c.env
}

func (c *DmsCLI) FS() afero.Fs {
	return c.fs
}

func (c *DmsCLI) Config() (*config.Config, error) {
	return c.cfgProvider.GetConfig()
}

func (c *DmsCLI) Passphrase(key string) (string, error) {
	return c.passphraseProvider.GetPassphrase(key)
}

func (c *DmsCLI) NewPassphrase(key string) (string, error) {
	return c.passphraseProvider.NewPassphrase(key)
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

	if cli.cfgProvider == nil {
		cli.cfgProvider = config.DefaultProvider()
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
		cli.cfgProvider = config.NewProvider(config.WithConfig(cfg))
	}
}

func WithPassphraseProvider(pp passphrase.Provider) func(*DmsCLI) {
	return func(cli *DmsCLI) {
		cli.passphraseProvider = pp
	}
}

func WithClientFn(clientFn func(cfg *config.Config, sctx actor.SecurityContext) (client.DmsClient, error)) func(*DmsCLI) {
	return func(cli *DmsCLI) {
		cli.clientFn = clientFn
	}
}
