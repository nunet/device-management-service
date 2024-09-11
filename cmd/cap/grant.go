package cap

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func newGrantCmd(afs afero.Afero) *cobra.Command {
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
		Use:   "grant <subjectDID>",
		Short: "Grant capabilities",
		Long:  `Grant (delegate) capabilities as anchors and side chains from a capability context`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
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

			trustCtx, _, err := CreateTrustContextFromKeyStore(afs, context)
			if err != nil {
				return fmt.Errorf("failed to create trust context: %w", err)
			}

			capCtx, err := LoadCapabilityContext(trustCtx, context)
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

			fmt.Println(string(tokensJSON))
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
