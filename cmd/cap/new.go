package cap

import (
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func newNewCmd(afs afero.Afero) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new capability context",
		Long:  `Create a new persistent capability context for DMS or personal usage`,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			context := dms.UserContextName
			if len(args) > 0 {
				context = args[0]
			}

			trustCtx, priv, err := CreateTrustContextFromKeyStore(afs, context)
			if err != nil {
				return fmt.Errorf("failed to create trust context: %w", err)
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

			rootDID := did.FromPublicKey(priv.GetPublic())
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
