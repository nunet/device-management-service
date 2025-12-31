// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	dmsUtils "gitlab.com/nunet/device-management-service/utils"
)

func newKeyCmd(
	dmsCli *cli.DmsCLI,
) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "key",
		Short: "Manage cryptographic keys",
		Long: `Manage cryptographic keys for the Device Management Service (DMS).

This command provides subcommands for creating new keys and retrieving Decentralized Identifiers (DIDs) associated with existing keys.`,
	}

	cmd.AddCommand(newKeyNewCmd(dmsCli))
	cmd.AddCommand(newKeyImportCmd(dmsCli))
	cmd.AddCommand(newKeyDIDCmd(dmsCli))
	cmd.AddCommand(newKeyLedgerAliasCmd(dmsCli))

	return cmd
}

func newKeyNewCmd(
	dmsCli *cli.DmsCLI,
) *cobra.Command {
	return &cobra.Command{
		Use:   "new <name>",
		Short: "Generate a key pair",
		Long: `Generate a key pair and save the private key into the user's local keystore.

This command creates a new cryptographic key pair, stores the private key securely, and displays the associated Decentralized Identifier (DID). If a key with the specified name already exists, the user will be prompted to confirm before overwriting it.

Example:
  nunet key new user`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := dmsCli.Config()
			if err != nil {
				return fmt.Errorf("get dms config: %w", err)
			}
			fs := dmsCli.FS()

			keyStoreDir := filepath.Join(cfg.General.UserDir, node.KeystoreDir)
			ks, err := keystore.New(fs, keyStoreDir, false)
			if err != nil {
				return fmt.Errorf("failed to create keystore: %w", err)
			}

			keyID := node.UserContextName
			if len(args) > 0 {
				keyID = args[0]
			}

			if ks.Exists(keyID) {
				confirmed, err := dmsUtils.PromptYesNo(
					cmd.InOrStdin(),
					cmd.OutOrStdout(),
					fmt.Sprintf("A key with name '%s' already exists. Do you want to overwrite it with a new one?", keyID),
				)
				if err != nil {
					return fmt.Errorf("failed to get user confirmation: %w", err)
				}
				if !confirmed {
					return dmsUtils.ErrOperationCancelled
				}
			}

			passphrase, err := dmsCli.NewPassphrase(keyID)
			if err != nil {
				return fmt.Errorf("get dms passphrase: %w", err)
			}

			priv, err := dms.GenerateAndStorePrivKey(ks, passphrase, keyID)
			if err != nil {
				return fmt.Errorf("failed to generate and store new private key: %w", err)
			}

			did := did.FromPublicKey(priv.GetPublic())
			fmt.Fprintln(cmd.OutOrStdout(), did)
			return nil
		},
	}
}

func newKeyImportCmd(
	dmsCli *cli.DmsCLI,
) *cobra.Command {
	return &cobra.Command{
		Use:   "import <name> <private-key-hex>",
		Short: "Import a private key",
		Long: `Import an existing private key into the user's local keystore.

This command takes a hex-encoded private key (in libp2p protobuf format or raw Ed25519 seed) and stores it securely with the given name.
If a key with the specified name already exists, the user will be prompted to confirm before overwriting it.

Example:
  nunet key import myweb3key 08011240...`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyID := args[0]
			hexKey := args[1]

			rawPriv, err := hex.DecodeString(hexKey)
			if err != nil {
				return fmt.Errorf("invalid hex string: %w", err)
			}

			cfg, err := dmsCli.Config()
			if err != nil {
				return fmt.Errorf("get dms config: %w", err)
			}
			fs := dmsCli.FS()

			keyStoreDir := filepath.Join(cfg.General.UserDir, node.KeystoreDir)
			ks, err := keystore.New(fs, keyStoreDir, false)
			if err != nil {
				return fmt.Errorf("failed to create keystore: %w", err)
			}

			if ks.Exists(keyID) {
				confirmed, err := dmsUtils.PromptYesNo(
					cmd.InOrStdin(),
					cmd.OutOrStdout(),
					fmt.Sprintf("A key with name '%s' already exists. Do you want to overwrite it?", keyID),
				)
				if err != nil {
					return fmt.Errorf("failed to get user confirmation: %w", err)
				}
				if !confirmed {
					return dmsUtils.ErrOperationCancelled
				}
			}

			passphrase, err := dmsCli.NewPassphrase(keyID)
			if err != nil {
				return fmt.Errorf("get dms passphrase: %w", err)
			}

			priv, err := dms.ImportAndStorePrivKey(ks, rawPriv, passphrase, keyID)
			if err != nil {
				return fmt.Errorf("failed to import and store private key: %w", err)
			}

			did := did.FromPublicKey(priv.GetPublic())
			fmt.Fprintln(cmd.OutOrStdout(), did)
			return nil
		},
	}
}

