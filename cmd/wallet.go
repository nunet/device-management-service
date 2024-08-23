package cmd

import (
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/backend"
)

var walletCmd = NewWalletCmd(networkService)

func NewWalletCmd(net backend.NetworkManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:               "wallet",
		Short:             "Wallet Management",
		PersistentPreRunE: isDMSRunning(net),
		Run: func(cmd *cobra.Command, _ []string) {
			_ = cmd.Help()
		},
	}

	cmd.AddCommand(walletNewCmd)
	return cmd
}
