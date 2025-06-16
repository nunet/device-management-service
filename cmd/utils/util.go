// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"bytes"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/env"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	"gitlab.com/nunet/device-management-service/utils"
)

const (
	DefaultUserContextName = "user"
)

func NewSecurityContext(
	dmsCLI *cli.DmsCLI,
	context string,
) (actor.SecurityContext, error) {
	if context == "" {
		context = DefaultUserContextName
	}

	// Generate ephemeral key pair
	privk, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	if err != nil {
		return nil, fmt.Errorf("generate ephemeral key pair: %w", err)
	}

	capCtx, err := LoadCapabilityContext(dmsCLI, context)
	if err != nil {
		return nil, fmt.Errorf("load capability context: %w", err)
	}

	return actor.NewBasicSecurityContext(pubk, privk, capCtx)
}

func NewCapabilityContext(dmsCLI *cli.DmsCLI, context string) (ucan.CapabilityContext, did.DID, error) {
	if context == "" {
		context = DefaultUserContextName
	}

	var ctxDID did.DID

	cfg, err := dmsCLI.Config()
	if err != nil {
		return nil, ctxDID, fmt.Errorf("get config: %w", err)
	}

	fs := dmsCLI.FS()

	keyStoreDir := filepath.Join(cfg.UserDir, node.KeystoreDir)
	ks, err := keystore.New(fs, keyStoreDir)
	if err != nil {
		return nil, ctxDID, fmt.Errorf("create keystore: %w", err)
	}

	passphrase, err := dmsCLI.Passphrase(context)
	if err != nil {
		return nil, ctxDID, fmt.Errorf("get passphrase: %w", err)
	}

	priv, err := dms.GenerateAndStorePrivKey(ks, passphrase, context)
	if err != nil {
		return nil, ctxDID, fmt.Errorf("generate and store private key: %w", err)
	}

	ctxDID = did.FromPublicKey(priv.GetPublic())

	trustCtx, err := did.NewTrustContextWithPrivateKey(priv)
	if err != nil {
		return nil, ctxDID, fmt.Errorf("create trust context: %w", err)
	}

	capCtx, err := ucan.NewCapabilityContextWithName(context, trustCtx, ctxDID, nil, ucan.TokenList{}, ucan.TokenList{}, ucan.TokenList{})
	if err != nil {
		return nil, ctxDID, fmt.Errorf("create capability context: %w", err)
	}

	if err := SaveCapabilityContext(dmsCLI, capCtx); err != nil {
		return nil, ctxDID, fmt.Errorf("save capability context: %w", err)
	}

	return capCtx, ctxDID, nil
}

// loadCapabilityContext is a helper function to reduce boilerplate in commands.
// It handles the common steps of loading a capability context: getting config,
// retrieving passphrase, loading trust context, and finally loading capability context.
func LoadCapabilityContext(dmsCLI *cli.DmsCLI, contextName string) (ucan.CapabilityContext, error) {
	if contextName == "" {
		contextName = DefaultUserContextName
	}

	cfg, err := dmsCLI.Config()
	if err != nil {
		return nil, fmt.Errorf("unable to get config: %w", err)
	}

	fs := dmsCLI.FS()

	passphrase, err := dmsCLI.Passphrase(contextName)
	if err != nil {
		return nil, fmt.Errorf("get dms passphrase: %w", err)
	}

	trustCtx, err := node.GetTrustContext(fs, contextName, passphrase, cfg.UserDir)
	if err != nil {
		return nil, fmt.Errorf("get trust context: %w", err)
	}

	capCtx, err := node.LoadCapabilityContext(trustCtx, fs, contextName, cfg.UserDir)
	if err != nil {
		return nil, fmt.Errorf("failed to load capability context: %w", err)
	}

	return capCtx, nil
}

// saveCapabilityContext is a helper function to save a capability context
func SaveCapabilityContext(dmsCLI *cli.DmsCLI, capCtx ucan.CapabilityContext) error {
	cfg, err := dmsCLI.Config()
	if err != nil {
		return fmt.Errorf("unable to get config: %w", err)
	}

	fs := dmsCLI.FS()

	if err := node.SaveCapabilityContext(capCtx, fs, cfg.UserDir); err != nil {
		return fmt.Errorf("save capability context: %w", err)
	}

	return nil
}

func NewClient(cfg *config.Config, sctx actor.SecurityContext) (client.DmsClient, error) {
	return client.NewClient(client.Config{
		Host:      fmt.Sprintf("%s:%d", cfg.Rest.Addr, cfg.Rest.Port),
		APIPrefix: "/api",
		Version:   "v1",
	}, sctx)
}

func NewTestCli(opts ...func(*cli.DmsCLI)) *cli.DmsCLI {
	defaults := []func(*cli.DmsCLI){}

	env := env.NewMockEnvironment()
	err := env.Setenv("DMS_PASSPHRASE", "pass")
	if err == nil {
		defaults = append(defaults, cli.WithEnv(env))
	}

	fs := afero.NewMemMapFs()
	cfg := &config.Config{General: config.General{
		UserDir: "/tmp/nunet/user",
		WorkDir: "/tmp/nunet/work",
		DataDir: "/tmp/nunet/data",
	}}

	defaults = append(defaults, cli.WithFS(fs), cli.WithConfig(cfg))

	dmsCli := cli.New(append(defaults, opts...)...)

	return dmsCli
}

func GetDMSPassphrase(
	env env.EnvironmentProvider, withConfirm bool,
) (string, error) {
	var err error
	passphrase := env.Getenv(node.DMSPassphraseEnv)
	if passphrase == "" {
		passphrase, err = utils.PromptForPassphrase(withConfirm)
		if err != nil {
			return "", fmt.Errorf("failed to get passphrase: %w", err)
		}
	}

	return passphrase, nil
}

func ExecuteCommand(
	command *cobra.Command, args ...string,
) (stdout, stderr string, err error) {
	var stdoutBuf, stderrBuf bytes.Buffer

	// Redirect to our buffers
	command.SetOut(&stdoutBuf)
	command.SetErr(&stderrBuf)

	// Set args and execute the command
	command.SetArgs(args)
	err = command.Execute()

	return stdoutBuf.String(), stderrBuf.String(), err
}
