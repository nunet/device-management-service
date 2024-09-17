package cap

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

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
		Long: `Add or modify capability anchors in a capability context.

An anchor is a basis of trust in the capability system. There are three types of anchors:

1. Root anchor: Represents absolute trust or effective root capability.
   Use the --root flag with a DID value to add a root anchor.

2. Require anchor: Represents input trust. We verify incoming messages based on the require anchor.
   Use the --require flag with a token to add a require anchor.

3. Provide anchor: Represents output trust. We emit invocation tokens based on our provide anchors and sign output.
   Use the --provide flag with a token to add a provide anchor.

Only one type of anchor can be added or modified per command execution.

Usage examples:
  nunet cap anchor --context user --root did:example:123456789abcdefghi
  nunet cap anchor --context dms  --require '{"some": "json", "token": "here"}'
  nunet cap anchor --context user --provide '{"another": "json", "token": "example"}'

Note: The --context flag is required to specify the capability context.`,
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

	useFlagContext(cmd, &context)
	useFlagRoot(cmd, &root)
	useFlagRequire(cmd, &require)
	useFlagProvide(cmd, &provide)

	_ = cmd.MarkFlagRequired(fnContext)
	cmd.MarkFlagsOneRequired(fnProvide, fnRoot, fnRequire)
	cmd.MarkFlagsMutuallyExclusive(fnProvide, fnRoot, fnRequire)

	return cmd
}
