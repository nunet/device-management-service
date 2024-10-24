// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// newTapCommand creates the Cobra command to set up a TAP interface
func newTapCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tap [main_interface] [vm_interface] [CIDR]",
		Short: "Create and configure a TAP interface",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, _ []string) error {
			fmt.Fprintln(cmd.OutOrStdout(), "Not creating tap network interface on MacOs because no firecracker support.")
			return nil
		},
	}
}
