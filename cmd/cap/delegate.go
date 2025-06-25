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

// DelegateCapOptions holds the command-line options for the delegate command.
type DelegateCapOptions struct {
	Context    string
	Caps       []string
	Topics     []string
	Audience   string
	Expiry     time.Time
	Duration   time.Duration
	AutoExpire bool
	Depth      uint64
	SelfSign   string
	Subject    string
}

func newDelegateCmd(dmsCLI *cli.DmsCLI) *cobra.Command {
	var opts DelegateCapOptions

	cmd := &cobra.Command{
		Use:   "delegate <did>",
		Short: "Delegate capabilities",
		Long: `Delegate capabilities to a subject

Capabilities are delegated based on provide anchors. No capabilities are delegated by default, you need to use --cap flag to explicitly specify the capabilities to delegate.

Example:
  nunet cap anchor --context user --provide '<token>'
  nunet cap delegate --context user --cap /public --duration 1h did:key:<some-key>`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Subject = args[0]
			return runDelegateCap(cmd.Context(), dmsCLI, opts, cli.CmdStreams(cmd))
		},
	}

	useFlagContext(cmd, &opts.Context)
	useFlagAudience(cmd, &opts.Audience)
	useFlagCap(cmd, &opts.Caps)
	useFlagTopic(cmd, &opts.Topics)
	useFlagExpiry(cmd, &opts.Expiry)
	useFlagDuration(cmd, &opts.Duration)
	useFlagAutoExpire(cmd, &opts.AutoExpire)
	useFlagDepth(cmd, &opts.Depth)
	cmd.Flags().StringVar(&opts.SelfSign, fnSelfSign, "no", "Self-sign option: 'no', 'also', or 'only'")

	_ = cmd.MarkFlagRequired(fnContext)
	cmd.MarkFlagsOneRequired(fnExpiry, fnDuration, fnAutoExpire)
	cmd.MarkFlagsMutuallyExclusive(fnExpiry, fnDuration, fnAutoExpire)
	cmd.MarkFlagsMutuallyExclusive(fnSelfSign, fnAutoExpire)

	return cmd
}

func runDelegateCap(_ context.Context, dmsCLI *cli.DmsCLI, opts DelegateCapOptions, streams cli.Streams) error {
	var expirationTime uint64
	switch {
	case !opts.Expiry.IsZero():
		expirationTime = uint64(opts.Expiry.UnixNano())
	case opts.Duration != 0:
		expirationTime = uint64(time.Now().Add(opts.Duration).UnixNano())
	case opts.AutoExpire:
		expirationTime = 0
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

	var selfSignMode ucan.SelfSignMode
	switch opts.SelfSign {
	case "no":
		selfSignMode = ucan.SelfSignNo
	case "also":
		selfSignMode = ucan.SelfSignAlso
	case "only":
		selfSignMode = ucan.SelfSignOnly
	default:
		return fmt.Errorf("invalid self-sign option: %s", opts.SelfSign)
	}

	capCtx, err := utils.LoadCapabilityContext(dmsCLI, opts.Context)
	if err != nil {
		return err
	}

	tokens, err := capCtx.Delegate(subjectDID, audienceDID, opts.Topics, expirationTime, opts.Depth, capabilities, selfSignMode)
	if err != nil {
		return fmt.Errorf("failed to delegate capabilities: %w", err)
	}

	tokensJSON, err := json.Marshal(tokens)
	if err != nil {
		return fmt.Errorf("unable to marshal tokens to json: %w", err)
	}

	fmt.Fprintln(streams.Out, string(tokensJSON))
	return nil
}
