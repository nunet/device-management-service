// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cap

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

// AnchorOptions holds the command-line options for the anchor command.
type AnchorOptions struct {
	Context string
	Root    string
	Provide string
	Require string
	Revoke  string
}

func newAnchorCmd(dmsCLI *cli.DmsCLI) *cobra.Command {
	var opts AnchorOptions

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runAnchorCmd(cmd.Context(), dmsCLI, opts, cli.CmdStreams(cmd))
		},
	}

	useFlagContext(cmd, &opts.Context)
	useFlagRoot(cmd, &opts.Root)
	useFlagRequire(cmd, &opts.Require)
	useFlagProvide(cmd, &opts.Provide)
	useFlagRevoke(cmd, &opts.Revoke)

	_ = cmd.MarkFlagRequired(fnContext)
	cmd.MarkFlagsOneRequired(fnProvide, fnRoot, fnRequire, fnRevoke)
	cmd.MarkFlagsMutuallyExclusive(fnProvide, fnRoot, fnRequire, fnRevoke)

	return cmd
}

func runAnchorCmd(_ context.Context, dmsCLI *cli.DmsCLI, opts AnchorOptions, _ cli.Streams) error {
	capCtx, err := utils.LoadCapabilityContext(dmsCLI, opts.Context)
	if err != nil {
		return err
	}

	switch {
	case opts.Root != "":
		rootDID, err := did.FromString(opts.Root)
		if err != nil {
			return fmt.Errorf("invalid root DID: %w", err)
		}

		if err := capCtx.AddRoots([]did.DID{rootDID}, ucan.TokenList{}, ucan.TokenList{}, ucan.TokenList{}); err != nil {
			return fmt.Errorf("failed to add root anchors: %w", err)
		}

	case opts.Require != "":
		var tokens ucan.TokenList
		if err := json.Unmarshal([]byte(opts.Require), &tokens); err != nil {
			return fmt.Errorf("unmarshal tokens: %w", err)
		}

		if err := capCtx.AddRoots(nil, tokens, ucan.TokenList{}, ucan.TokenList{}); err != nil {
			return fmt.Errorf("failed to add require anchors: %w", err)
		}

	case opts.Provide != "":
		var tokens ucan.TokenList
		if err := json.Unmarshal([]byte(opts.Provide), &tokens); err != nil {
			return fmt.Errorf("unmarshal tokens: %w", err)
		}

		if err := capCtx.AddRoots(nil, ucan.TokenList{}, tokens, ucan.TokenList{}); err != nil {
			return fmt.Errorf("failed to add provide anchors: %w", err)
		}
	case opts.Revoke != "":
		var token ucan.Token
		if err := json.Unmarshal([]byte(opts.Revoke), &token); err != nil {
			return fmt.Errorf("unmarshal tokens: %w", err)
		}

		if err := capCtx.AddRoots(nil, ucan.TokenList{}, ucan.TokenList{}, ucan.TokenList{Tokens: []*ucan.Token{&token}}); err != nil {
			return fmt.Errorf("failed to add revoke anchors: %w", err)
		}

	default:
		return fmt.Errorf("one of --provide, --root, or --require or --revoke must be specified")
	}

	if err := utils.SaveCapabilityContext(dmsCLI, capCtx); err != nil {
		return err
	}

	return nil
}
