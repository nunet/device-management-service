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

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/internal/config"
)

func newListCmd(afs afero.Afero, cfg *config.Config) *cobra.Command {
	var context string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List capability anchors",
		Long: `List all capability anchors in a capability context

It outputs DIDs and capability tokens set for root, provide, require and revoke anchors.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			trustCtx, err := node.GetTrustContext(afs, context, cfg.UserDir)
			if err != nil {
				return fmt.Errorf("get trust context: %w", err)
			}

			capCtx, err := node.LoadCapabilityContext(trustCtx, context, cfg.UserDir)
			if err != nil {
				return fmt.Errorf("failed to load capability context: %w", err)
			}

			roots, require, provide, revoke := capCtx.ListRoots()

			fmt.Println("roots:")
			for _, root := range roots {
				fmt.Printf("\t%s\n", root)
			}

			fmt.Println("require:")
			for _, t := range require.Tokens {
				data, err := json.Marshal(t)
				if err != nil {
					return fmt.Errorf("failed to marshal capability token: %w", err)
				}
				fmt.Printf("\t%s\n", string(data))
			}

			fmt.Println("provide:")
			for _, t := range provide.Tokens {
				data, err := json.Marshal(t)
				if err != nil {
					return fmt.Errorf("failed to marshal capability token: %w", err)
				}
				fmt.Printf("\t%s\n", string(data))
			}

			fmt.Println("revoke:")
			for _, t := range revoke.Tokens {
				data, err := json.Marshal(t)
				if err != nil {
					return fmt.Errorf("failed to marshal capability token: %w", err)
				}
				fmt.Printf("\t%s\n", string(data))
			}
			return nil
		},
	}

	useFlagContext(cmd, &context)

	return cmd
}
