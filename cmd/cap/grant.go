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
	"time"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

type grantOptions struct {
	Context  string
	Caps     []string
	Topics   []string
	Audience string
	Expiry   time.Time
	Duration time.Duration
	Depth    uint64
	Subject  string
}

func newGrantCmd(dmsCLI *cli.DmsCLI) *cobra.Command {
	var opts grantOptions

	cmd := &cobra.Command{
		Use:   "grant <did>",
		Short: "Grant capabilities",
		Long: `Grant a self-sign token delegating capabilities

It is not necessary to set up a anchor before granting a capability because this operation is self-signed.

Example:
  nunet cap grant --context user --cap /public --duration 1h did:key:<some-key>
`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Subject = args[0]
			return runGrantCmd(cmd.Context(), dmsCLI, opts, cli.CmdStreams(cmd))
		},
	}

	useFlagContext(cmd, &opts.Context)
	useFlagCap(cmd, &opts.Caps)
	useFlagTopic(cmd, &opts.Topics)
	useFlagAudience(cmd, &opts.Audience)
	useFlagExpiry(cmd, &opts.Expiry)
	useFlagDuration(cmd, &opts.Duration)
	useFlagDepth(cmd, &opts.Depth)

	_ = cmd.MarkFlagRequired(fnContext)
	cmd.MarkFlagsOneRequired(fnExpiry, fnDuration)

	return cmd
}

func runGrantCmd(_ context.Context, dmsCLI *cli.DmsCLI, opts grantOptions, streams cli.Streams) error {
	var expirationTime uint64
	switch {
	case !opts.Expiry.IsZero():
		expirationTime = uint64(opts.Expiry.UnixNano())
	case opts.Duration != 0:
		expirationTime = uint64(time.Now().Add(opts.Duration).UnixNano())
	default:
		return fmt.Errorf("either expiration or duration must be specified")
	}

	subjectDID, err := did.FromString(opts.Subject)
	if err != nil {
		return fmt.Errorf("invalid subject DID: %w", err)
	}

	var audienceDID did.DID
	if opts.Audience != "" {
		audienceDID, err = did.FromString(opts.Audience)
		if err != nil {
			return fmt.Errorf("invalid audience DID: %w", err)
		}
	}

	capabilities := make([]ucan.Capability, len(opts.Caps))
	for i, cap := range opts.Caps {
		capabilities[i] = ucan.Capability(cap)
	}

	capCtx, err := utils.LoadCapabilityContext(dmsCLI, opts.Context)
	if err != nil {
		return err
	}

	tokens, err := capCtx.Grant(ucan.Delegate, subjectDID, audienceDID, opts.Topics, expirationTime, opts.Depth, capabilities)
	if err != nil {
		return fmt.Errorf("failed to grant capabilities: %w", err)
	}

	tokensJSON, err := json.Marshal(tokens)
	if err != nil {
		return fmt.Errorf("unable to marshal tokens to json: %w", err)
	}

	fmt.Fprintln(streams.Out, string(tokensJSON))
	return nil
}
