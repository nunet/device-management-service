// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cap

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/internal/config"

	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
	dmsUtils "gitlab.com/nunet/device-management-service/utils"
)

type NewCapOptions struct {
	Force    bool
	Context  string
	PrismURL string
}

func newNewCmd(dmsCLI *cli.DmsCLI) *cobra.Command {
	var opts NewCapOptions

	cmd := &cobra.Command{
		Use:   "new <name>",
		Short: "Create a new capability context",
		Long: `Create a new persistent capability context

If the specified key has a PRISM DID association (created via 'nunet key create-prism'),
the capability context will use the PRISM DID. Otherwise, it will use a did:key identity.

Example:
  nunet cap new user
  nunet cap new myprism            # Uses PRISM DID if associated
  nunet cap new ledger             # ledger account 0
  nunet cap new ledger:3           # ledger account 3
  nunet cap new ledger:finance     # ledger alias "finance"`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			opts.Context = node.UserContextName
			if len(args) > 0 {
				opts.Context = args[0]
			}

			return runNewCap(cmd.Context(), dmsCLI, opts, cli.CmdStreams(cmd))
		},
	}

	cmd.Flags().BoolVarP(&opts.Force, fnForce, "f", false, "force overwrite of existing context")
	cmd.Flags().StringVar(&opts.PrismURL, "prism-url", "", "PRISM resolver URL (e.g., http://localhost:8080). If not specified, uses default resolver.")

	return cmd
}

func runNewCap(ctx context.Context, dmsCLI *cli.DmsCLI, opts NewCapOptions, streams cli.Streams) error {
	var passphrase string
	var privKey crypto.PrivKey

	fs := dmsCLI.FS()
	cfg, err := dmsCLI.Config()
	if err != nil {
		return fmt.Errorf("unable to get config: %w", err)
	}

	// ledger doesnt need keystore
	if node.IsLedgerContext(opts.Context) || node.IsEternlContext(opts.Context) {
		return GenCaps(ctx, cfg, fs, opts, streams, nil)
	}

	// set up keystore
	ks := dmsCLI.Keystore()
	if ks == nil {
		keyStoreDir := filepath.Join(cfg.General.UserDir, node.KeystoreDir)
		ks, err = keystore.New(fs, keyStoreDir, false)
		if err != nil {
			return fmt.Errorf("failed to open keystore: %w", err)
		}
	}

	// extract priv key
	if ks.Exists(opts.Context) {
		fmt.Fprintf(streams.Out, "Using identity at %s/%s.json...\n", ks.Dir(), opts.Context)
		passphrase, err = dmsCLI.Passphrase(opts.Context)
		if err != nil {
			return fmt.Errorf("failed to get passphrase: %w", err)
		}

		key, err := ks.Get(opts.Context, passphrase)
		if err != nil {
			return fmt.Errorf("failed to get key from keystore: %w", err)
		}

		privKey, err = key.PrivKey()
		if err != nil {
			return fmt.Errorf("unable to convert key from keystore to private key: %v", err)
		}
	} else {
		fmt.Fprintf(streams.Out, "A new identity will be created for '%s' context...\n", opts.Context)
		passphrase, err = dmsCLI.NewPassphrase(opts.Context)
		if err != nil {
			return fmt.Errorf("failed to create new passphrase: %w", err)
		}

		privKey, err = dms.GenerateAndStorePrivKey(ks, passphrase, opts.Context)
		if err != nil {
			return fmt.Errorf("failed to create new key: %w", err)
		}
	}
	if !ks.Exists(opts.Context) {
		return fmt.Errorf("key missing: %s", opts.Context)
	}

	return GenCaps(ctx, cfg, fs, opts, streams, privKey)
}

