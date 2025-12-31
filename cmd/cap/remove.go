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
	"os"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

// RemoveCapOptions holds the command-line options for the remove command.
type RemoveCapOptions struct {
	Context string
	Root    string
	Provide string
	Require string
}

func newRemoveCmd(dmsCLI *cli.DmsCLI) *cobra.Command {
	var opts RemoveCapOptions

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
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runRemoveCap(cmd.Context(), dmsCLI, opts, cli.CmdStreams(cmd))
		},
	}

	useFlagContext(cmd, &opts.Context)
	useFlagRoot(cmd, &opts.Root)
	useFlagRequire(cmd, &opts.Require)
	useFlagProvide(cmd, &opts.Provide)

	_ = cmd.MarkFlagRequired(fnContext)
	cmd.MarkFlagsOneRequired(fnProvide, fnRoot, fnRequire)
	cmd.MarkFlagsMutuallyExclusive(fnProvide, fnRoot, fnRequire)

	return cmd
}

func runRemoveCap(_ context.Context, dmsCLI *cli.DmsCLI, opts RemoveCapOptions, _ cli.Streams) error {
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

		capCtx.RemoveRoots([]did.DID{rootDID}, ucan.TokenList{}, ucan.TokenList{})

	case opts.Require != "":
		var token ucan.Token
		if err := json.Unmarshal([]byte(opts.Require), &token); err != nil {
			return fmt.Errorf("unmarshal tokens: %w", err)
		}

		capCtx.RemoveRoots(nil, ucan.TokenList{Tokens: []*ucan.Token{&token}}, ucan.TokenList{})

	case opts.Provide != "":
		var token ucan.Token
		if err := json.Unmarshal([]byte(opts.Provide), &token); err != nil {
			return fmt.Errorf("unmarshal tokens: %w", err)
		}

		capCtx.RemoveRoots(nil, ucan.TokenList{}, ucan.TokenList{Tokens: []*ucan.Token{&token}})

	default:
		return fmt.Errorf("one of --provide, --root, or --require must be specified")
	}

	if err := utils.SaveCapabilityContext(dmsCLI, capCtx); err != nil {
		return err
	}

	// Send SIGUSR1 to running DMS to reload contexts
	if err := signalDMSReload(dmsCLI); err != nil {
		// Log the error but don't fail - DMS might not be running (expected during initial setup)
		fmt.Fprintf(os.Stderr, "Warning: Could not signal DMS to reload (DMS may not be running): %v\n", err)
	} else {
		fmt.Println("Successfully signaled DMS to reload capability contexts")
	}

	return nil
}
