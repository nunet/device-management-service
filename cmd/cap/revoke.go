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
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

// RevokeCapOptions holds the command-line options for the revoke command.
type RevokeCapOptions struct {
	Context string
	Token   string
}

func newRevokeCmd(dmsCLI *cli.DmsCLI) *cobra.Command {
	var opts RevokeCapOptions

	cmd := &cobra.Command{
		Use:   "revoke <token>",
		Short: "Revoke a token",
		Long: `Revoke a granted or deleated token

Example:
  nunet cap revoke --context user '{"some": "json", "token": "here"}'

The above command revokes a token`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Token = args[0]
			return runRevokeCap(cmd.Context(), dmsCLI, opts, cli.CmdStreams(cmd))
		},
	}

	useFlagContext(cmd, &opts.Context)
	_ = cmd.MarkFlagRequired(fnContext)

	return cmd
}

func runRevokeCap(_ context.Context, dmsCLI *cli.DmsCLI, opts RevokeCapOptions, streams cli.Streams) error {
	capCtx, err := utils.LoadCapabilityContext(dmsCLI, opts.Context)
	if err != nil {
		return err
	}

	var tokens ucan.TokenList
	if err := json.Unmarshal([]byte(opts.Token), &tokens); err != nil {
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
	fmt.Fprintln(streams.Out, string(outputJSON))

	return nil
}
