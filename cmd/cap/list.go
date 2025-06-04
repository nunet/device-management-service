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
	"strings"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

type ListCapOptions struct {
	Context string
}

func newListCmd(dmsCLI *cli.DmsCLI) *cobra.Command {
	var opts ListCapOptions

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List capability anchors",
		Long: `List all capability anchors in a capability context

It outputs DIDs and capability tokens set for root, provide, require and revoke anchors.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runListCap(cmd.Context(), dmsCLI, opts, cli.CmdStreams(cmd))
		},
	}

	useFlagContext(cmd, &opts.Context)

	return cmd
}

func runListCap(_ context.Context, dmsCLI *cli.DmsCLI, opts ListCapOptions, streams cli.Streams) error {
	capCtx, err := utils.LoadCapabilityContext(dmsCLI, opts.Context)
	if err != nil {
		return err
	}

	roots, require, provide, revoke := capCtx.ListRoots()

	list, err := formatCapabilityList(roots, require, provide, revoke)
	if err != nil {
		return fmt.Errorf("failed to format capability list: %w", err)
	}

	fmt.Fprint(streams.Out, list)
	return nil
}

func formatCapabilityList(roots []did.DID, require, provide, revoke ucan.TokenList) (string, error) {
	var sb strings.Builder

	fmt.Fprintf(&sb, "roots:\n")
	for _, root := range roots {
		fmt.Fprintf(&sb, "\t%s\n", root)
	}

	fmt.Fprintf(&sb, "require:\n")
	for _, t := range require.Tokens {
		data, err := json.Marshal(t)
		if err != nil {
			return "", fmt.Errorf("failed to marshal capability token: %w", err)
		}
		fmt.Fprintf(&sb, "\t%s\n", string(data))
	}

	fmt.Fprintf(&sb, "provide:\n")
	for _, t := range provide.Tokens {
		data, err := json.Marshal(t)
		if err != nil {
			return "", fmt.Errorf("failed to marshal capability token: %w", err)
		}
		fmt.Fprintf(&sb, "\t%s\n", string(data))
	}

	fmt.Fprintf(&sb, "revoke:\n")
	for _, t := range revoke.Tokens {
		data, err := json.Marshal(t)
		if err != nil {
			return "", fmt.Errorf("failed to marshal capability token: %w", err)
		}
		fmt.Fprintf(&sb, "\t%s\n", string(data))
	}

	return sb.String(), nil
}
