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
	"os"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	dmsUtils "gitlab.com/nunet/device-management-service/utils"
)

func newKeyCmd(fs afero.Afero, cfg *config.Config) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage cryptographic keys",
		Long: `Manage cryptographic keys for the Device Management Service (DMS).

This command provides subcommands for creating new keys and retrieving Decentralized Identifiers (DIDs) associated with existing keys.`,
	}

	cmd.AddCommand(newKeyNewCmd(fs, cfg))
	cmd.AddCommand(newKeyDIDCmd(fs, cfg))

	return cmd
}

func newKeyNewCmd(fs afero.Afero, cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "new <name>",
		Short: "Generate a key pair",
		Long: `Generate a key pair and save the private key into the user's local keystore.

This command creates a new cryptographic key pair, stores the private key securely, and displays the associated Decentralized Identifier (DID). If a key with the specified name already exists, the user will be prompted to confirm before overwriting it.

Example:
  nunet key new user`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyStoreDir := filepath.Join(cfg.General.UserDir, dms.KeystoreDir)
			ks, err := keystore.New(fs, keyStoreDir)
			if err != nil {
				return fmt.Errorf("failed to create keystore: %w", err)
			}

			keyID := dms.UserContextName
			if len(args) > 0 {
				keyID = args[0]
			}

			keys, err := ks.ListKeys()
			if err != nil {
				return fmt.Errorf("failed to list keys: %w", err)
			}

			if dmsUtils.SliceContains(keys, keyID) {
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

			passphrase := os.Getenv("DMS_PASSPHRASE")
			if passphrase == "" {
				passphrase, err = utils.PromptForPassphrase(true)
				if err != nil {
					return fmt.Errorf("failed to get passphrase: %w", err)
				}
			}

			priv, err := dms.GenerateAndStorePrivKey(ks, passphrase, keyID)
			if err != nil {
				return fmt.Errorf("failed to generate and store new private key")
			}

			did := did.FromPublicKey(priv.GetPublic())
			fmt.Println(did)
			return nil
		},
	}
}

func newKeyDIDCmd(fs afero.Afero, cfg *config.Config) *cobra.Command {
	return &cobra.Command{
		Use:   "did <name>",
		Short: "Get a key's DID",
		Long: `Get the DID (Decentralized Identifier) for a specified key.

This command retrieves the DID associated with either a key stored in the local keystore or a hardware ledger.

For keys in the local keystore, the user will be prompted for the passphrase to decrypt the key. To avoid passphrase prompting, it's possible to set a DMS_PASSPHRASE environment variable. For the ledger option, it uses the first account (index 0) on the connected hardware wallet.

Example:
  nunet key did user
  nunet key did ledger # if using ledger`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			keyName := args[0]

			if keyName == "ledger" {
				provider, err := did.NewLedgerWalletProvider(0)
				if err != nil {
					return err
				}

				fmt.Println(provider.DID())
				return nil
			}

			keyStoreDir := filepath.Join(cfg.General.UserDir, dms.KeystoreDir)
			ks, err := keystore.New(fs, keyStoreDir)
			if err != nil {
				return fmt.Errorf("failed to open keystore: %w", err)
			}

			passphrase := os.Getenv("DMS_PASSPHRASE")
			if passphrase == "" {
				passphrase, err = utils.PromptForPassphrase(false)
				if err != nil {
					return fmt.Errorf("failed to get passphrase: %w", err)
				}
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
			fmt.Println(did)
			return nil
		},
	}
}
