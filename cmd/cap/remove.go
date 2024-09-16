package cap

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func newRemoveCmd(afs afero.Afero) *cobra.Command {
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
		Use:   "remove",
		Short: "Remove capability anchors",
		Long:  `Remove capability anchors in a capability context`,
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

			switch {
			case root != "":
				rootDID, err := did.FromString(root)
				if err != nil {
					return fmt.Errorf("invalid root DID: %w", err)
				}

				capCtx.RemoveRoots([]did.DID{rootDID}, ucan.TokenList{}, ucan.TokenList{})

			case require != "":
				var token ucan.Token
				if err := json.Unmarshal([]byte(require), &token); err != nil {
					return fmt.Errorf("unmarshal tokens: %w", err)
				}

				capCtx.RemoveRoots(nil, ucan.TokenList{Tokens: []*ucan.Token{&token}}, ucan.TokenList{})

			case provide != "":
				var token ucan.Token
				if err := json.Unmarshal([]byte(provide), &token); err != nil {
					return fmt.Errorf("unmarshal tokens: %w", err)
				}

				capCtx.RemoveRoots(nil, ucan.TokenList{}, ucan.TokenList{Tokens: []*ucan.Token{&token}})

			default:
				return fmt.Errorf("one of --provide, --root, or --require must be specified")
			}

			if err := SaveCapabilityContext(capCtx, context); err != nil {
				return fmt.Errorf("save capability context: %w", err)
			}

			return nil
		},
	}

	useFlagContext(cmd, &context)
	cmd.Flags().StringVar(&root, fnRoot, "", "DID to add as root anchor")
	cmd.Flags().StringVar(&provide, fnProvide, "", "Tokens to add as provide anchor (in JSON format)")
	cmd.Flags().StringVar(&require, fnRequire, "", "Tokens to add as require anchor (in JSON format)")

	_ = cmd.MarkFlagRequired(fnContext)
	cmd.MarkFlagsOneRequired(fnProvide, fnRoot, fnRequire)
	cmd.MarkFlagsMutuallyExclusive(fnProvide, fnRoot, fnRequire)

	return cmd
}
