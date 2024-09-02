package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/utils"
)

// NewWalletCmd is a constructor for `wallet` parent command
func newWalletCmd(client *utils.HTTPClient) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "wallet",
		Short: "Wallet Management",
		Run: func(cmd *cobra.Command, _ []string) {
			_ = cmd.Help()
		},
	}
	cmd.AddCommand(newWalletNewCmd(client))
	return cmd
}
