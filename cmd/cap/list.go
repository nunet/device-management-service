package cap

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/lib/did"
)

func newListCmd(afs afero.Afero) *cobra.Command {
	var context string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List capability anchors",
		Long:  "List all capability anchors in a capability context",
		RunE: func(_ *cobra.Command, _ []string) error {
			var trustCtx did.TrustContext
			if IsLedgerContext(context) {
				provider, err := did.NewLedgerWalletProvider(0)
				if err != nil {
					return err
				}

				trustCtx = did.NewTrustContextWithProvider(provider)
				context = LedgerContext(context)
			} else {
				var err error
				trustCtx, _, err = CreateTrustContextFromKeyStore(afs, context)
				if err != nil {
					return fmt.Errorf("failed to create trust context: %w", err)
				}
			}

			capCtx, err := LoadCapabilityContext(trustCtx, context)
			if err != nil {
				return fmt.Errorf("failed to load capability context: %w", err)
			}

			roots, require, provide := capCtx.ListRoots()

			fmt.Println("roots:")
			for _, root := range roots {
				fmt.Printf("\t%s\n", root)
			}

			fmt.Println("require:")
			for _, t := range require.Tokens {
				data, err := json.Marshal(t)
				if err != nil {
					return fmt.Errorf("failed to marshal capability token: %w", err)
				}
				fmt.Printf("\t%s\n", string(data))
			}

			fmt.Println("provide:")
			for _, t := range provide.Tokens {
				data, err := json.Marshal(t)
				if err != nil {
					return fmt.Errorf("failed to marshal capability token: %w", err)
				}
				fmt.Printf("\t%s\n", string(data))
			}

			return nil
		},
	}

	useFlagContext(cmd, &context)

	return cmd
}
