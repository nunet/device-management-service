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

func newGrantCmd(afs afero.Afero, cfg *config.Config) *cobra.Command {
	var (
		context  string
		caps     []string
		topics   []string
		audience string
		expiry   time.Time
		duration time.Duration
		depth    uint64
	)

	cmd := &cobra.Command{
		Use:   "grant <did>",
		Short: "Grant capabilities",
		Long: `Grant a self-sign token delegating capabilities

It is not necessary to set up a anchor before granting a capability because this operation is self-signed.

Example:
  nunet cap grant --context user --cap /public --duration 1h did:key:<some-key>

The above command emits a self-signed token with the specified capabilities delegated from 'user' to the sbjects's DID. `,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			subject := args[0]

			var expirationTime uint64
			switch {
			case !expiry.IsZero():
				expirationTime = uint64(expiry.UnixNano())
			case duration != 0:
				expirationTime = uint64(time.Now().Add(duration).UnixNano())
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

			tokens, err := capCtx.Grant(ucan.Delegate, subjectDID, audienceDID, topics, expirationTime, depth, capabilities)
			if err != nil {
				return fmt.Errorf("failed to grant capabilities: %w", err)
			}

			tokensJSON, err := json.Marshal(tokens)
			if err != nil {
				return fmt.Errorf("unable to marshal tokens to json: %w", err)
			}

			// fmt.Println(string(tokensJSON))
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
	useFlagDepth(cmd, &depth)

	_ = cmd.MarkFlagRequired(fnContext)
	cmd.MarkFlagsOneRequired(fnExpiry, fnDuration)

	return cmd
}