// GenCaps generates capability files.
func GenCaps(
	_ context.Context, cfg *config.Config, fs afero.Fs, opts NewCapOptions, streams cli.Streams, privKey crypto.PrivKey,
) error {
	var err error
	var trustCtx did.TrustContext
	var rootDID did.DID

	switch {
	case node.IsLedgerContext(opts.Context):
		// need userDir for the resolver
		idx, err := node.ResolveLedgerIndex(fs, cfg.General.UserDir, node.GetContextKey(opts.Context))
		if err != nil {
			return err
		}

		provider, err := did.NewLedgerWalletProvider(idx)
		if err != nil {
			return err
		}

		trustCtx = did.NewTrustContextWithProvider(provider)
		rootDID = provider.DID()
		opts.Context = node.GetContextKey(opts.Context) // normalize context name
	case node.IsEternlContext(opts.Context):

		provider, err := did.NewEternlWalletProvider()
		if err != nil {
			return err
		}

		trustCtx = did.NewTrustContextWithProvider(provider)
		rootDID = provider.DID()
		opts.Context = node.GetContextKey(opts.Context)
	default:
		// Check if this key has a PRISM DID association
		prismDIDStr, err := node.GetPrismDID(fs, cfg.General.UserDir, opts.Context)
		if err != nil {
			return fmt.Errorf("unable to check for PRISM DID association: %w", err)
		}

		if prismDIDStr != "" {
			// Use PRISM DID if available
			prismDID, err := did.FromString(prismDIDStr)
			if err != nil {
				return fmt.Errorf("invalid PRISM DID for key %s: %w", opts.Context, err)
			}

			// Create PRISM provider
			provider, err := did.ProviderFromPRISMPrivateKey(prismDID, privKey)
			if err != nil {
				return fmt.Errorf("unable to create PRISM provider: %w", err)
			}

			// Create trust context with PRISM provider
			trustCtx = did.NewTrustContextWithProvider(provider)

			// Configure PRISM resolver if URL is provided
			originalConfig := did.GetPRISMResolverConfig()
			if opts.PrismURL != "" {
				did.SetPRISMResolverConfig(did.PRISMResolverConfig{
					ResolverURL:                 opts.PrismURL,
					PreferredVerificationMethod: originalConfig.PreferredVerificationMethod,
					HTTPClient:                  originalConfig.HTTPClient,
				})
			}

			// Try to resolve PRISM DID to get anchor for verification
			// This is optional - if resolution fails, we'll continue without the anchor
			// The anchor will be resolved on-demand when needed for verification
			anchor, err := did.GetAnchorForDID(prismDID)

			// Restore original config if we changed it
			if opts.PrismURL != "" {
				did.SetPRISMResolverConfig(originalConfig)
			}

			if err != nil {
				fmt.Fprintf(streams.Out, "⚠️  Warning: Could not resolve PRISM DID anchor (will resolve on-demand): %v\n", err)
				fmt.Fprintf(streams.Out, "   This is normal if the DID was just created or the resolver is unavailable.\n")
			} else {
				// Add PRISM anchor to trust context for faster verification
				trustCtx.AddAnchor(anchor)
			}

			// Use PRISM DID as root
			rootDID = prismDID

			fmt.Fprintf(streams.Out, "✅ Using PRISM DID: %s\n", prismDIDStr)
			if opts.PrismURL != "" {
				fmt.Fprintf(streams.Out, "   Resolver URL: %s\n", opts.PrismURL)
			}
		} else {
			// Fall back to did:key if no PRISM DID association
			trustCtx, err = did.NewTrustContextWithPrivateKey(privKey)
			if err != nil {
				return fmt.Errorf("unable to create trust context: %w", err)
			}

			rootDID = did.FromPublicKey(privKey.GetPublic())
			fmt.Fprintf(streams.Out, "Using did:key identity (no PRISM DID association found)\n")
		}
	}

	capStoreDir := filepath.Join(cfg.General.UserDir, node.CapstoreDir)
	capStoreFile := filepath.Join(capStoreDir, fmt.Sprintf("%s.cap", opts.Context))

	fileExists, err := afero.Exists(fs, capStoreFile)
	if err != nil {
		return fmt.Errorf("unable to check if capability context file exists: %w", err)
	}

	if fileExists && !opts.Force {
		confirmed, err := dmsUtils.PromptYesNo(
			streams.In,
			streams.Out,
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
		if err := fs.MkdirAll(capStoreDir, 0o700); err != nil {
			return fmt.Errorf("unable to create capability store directory: %w", err)
		}
	}

	capCtx, err := ucan.NewCapabilityContextWithName(opts.Context, trustCtx, rootDID, nil, ucan.TokenList{}, ucan.TokenList{}, ucan.TokenList{})
	if err != nil {
		return fmt.Errorf("unable to create capability context: %w", err)
	}

	if err := node.SaveCapabilityContext(capCtx, fs, cfg.UserDir); err != nil {
		return fmt.Errorf("save capability context: %w", err)
	}

	return nil
}
