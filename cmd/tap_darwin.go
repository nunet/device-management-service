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
