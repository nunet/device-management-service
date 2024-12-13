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

	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func newAnchorCmd(afs afero.Afero, cfg *config.Config) *cobra.Command {
	var (
		context string
		root    string
		provide string
		require string
		revoke  string
	)

	const (
		fnProvide = "provide"
		fnRoot    = "root"
		fnRequire = "require"
		fnRevoke  = "revoke"
	)

	cmd := &cobra.Command{
		Use:   "anchor",
		Short: "Manage capability anchors",
		Long: `Add or modify capability anchors in a capability context.

An anchor is a basis of trust in the capability system. There are three types of anchors:

1. Root anchor: Represents absolute trust or effective root capability.
   Use the --root flag with a DID value to add a root anchor.

2. Require anchor: Represents input trust. We verify incoming messages based on the require anchor.
   Use the --require flag with a token to add a require anchor.

3. Provide anchor: Represents output trust. We emit invocation tokens based on our provide anchors and sign output.
   Use the --provide flag with a token to add a provide anchor.

Only one type of anchor can be added or modified per command execution.

Usage examples:
  nunet cap anchor --context user --root did:example:123456789abcdefghi
  nunet cap anchor --context dms  --require '{"some": "json", "token": "here"}'
  nunet cap anchor --context user --provide '{"another": "json", "token": "example"}'
  nunet cap anchor --context user --revoke '{"another": "revocation", "token": "example"}'

Note: The --context flag is required to specify the capability context.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			var trustCtx did.TrustContext
			if node.IsLedgerContext(context) {
				provider, err := did.NewLedgerWalletProvider(0)
				if err != nil {
					return err
				}

				trustCtx = did.NewTrustContextWithProvider(provider)
				context = node.LedgerContext(context)
			} else {
				var err error
				trustCtx, _, err = node.CreateTrustContextFromKeyStore(afs, context, cfg)
				if err != nil {
					return fmt.Errorf("failed to create trust context: %w", err)
				}
			}

			capCtx, err := node.LoadCapabilityContext(trustCtx, context, cfg)
			if err != nil {
				return fmt.Errorf("failed to load capability context: %w", err)
			}

			switch {
			case root != "":
				rootDID, err := did.FromString(root)
				if err != nil {
					return fmt.Errorf("invalid root DID: %w", err)
				}

				if err := capCtx.AddRoots([]did.DID{rootDID}, ucan.TokenList{}, ucan.TokenList{}, ucan.TokenList{}); err != nil {
					return fmt.Errorf("failed to add root anchors: %w", err)
				}

			case require != "":
				var tokens ucan.TokenList
				if err := json.Unmarshal([]byte(require), &tokens); err != nil {
					return fmt.Errorf("unmarshal tokens: %w", err)
				}

				if err := capCtx.AddRoots(nil, tokens, ucan.TokenList{}, ucan.TokenList{}); err != nil {
					return fmt.Errorf("failed to add require anchors: %w", err)
				}

			case provide != "":
				var tokens ucan.TokenList
				if err := json.Unmarshal([]byte(provide), &tokens); err != nil {
					return fmt.Errorf("unmarshal tokens: %w", err)
				}

				if err := capCtx.AddRoots(nil, ucan.TokenList{}, tokens, ucan.TokenList{}); err != nil {
					return fmt.Errorf("failed to add provide anchors: %w", err)
				}
			case revoke != "":
				var tokens ucan.TokenList
				if err := json.Unmarshal([]byte(revoke), &tokens); err != nil {
					return fmt.Errorf("unmarshal tokens: %w", err)
				}

				if err := capCtx.AddRoots(nil, ucan.TokenList{}, ucan.TokenList{}, tokens); err != nil {
					return fmt.Errorf("failed to add revoke anchors: %w", err)
				}

			default:
				return fmt.Errorf("one of --provide, --root, or --require or --revoke must be specified")
			}

			if err := node.SaveCapabilityContext(capCtx, cfg); err != nil {
				return fmt.Errorf("save capability context: %w", err)
			}

			return nil
		},
	}

	useFlagContext(cmd, &context)
	useFlagRoot(cmd, &root)
	useFlagRequire(cmd, &require)
	useFlagProvide(cmd, &provide)
	useFlagRevoke(cmd, &revoke)

	_ = cmd.MarkFlagRequired(fnContext)
	cmd.MarkFlagsOneRequired(fnProvide, fnRoot, fnRequire, fnRevoke)
	cmd.MarkFlagsMutuallyExclusive(fnProvide, fnRoot, fnRequire, fnRevoke)

	return cmd
}
