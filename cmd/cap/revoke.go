// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cap

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/env"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func newRevokeCmd(afs afero.Afero, env env.EnvironmentProvider, cfg *config.Config) *cobra.Command {
	var context string

	cmd := &cobra.Command{
		Use:   "revoke <token>",
		Short: "Revoke a token",
		Long: `Revoke a granted or deleated token

Example:
  nunet cap revoke --context user '{"some": "json", "token": "here"}'

The above command revokes a token`,
		RunE: func(cmd *cobra.Command, args []string) error {
			passphrase, err := utils.GetDMSPassphrase(env, false)
			if err != nil {
				return fmt.Errorf("get dms passphrase: %w", err)
			}

			trustCtx, err := node.GetTrustContext(afs, context, passphrase, cfg.UserDir)
			if err != nil {
				return fmt.Errorf("get trust context: %w", err)
			}

			capCtx, err := node.LoadCapabilityContext(trustCtx, context, cfg.UserDir)
			if err != nil {
				return fmt.Errorf("failed to load capability context: %w", err)
			}

			var tokens ucan.TokenList
			if err := json.Unmarshal([]byte(args[0]), &tokens); err != nil {
				return fmt.Errorf("unmarshal tokens: %w", err)
			}

			var outputJSON []byte
			for _, token := range tokens.Tokens {
				revocationTokens, err := capCtx.Revoke(token)
				if err != nil {
					return fmt.Errorf("failed to revoke: %w", err)
				}
				tokensJSON, err := json.Marshal(revocationTokens)
				if err != nil {
					return fmt.Errorf("unable to marshal tokens to json: %w", err)
				}

				outputJSON = append(outputJSON, tokensJSON...)
				outputJSON = append(outputJSON, []byte("\n")...)
			}
			fmt.Println("Revoked tokens", string(outputJSON))
			fmt.Fprintln(cmd.OutOrStdout(), string(outputJSON))

			return nil
		},
	}

	useFlagContext(cmd, &context)
	_ = cmd.MarkFlagRequired(fnContext)

	return cmd
}