func newKeyDIDCmd(
	dmsCli *cli.DmsCLI,
) *cobra.Command {
	return &cobra.Command{
		Use:   "did <name>",
		Short: "Get a key's DID",
		Long: `Get the DID (Decentralized Identifier) for a specified key.

This command retrieves the DID associated with either a key stored in the local keystore
or a hardware ledger.  For the ledger you can now supply an account index or a named alias.

Examples:
  nunet key did user                 # key from keystore
  nunet key did ledger               # ledger account 0 (default)
  nunet key did ledger:3             # ledger account 3
  nunet key did ledger:business      # ledger alias "business"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := dmsCli.ConfigLoader().Load()
			if err != nil {
				return fmt.Errorf("get dms config: %w", err)
			}
			fs := dmsCli.FS()
			env := dmsCli.Env()

			keyName := args[0]
			// Ledger branch
			if node.IsLedgerContext(keyName) {
				idx, err := node.ResolveLedgerIndex(
					fs, cfg.General.UserDir, node.GetContextKey(keyName),
				)
				if err != nil {
					return err
				}

				provider, err := did.NewLedgerWalletProvider(idx)
				if err != nil {
					return err
				}

				fmt.Println(provider.DID())
				return nil
			}

			if node.IsEternlContext(keyName) {
				provider, err := did.NewEternlWalletProvider()
				if err != nil {
					return err
				}

				fmt.Println(provider.DID())
				return nil
			}

			keyStoreDir := filepath.Join(cfg.General.UserDir, node.KeystoreDir)
			ks, err := keystore.New(fs, keyStoreDir, false)
			if err != nil {
				return fmt.Errorf("failed to open keystore: %w", err)
			}

			passphrase, err := utils.GetDMSPassphrase(env, false)
			if err != nil {
				return fmt.Errorf("get dms passphrase: %w", err)
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
			fmt.Fprintln(cmd.OutOrStdout(), did)
			return nil
		},
	}
}

func newKeyLedgerAliasCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ledger-alias",
		Short: "Manage aliases for Ledger accounts",
	}
	cmd.AddCommand(newKeyLedgerAliasSetCmd(dmsCli))
	return cmd
}

// Child: `nunet key ledger-alias set <alias> <index>`
func newKeyLedgerAliasSetCmd(dmsCli *cli.DmsCLI) *cobra.Command {
	return &cobra.Command{
		Use:   "set <alias> <index>",
		Short: "Create or update a Ledger alias",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			alias := args[0]
			idx, err := strconv.Atoi(args[1])
			if err != nil || idx < 0 {
				return fmt.Errorf("index must be a non-negative integer")
			}

			cfg, err := dmsCli.ConfigLoader().Load()
			if err != nil {
				return fmt.Errorf("get dms config: %w", err)
			}

			if err := node.SetLedgerAlias(dmsCli.FS(), cfg.General.UserDir, alias, idx); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(),
				"Alias %q → account %d saved\n",
				alias, idx)
			return nil
		},
	}
}
