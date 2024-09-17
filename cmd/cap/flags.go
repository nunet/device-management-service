package cap

import (
	"time"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms"
)

func useFlagContext(cmd *cobra.Command, context *string) {
	cmd.Flags().StringVarP(context, fnContext, "c", dms.UserContextName, "specifies capability context")
}

func useFlagAudience(cmd *cobra.Command, audience *string) {
	cmd.Flags().StringVarP(audience, fnAudience, "a", "", "audience DID (optional)")
}

func useFlagCap(cmd *cobra.Command, caps *[]string) {
	cmd.Flags().StringSliceVar(caps, fnCap, []string{}, "capabilities to grant/delegate (can be specified multiple times)")
}

func useFlagTopic(cmd *cobra.Command, topics *[]string) {
	cmd.Flags().StringSliceVarP(topics, fnTopic, "t", []string{}, "topics to grant/delegate (can be specified multiple times)")
}

func useFlagExpiry(cmd *cobra.Command, expiry *time.Time) {
	cmd.Flags().VarP(utils.NewTimeValue(expiry), fnExpiry, "e", "set expiration date (ISO 8601 format)")
}

func useFlagDuration(cmd *cobra.Command, duration *time.Duration) {
	cmd.Flags().DurationVar(duration, fnDuration, 0, "set duration time (specify unit)")
}

func useFlagDepth(cmd *cobra.Command, depth *uint64) {
	cmd.Flags().Uint64VarP(depth, fnDepth, "d", 0, "delegation depth (optional, default=0)")
}

func useFlagRoot(cmd *cobra.Command, root *string) {
	cmd.Flags().StringVar(root, fnRoot, "", "DID to add as root anchor (represents absolute trust)")
}

func useFlagRequire(cmd *cobra.Command, require *string) {
	cmd.Flags().StringVar(require, fnRequire, "", "JWT to add as require anchor (for input trust)")
}

func useFlagProvide(cmd *cobra.Command, provide *string) {
	cmd.Flags().StringVar(provide, fnProvide, "", "JWT to add as provide anchor (for output trust)")
}
