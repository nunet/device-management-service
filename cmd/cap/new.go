// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cap

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func newNewCmd(afs afero.Afero) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new capability context",
		Long: `Create a new persistent capability context

Example:
  nunet cap new user
  nunet cap new ledger:user  # if using ledger`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			context := dms.UserContextName
			if len(args) > 0 {
				context = args[0]
			}

			var trustCtx did.TrustContext
			var rootDID did.DID
			if IsLedgerContext(context) {
				provider, err := did.NewLedgerWalletProvider(0)
				if err != nil {
					return err
				}

				trustCtx = did.NewTrustContextWithProvider(provider)
				rootDID = provider.DID()
				context = LedgerContext(context)
			} else {
				var priv crypto.PrivKey
				var err error
				trustCtx, priv, err = CreateTrustContextFromKeyStore(afs, context)
				if err != nil {
					return fmt.Errorf("failed to create trust context: %w", err)
				}
				rootDID = did.FromPublicKey(priv.GetPublic())
			}

			capStoreDir := filepath.Join(config.GetConfig().General.UserDir, dms.CapstoreDir)
			capStoreFile := filepath.Join(capStoreDir, fmt.Sprintf("%s.cap", context))

			fileExists, err := afs.Exists(capStoreFile)
			if err != nil {
				return fmt.Errorf("unable to check if capability context file exists: %w", err)
			}

			if fileExists {
				confirmed, err := utils.PromptYesNo(
					cmd.InOrStdin(),
					cmd.OutOrStdout(),
					fmt.Sprintf(
						"WARNING: A capability context file already exists at %s. Creating a new one will overwrite the existing context. Do you want to proceed?",
						capStoreFile,
					),
				)
				if err != nil {
					return fmt.Errorf("failed to get user confirmation: %w", err)
				}
				if !confirmed {
					return fmt.Errorf("operation cancelled by user")
				}
			} else {
				if err := afs.MkdirAll(capStoreDir, 0o700); err != nil {
					return fmt.Errorf("unable to create capability store directory: %w", err)
				}
			}

			capCtx, err := ucan.NewCapabilityContext(trustCtx, rootDID, nil, ucan.TokenList{}, ucan.TokenList{})
			if err != nil {
				return fmt.Errorf("unable to create capability context: %w", err)
			}

			if err := SaveCapabilityContext(capCtx, context); err != nil {
				return fmt.Errorf("save capability context: %w", err)
			}

			return nil
		},
	}

	return cmd
}
