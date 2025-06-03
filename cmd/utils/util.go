// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"fmt"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/actor"
	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/env"
	"gitlab.com/nunet/device-management-service/utils"
)

const (
	DefaultUserContextName = "user"
)

func NewSecurityContext(
	fs afero.Afero, env env.EnvironmentProvider,
	context string, cfg *config.Config,
) (actor.SecurityContext, error) {
	if context == "" {
		context = DefaultUserContextName
	}

	// Generate ephemeral key pair
	privk, pubk, err := crypto.GenerateKeyPair(crypto.Ed25519)
	if err != nil {
		return nil, fmt.Errorf("generate ephemeral key pair: %w", err)
	}

	passphrase, err := GetDMSPassphrase(env, false)
	if err != nil {
		return nil, fmt.Errorf("get dms passphrase: %w", err)
	}

	// Create trust context
	trustCtx, err := node.GetTrustContext(fs, context, passphrase, cfg.UserDir)
	if err != nil {
		return nil, fmt.Errorf("create trust context: %w", err)
	}

	// Load capability context
	capCtx, err := node.LoadCapabilityContext(fs, trustCtx, context, cfg.UserDir)
	if err != nil {
		return nil, fmt.Errorf("load capability context: %w", err)
	}

	return actor.NewBasicSecurityContext(pubk, privk, capCtx)
}

func NewClient(cfg *config.Config, sctx actor.SecurityContext) (*client.Client, error) {
	return client.NewClient(client.Config{
		Host:      fmt.Sprintf("%s:%d", cfg.Rest.Addr, cfg.Rest.Port),
		APIPrefix: "/api",
		Version:   "v1",
	}, sctx)
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
