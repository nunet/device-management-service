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
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/env"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func newRemoveCmd(afs afero.Afero, env env.EnvironmentProvider, cfg *config.Config) *cobra.Command {
	var (
		context string
		root    string
		provide string
		require string
	)

	const (
		fnProvide = "provide"
		fnRoot    = "root"
		fnRequire = "require"
	)

	cmd := &cobra.Command{
		Use:   "remove",
		Short: "Remove capability anchors",
		Long: `Remove capability anchors in a capability context

One capability anchor must be specified at a time.

Example:
  nunet cap remove --context user --root did:key:abcd1234
  nunet cap remove --context user --require '<the-token>'`,
		RunE: func(_ *cobra.Command, _ []string) error {
			passphrase, err := utils.GetDMSPassphrase(env, false)
			if err != nil {
				return fmt.Errorf("get dms passphrase: %w", err)
			}

			trustCtx, err := node.GetTrustContext(afs, context, passphrase, cfg.UserDir)
			if err != nil {
				return fmt.Errorf("get trust context: %w", err)
			}

			capCtx, err := node.LoadCapabilityContext(afs, trustCtx, context, cfg.UserDir)
			if err != nil {
				return fmt.Errorf("failed to load capability context: %w", err)
			}

			switch {
			case root != "":
				rootDID, err := did.FromString(root)
				if err != nil {
					return fmt.Errorf("invalid root DID: %w", err)
				}

				capCtx.RemoveRoots([]did.DID{rootDID}, ucan.TokenList{}, ucan.TokenList{})

			case require != "":
				var token ucan.Token
				if err := json.Unmarshal([]byte(require), &token); err != nil {
					return fmt.Errorf("unmarshal tokens: %w", err)
				}

				capCtx.RemoveRoots(nil, ucan.TokenList{Tokens: []*ucan.Token{&token}}, ucan.TokenList{})

			case provide != "":
				var token ucan.Token
				if err := json.Unmarshal([]byte(provide), &token); err != nil {
					return fmt.Errorf("unmarshal tokens: %w", err)
				}

				capCtx.RemoveRoots(nil, ucan.TokenList{}, ucan.TokenList{Tokens: []*ucan.Token{&token}})

			default:
				return fmt.Errorf("one of --provide, --root, or --require must be specified")
			}

			if err := node.SaveCapabilityContext(afs, capCtx, cfg.UserDir); err != nil {
				return fmt.Errorf("save capability context: %w", err)
			}

			return nil
		},
	}

	useFlagContext(cmd, &context)
	useFlagRoot(cmd, &root)
	useFlagRequire(cmd, &require)
	useFlagProvide(cmd, &provide)

	_ = cmd.MarkFlagRequired(fnContext)
	cmd.MarkFlagsOneRequired(fnProvide, fnRoot, fnRequire)
	cmd.MarkFlagsMutuallyExclusive(fnProvide, fnRoot, fnRequire)

	return cmd
}
