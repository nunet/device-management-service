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
	"gitlab.com/nunet/device-management-service/lib/sys"
)

// newTapCommand creates the Cobra command to set up a TAP interface
func newTapCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "tap [main_interface] [vm_interface] [CIDR]",
		Short: "Create and configure a TAP interface",
		Long: `Create a TAP interface using the provided interface name and configure IP forwarding and iptables rules.

Example:
  nunet tap eth0 tap0 172.16.0.1/24

Note: The command requires root privileges or CAP_NET_ADMIN=ep capability.
`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			mainInterface := args[0]
			vmInterface := args[1]
			cidr := args[2]

			// Check if the interfaces
			_, err := sys.GetNetInterfaceByName(mainInterface)
			if err != nil {
				return fmt.Errorf("couldn't read main interface %q", mainInterface)
			}
			_, err = sys.GetNetInterfaceByName(vmInterface)
			if err == nil {
				return fmt.Errorf("interface %q already exists", vmInterface)
			}

			// Create the TAP interface
			iface, err := sys.NewTunTapInterface(vmInterface, sys.NetTapMode, true)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "TAP interface %s created\n", iface.Name())

			// Assign IP address to the TAP interface
			err = iface.SetAddress(cidr)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "IP address %s assigned to TAP interface %s\n", cidr, iface.Name())

			// Bring the TAP interface up
			err = iface.Up()
			if err != nil {
				return err
			}

			// Add iptables rules for connection tracking
			err = sys.AddRelEstRule("FORWARD")
			if err != nil {
				return err
			}

			// Add iptables rules to allow forwarding between interfaces
			err = sys.AddForwardIntRule(vmInterface, mainInterface)
			if err != nil {
				return err
			}

			// Check IP Forwarding kernel parameter
			if enabled, err := sys.ForwardingEnabled(); !enabled || err != nil {
				return fmt.Errorf("IP forwarding looks to be disabled. Please enable it using 'sysctl -w sys.ipv4.ip_forward=1'")
			}

			fmt.Fprintf(cmd.OutOrStdout(), "TAP interface %s created and configured successfully\n", vmInterface)
			return nil
		},
	}
}
