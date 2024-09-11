package cap

import (
	"time"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms"
)

func useFlagContext(cmd *cobra.Command, context *string) {
	cmd.Flags().StringVarP(context, fnContext, "c", dms.UserContextName, "Operation context; it specifies the key and capability context to use; defaults to user context")
}

func useFlagAudience(cmd *cobra.Command, audience *string) {
	cmd.Flags().StringVarP(audience, fnAudience, "a", "", "Audience DID (optional)")
}

func useFlagCap(cmd *cobra.Command, caps *[]string) {
	cmd.Flags().StringSliceVar(caps, fnCap, []string{}, "Capabilities to grant/delegate (can be specified multiple times)")
}

func useFlagTopic(cmd *cobra.Command, topics *[]string) {
	cmd.Flags().StringSliceVarP(topics, fnTopic, "t", []string{}, "Topics to grant/delegate (can be specified multiple times)")
}

func useFlagExpiry(cmd *cobra.Command, expiry *time.Time) {
	cmd.Flags().VarP(utils.NewTimeValue(expiry), fnExpiry, "e", "Expiration time")
}

func useFlagDuration(cmd *cobra.Command, duration *time.Duration) {
	cmd.Flags().DurationVar(duration, fnDuration, 0, "Duration")
}

func useFlagDepth(cmd *cobra.Command, depth *uint64) {
	cmd.Flags().Uint64VarP(depth, fnDepth, "d", 0, "Delegation depth (optional, default=0)")
}
