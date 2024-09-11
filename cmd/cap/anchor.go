package cap

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func newAnchorCmd(afs afero.Afero) *cobra.Command {
	var (
		context string
		root    string
		provide string
		require string
	)

	const (
		fnProvide = "provide"
		fnRoot    = "root"
		fnRequire = "require"
	)

	cmd := &cobra.Command{
		Use:   "anchor",
		Short: "Manage capability anchors",
		Long:  `Add or modify capability anchors in a capability context`,
		RunE: func(_ *cobra.Command, _ []string) error {
			trustCtx, _, err := CreateTrustContextFromKeyStore(afs, context)
			if err != nil {
				return fmt.Errorf("failed to create trust context: %w", err)
			}

			capCtx, err := LoadCapabilityContext(trustCtx, context)
			if err != nil {
				return fmt.Errorf("failed to load capability context: %w", err)
			}

			switch {
			case root != "":
				rootDID, err := did.FromString(root)
				if err != nil {
					return fmt.Errorf("invalid root DID: %w", err)
				}

				if err := capCtx.AddRoots([]did.DID{rootDID}, ucan.TokenList{}, ucan.TokenList{}); err != nil {
					return fmt.Errorf("failed to add root anchors: %w", err)
				}

			case require != "":
				var tokens ucan.TokenList
				if err := json.Unmarshal([]byte(require), &tokens); err != nil {
					return fmt.Errorf("unmarshal tokens: %w", err)
				}

				if err := capCtx.AddRoots(nil, tokens, ucan.TokenList{}); err != nil {
					return fmt.Errorf("failed to add require anchors: %w", err)
				}

			case provide != "":
				var tokens ucan.TokenList
				if err := json.Unmarshal([]byte(provide), &tokens); err != nil {
					return fmt.Errorf("unmarshal tokens: %w", err)
				}

				if err := capCtx.AddRoots(nil, ucan.TokenList{}, tokens); err != nil {
					return fmt.Errorf("failed to add provide anchors: %w", err)
				}

			default:
				return fmt.Errorf("one of --provide, --root, or --require must be specified")
			}

			if err := SaveCapabilityContext(capCtx, context); err != nil {
				return fmt.Errorf("save capability context: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&context, flagContext, dms.KeyIDPrivKey, "Operation context; it specifies the key and capability context to use; defaults to dms context")
	cmd.Flags().StringVar(&root, flagRoot, "", "DID to add as root anchor")
	cmd.Flags().StringVar(&provide, flagProvide, "", "Tokens to add as provide anchor (in JSON format)")
	cmd.Flags().StringVar(&require, flagRequire, "", "Tokens to add as require anchor (in JSON format)")

	cmd.MarkFlagsOneRequired(fnProvide, fnRoot, fnRequire)
	cmd.MarkFlagsMutuallyExclusive(fnProvide, fnRoot, fnRequire)

	return cmd
}
