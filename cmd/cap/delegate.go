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

func newDelegateCmd(afs afero.Afero) *cobra.Command {
	var (
		context  string
		caps     []string
		topics   []string
		audience string
		expiry   time.Time
		duration time.Duration
		depth    uint64
		selfSign string
	)

	cmd := &cobra.Command{
		Use:   "delegate <subjectDID>",
		Short: "Delegate capabilities",
		Long:  `Delegate capabilities for a subject`,
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

			trustCtx, _, err := CreateTrustContextFromKeyStore(afs, context)
			if err != nil {
				return fmt.Errorf("failed to create trust context: %w", err)
			}

			capCtx, err := LoadCapabilityContext(trustCtx, context)
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
	cmd.Flags().StringVar(&selfSign, "self-sign", "no", "Self-sign option: 'no', 'also', or 'only'")

	_ = cmd.MarkFlagRequired(fnContext)
	cmd.MarkFlagsOneRequired(fnExpiry, fnDuration)

	return cmd
}
