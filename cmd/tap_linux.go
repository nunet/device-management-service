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
	"io"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// newTapCommand creates the Cobra command to set up a TAP interface
func newTapCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tap [main_interface] [vm_interface] [CIDR]",
		Short: "Create and configure a TAP interface",
		Long: `Create a TAP interface using the provided interface name and configure IP forwarding and iptables rules.

Example:
  nunet tap eth0 tap0 172.16.0.1/24

Note: The command requires root privileges.
`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			// check if the user is root
			if os.Getuid() != 0 {
				return fmt.Errorf("this command requires root privileges to execute")
			}

			mainInterface := args[0]
			vmInterface := args[1]
			cidr := args[2]

			// Step 1: Create the TAP interface
			err := runCommand(cmd.OutOrStdout(), fmt.Sprintf("ip tuntap add %s mode tap", vmInterface))
			if err != nil {
				return err
			}

			// Step 2: Assign IP address to the TAP interface
			err = runCommand(cmd.OutOrStdout(), fmt.Sprintf("ip addr add %s dev %s", cidr, vmInterface))
			if err != nil {
				return err
			}

			// Step 3: Bring the TAP interface up
			err = runCommand(cmd.OutOrStdout(), fmt.Sprintf("ip link set %s up", vmInterface))
			if err != nil {
				return err
			}

			// Step 4: Enable IP forwarding
			err = runCommand(cmd.OutOrStdout(), "echo 1 > /proc/sys/net/ipv4/ip_forward")
			if err != nil {
				return err
			}

			// Step 5: Add iptables rules for connection tracking
			err = runCommand(cmd.OutOrStdout(), "iptables -C FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT || iptables -A FORWARD -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT")
			if err != nil {
				return err
			}

			// Step 6: Add iptables rules to allow forwarding between interfaces
			err = runCommand(cmd.OutOrStdout(), fmt.Sprintf("iptables -C FORWARD -i %s -o %s -j ACCEPT || iptables -A FORWARD -i %s -o %s -j ACCEPT", vmInterface, mainInterface, vmInterface, mainInterface))
			if err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "TAP interface %s created and configured successfully\n", vmInterface)
			return nil
		},
	}
}

// Helper function to execute shell commands
func runCommand(stdout io.Writer, command string) error {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to execute command: %s, Error: %w, Output: %s", command, err, output)
	}
	fmt.Fprintf(stdout, "Executed: %s\n", command)
	return nil
}
