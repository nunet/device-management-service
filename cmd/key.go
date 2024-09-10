package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/utils"
)

func newKeyCmd(fs afero.Afero) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage keys",
		Long:  `Manage keys for the Device Management Service`,
	}

	cmd.AddCommand(newKeyNewCmd(fs))
	cmd.AddCommand(newKeyDIDCmd(fs))

	return cmd
}

func newKeyNewCmd(fs afero.Afero) *cobra.Command {
	return &cobra.Command{
		Use:   "new <name>",
		Short: "Generate a new keypair",
		Long:  `Generate a new keypair, saving the private key into user's local keystore. Optionally specify a name for the key.`,
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyStoreDir := filepath.Join(config.GetConfig().General.UserDir, dms.KeystoreDir)
			ks, err := keystore.New(fs, keyStoreDir)
			if err != nil {
				return fmt.Errorf("failed to create keystore: %w", err)
			}

			keyID := dms.KeyIDPrivKey
			if len(args) > 0 {
				keyID = args[0]
			}

			keys, err := ks.ListKeys()
			if err != nil {
				return fmt.Errorf("failed to list keys: %w", err)
			}

			if utils.SliceContains(keys, keyID) {
				confirmed, err := utils.PromptYesNo(
					cmd.InOrStdin(),
					cmd.OutOrStdout(),
					fmt.Sprintf("A key with name '%s' already exists. Do you want to overwrite it with a new one?", keyID),
				)
				if err != nil {
					return fmt.Errorf("failed to get user confirmation: %w", err)
				}
				if !confirmed {
					return fmt.Errorf("operation cancelled by user")
				}
			}

			passphrase, err := promptForPassphrase(true)
			if err != nil {
				return fmt.Errorf("failed to get passphrase: %w", err)
			}

			priv, err := dms.GenerateAndStorePrivKey(ks, passphrase, keyID)
			if err != nil {
				return fmt.Errorf("failed to generate and store new private key")
			}

			did := did.FromPublicKey(priv.GetPublic())
			fmt.Printf("New keypair generated and private key saved successfully with name '%s'\nDID: %s\n", keyID, did.String())
			return nil
		},
	}
}

func newKeyDIDCmd(fs afero.Afero) *cobra.Command {
	return &cobra.Command{
		Use:   "did <key-name>",
		Short: "Get DID for a key",
		Long:  `Get the DID (Decentralized Identifier) for a specified key`,
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			keyName := args[0]

			keyStoreDir := filepath.Join(config.GetConfig().General.UserDir, dms.KeystoreDir)
			ks, err := keystore.New(fs, keyStoreDir)
			if err != nil {
				return fmt.Errorf("failed to open keystore: %w", err)
			}

			passphrase, err := promptForPassphrase(false)
			if err != nil {
				return fmt.Errorf("failed to get passphrase: %w", err)
			}

			key, err := ks.Get(keyName, passphrase)
			if err != nil {
				return fmt.Errorf("failed to get key: %w", err)
			}

			priv, err := key.PrivKey()
			if err != nil {
				return fmt.Errorf("unable to convert key from keystore to private key: %v", err)
			}

			did := did.FromPublicKey(priv.GetPublic())
			fmt.Println(did.String())
			return nil
		},
	}
}
