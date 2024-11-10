// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

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

func useFlagAutoExpire(cmd *cobra.Command, autoExpire *bool) {
	cmd.Flags().BoolVar(autoExpire, fnAutoExpire, false, "set auto expiration")
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
