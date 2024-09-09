package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/dms"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run",
		Short: "Start the Device Management Service",
		Long:  `The Device Management Service (DMS) is a system application for computing and service providers. It handles networking and device management.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			passphrase := os.Getenv("DMS_PASSPHRASE")

			var err error
			if passphrase == "" {
				fmt.Print("Please enter the DMS passphrase. This will be used to encrypt/decrypt the keystore containing necessary secrets for DMS:\n")
				passphrase, err = promptForPassphrase()
				if err != nil {
					return fmt.Errorf("error reading passphrase from stdin: %w", err)
				}

				// TODO: validate passphrase (minimum x characters)
				if passphrase == "" {
					return fmt.Errorf("invalid passphrase")
				}
			}

			return dms.Run(passphrase)
		},
	}
}
