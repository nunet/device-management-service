package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

// NewWalletNewCmd is a constructor for `wallet new` command
func newWalletNewCmd(client *utils.HTTPClient) *cobra.Command {
	fnEth := "ethereum"
	fnCardano := "cardano"

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Create new wallet",
		RunE: func(cmd *cobra.Command, _ []string) error {
			eth, _ := cmd.Flags().GetBool(fnEth)

			var (
				pair  *types.BlockchainAddressPrivKey
				query string
			)

			if eth {
				query = "?blockchain=ethereum"
			}
			resBody, resCode, err := client.MakeRequest("GET", "/onboarding/address/new"+query, nil)
			if err != nil {
				return fmt.Errorf("could not make request: %w", err)
			}

			if resCode != 200 {
				return fmt.Errorf("request failed with status code: %d", resCode)
			}

			if err := json.Unmarshal(resBody, &pair); err != nil {
				return fmt.Errorf("could not unmarshal response body: %w", err)
			}

			printWallet(cmd.OutOrStdout(), pair)
			return nil
		},
	}

	cmd.Flags().BoolP(fnEth, "e", false, "create Ethereum wallet")
	cmd.Flags().BoolP(fnCardano, "c", false, "create Cardano wallet")
	cmd.MarkFlagsOneRequired(fnEth, fnCardano)
	cmd.MarkFlagsMutuallyExclusive(fnEth, fnCardano)

	return cmd
}
