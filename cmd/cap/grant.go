package cap

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func newGrantCmd(afs afero.Afero) *cobra.Command {
	var (
		context  string
		caps     []string
		topics   []string
		audience string
		expire   int64
		duration int64
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
			case expire != 0:
				expirationTime = uint64(time.Duration(expire) * time.Second)
			case duration != 0:
				expirationTime = uint64(time.Now().UnixNano() + int64(time.Duration(duration)*time.Second))
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

	cmd.Flags().StringVar(&context, flagContext, dms.UserContextName, "Operation context; it specifies the key and capability context to use; defaults to user context")
	cmd.Flags().StringVar(&audience, flagAudience, "", "Audience DID (optional)")
	cmd.Flags().StringSliceVar(&caps, flagCap, []string{}, "Capabilities to grant (can be specified multiple times)")
	cmd.Flags().StringSliceVar(&topics, flagTopic, []string{}, "Topics to grant (can be specified multiple times)")
	// TODO parse dates; this is ugly
	cmd.Flags().Int64Var(&expire, flagExpire, 0, "Expiration time as Unix timestamp")
	// TODO parse duration; this is ugly
	cmd.Flags().Int64Var(&duration, flagDuration, 0, "Duration in seconds from now")
	cmd.Flags().Uint64Var(&depth, flagDepth, 0, "Delegation depth (optional, default=0)")

	cmd.MarkFlagsOneRequired("expire", "duration")

	return cmd
}
