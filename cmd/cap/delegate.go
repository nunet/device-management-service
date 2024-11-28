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
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func newDelegateCmd(afs afero.Afero, cfg *config.Config) *cobra.Command {
	var (
		context    string
		caps       []string
		topics     []string
		audience   string
		expiry     time.Time
		duration   time.Duration
		autoExpire bool
		depth      uint64
		selfSign   string
	)

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
			subject := args[0]

			var expirationTime uint64
			switch {
			case !expiry.IsZero():
				expirationTime = uint64(expiry.UnixNano())
			case duration != 0:
				expirationTime = uint64(time.Now().Add(duration).UnixNano())
			case autoExpire:
				expirationTime = 0
			default:
				return fmt.Errorf("either expiration or duration must be specified")
			}

			subjectDID, err := did.FromString(subject)
			if err != nil {
				return fmt.Errorf("invalid subject DID: %w", err)
			}

			var audienceDID did.DID
			if audience != "" {
				audienceDID, err = did.FromString(audience)
				if err != nil {
					return fmt.Errorf("invalid audience DID: %w", err)
				}
			}

			capabilities := make([]ucan.Capability, len(caps))
			for i, cap := range caps {
				capabilities[i] = ucan.Capability(cap)
			}

			var selfSignMode ucan.SelfSignMode
			switch selfSign {
			case "no":
				selfSignMode = ucan.SelfSignNo
			case "also":
				selfSignMode = ucan.SelfSignAlso
			case "only":
				selfSignMode = ucan.SelfSignOnly
			default:
				return fmt.Errorf("invalid self-sign option: %s", selfSign)
			}

			var trustCtx did.TrustContext
			if IsLedgerContext(context) {
				provider, err := did.NewLedgerWalletProvider(0)
				if err != nil {
					return err
				}

				trustCtx = did.NewTrustContextWithProvider(provider)
				context = LedgerContext(context)
			} else {
				trustCtx, _, err = CreateTrustContextFromKeyStore(afs, context, cfg)
				if err != nil {
					return fmt.Errorf("failed to create trust context: %w", err)
				}
			}

			capCtx, err := LoadCapabilityContext(trustCtx, context, cfg)
			if err != nil {
				return fmt.Errorf("failed to load capability context: %w", err)
			}

			tokens, err := capCtx.Delegate(subjectDID, audienceDID, topics, expirationTime, depth, capabilities, selfSignMode)
			if err != nil {
				return fmt.Errorf("failed to delegate capabilities: %w", err)
			}

			tokensJSON, err := json.Marshal(tokens)
			if err != nil {
				return fmt.Errorf("unable to marshal tokens to json: %w", err)
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(tokensJSON))
			return nil
		},
	}

	useFlagContext(cmd, &context)
	useFlagAudience(cmd, &audience)
	useFlagCap(cmd, &caps)
	useFlagTopic(cmd, &topics)
	useFlagExpiry(cmd, &expiry)
	useFlagDuration(cmd, &duration)
	useFlagAutoExpire(cmd, &autoExpire)
	useFlagDepth(cmd, &depth)
	cmd.Flags().StringVar(&selfSign, fnSelfSign, "no", "Self-sign option: 'no', 'also', or 'only'")

	_ = cmd.MarkFlagRequired(fnContext)
	cmd.MarkFlagsOneRequired(fnExpiry, fnDuration, fnAutoExpire)
	cmd.MarkFlagsMutuallyExclusive(fnExpiry, fnDuration, fnAutoExpire)
	cmd.MarkFlagsMutuallyExclusive(fnSelfSign, fnAutoExpire)

	return cmd
}
