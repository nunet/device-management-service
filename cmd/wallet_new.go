package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/cmd/backend"
	"gitlab.com/nunet/device-management-service/types"
)

var (
	walletNewCmd     = NewWalletNewCmd(walletService)
	flagEth, flagAda bool
)

func NewWalletNewCmd(wallet backend.WalletManager) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create new wallet",
		RunE: func(cmd *cobra.Command, args []string) error {
			eth, _ := cmd.Flags().GetBool("ethereum")
			ada, _ := cmd.Flags().GetBool("cardano")

			var pair *types.BlockchainAddressPrivKey
			var err error

			// limit wallet creation to one at a time
			if ada && eth {
				return fmt.Errorf("cannot create both wallets")
			} else if ada {
				pair, err = wallet.GetCardanoAddressAndMnemonic()
			} else if eth {
				pair, err = wallet.GetEthereumAddressAndPrivateKey()
			} else {
				cmd.Help()
				return fmt.Errorf("no wallet flag specified")
			}

			// share error handling for both addresses
			if err != nil {
				return fmt.Errorf("generate wallet address failed: %w", err)
			}

			printWallet(cmd.OutOrStdout(), pair)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&flagEth, "ethereum", "e", false, "create Ethereum wallet")
	cmd.Flags().BoolVarP(&flagAda, "cardano", "c", false, "create Cardano wallet")

	return cmd
}
