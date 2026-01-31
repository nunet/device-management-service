// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"google.golang.org/protobuf/proto"

	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	prismpb "gitlab.com/nunet/device-management-service/proto/generated/prism"
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
	cmd.AddCommand(newKeyImportPrismCmd(dmsCli))
	cmd.AddCommand(newKeyCreatePrismCmd(dmsCli))
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

func newKeyImportPrismCmd(
	dmsCli *cli.DmsCLI,
) *cobra.Command {
	return &cobra.Command{
		Use:   "import-prism <name> <prism-did> <private-key-hex>",
		Short: "Import a PRISM identity (DID + private key)",
		Long: `Import an existing PRISM identity into the user's local keystore.

This command takes a PRISM DID and a hex-encoded private key (in libp2p protobuf format or raw Ed25519/secp256k1 key) 
and stores it securely with the given name. The private key will be associated with the PRISM DID for signing UCAN tokens.

Supported key formats:
  - libp2p protobuf format (hex encoded)
  - Raw Ed25519 seed (32 bytes, hex encoded)
  - Raw Ed25519 private key (64 bytes, hex encoded)
  - Raw secp256k1 private key (32 bytes, hex encoded)

If a key with the specified name already exists, the user will be prompted to confirm before overwriting it.

Example:
  nunet key import-prism myprism did:prism:9b5118411248d9663b6ab15128fba8106511230ff654e7514cdcc4ce919bde9b 08011240...`,
		Args: cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyID := args[0]
			prismDIDStr := args[1]
			hexKey := args[2]

			// Parse PRISM DID
			prismDID, err := did.FromString(prismDIDStr)
			if err != nil {
				return fmt.Errorf("invalid PRISM DID: %w", err)
			}

			if prismDID.Method() != "prism" {
				return fmt.Errorf("expected PRISM DID (did:prism:...), got %s", prismDID.Method())
			}

			// Decode private key
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

			// Import and store the private key (same as regular import)
			priv, err := dms.ImportAndStorePrivKey(ks, rawPriv, passphrase, keyID)
			if err != nil {
				return fmt.Errorf("failed to import and store private key: %w", err)
			}

			// Verify we can create a PRISM provider
			provider, err := did.ProviderFromPRISMPrivateKey(prismDID, priv)
			if err != nil {
				return fmt.Errorf("failed to create PRISM provider: %w", err)
			}

			// Store the PRISM DID association
			if err := node.SetPrismDID(fs, cfg.General.UserDir, keyID, prismDIDStr); err != nil {
				return fmt.Errorf("failed to store PRISM DID association: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "PRISM identity imported successfully\n")
			fmt.Fprintf(cmd.OutOrStdout(), "DID: %s\n", provider.DID())
			fmt.Fprintf(cmd.OutOrStdout(), "Key name: %s\n", keyID)
			fmt.Fprintf(cmd.OutOrStdout(), "\nThe PRISM DID association has been stored.\n")
			fmt.Fprintf(cmd.OutOrStdout(), "This key will be used with the PRISM DID when signing UCAN tokens.\n")

			return nil
		},
	}
}

func newKeyCreatePrismCmd(
	dmsCli *cli.DmsCLI,
) *cobra.Command {
	var (
		prismURL          string
		keyType           string
		waitConfirmation  bool
		timeout           string
		submissionTimeout string
	)

	cmd := &cobra.Command{
		Use:   "create-prism <name>",
		Short: "Create a new PRISM identity",
		Long: `Create a new PRISM identity by generating keys, creating a PRISM DID operation,
submitting it to the PRISM network via NeoPRISM, and importing it into DMS.

This command automates the complete PRISM identity creation workflow:
1. Generates cryptographic keys locally (Secp256k1 or Ed25519)
2. Creates a PRISM DID operation with the generated keys
3. Submits the operation to NeoPRISM (which handles blockchain transaction fees)
4. Optionally waits for blockchain confirmation
5. Imports the created identity into DMS

The command returns all necessary credentials (DID, public/private keys) and sets up
the identity for use with UCAN capabilities.

Note: PRISM identities are independent of Cardano wallets. NeoPRISM handles wallet
management and transaction fees when submitting operations to the blockchain.

Example:
  nunet key create-prism myprism`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyID := args[0]

			// Validate key type
			if keyType != "secp256k1" && keyType != "ed25519" {
				return fmt.Errorf("invalid key type: %s (supported: secp256k1, ed25519)", keyType)
			}

			// Parse timeout
			timeoutDuration, err := time.ParseDuration(timeout)
			if err != nil {
				return fmt.Errorf("invalid timeout duration: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "🔑 Generating %s key pair...\n", keyType)
			// Generate keys
			privKey, pubKey, err := generatePRISMKeys(keyType)
			if err != nil {
				return fmt.Errorf("generate keys: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✅ Key pair generated successfully\n\n")

			fmt.Fprintf(cmd.OutOrStdout(), "📝 Creating PRISM DID operation...\n")
			// Create PRISM operation
			signedOpHex, err := did.CreateSignedPRISMOperationSimple(privKey, pubKey, "master-0")
			if err != nil {
				return fmt.Errorf("create PRISM operation: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✅ PRISM operation created\n\n")

			fmt.Fprintf(cmd.OutOrStdout(), "🔍 Extracting DID from operation...\n")
			// Extract DID from operation
			prismDIDStr, err := extractDIDFromSignedOperation(signedOpHex)
			if err != nil {
				return fmt.Errorf("extract DID from operation: %w", err)
			}

			// Parse PRISM DID
			prismDID, err := did.FromString(prismDIDStr)
			if err != nil {
				return fmt.Errorf("parse PRISM DID: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✅ DID extracted: %s\n\n", prismDIDStr)

			// Parse submission timeout
			submissionTimeoutDuration := 2 * time.Minute // Default 2 minutes (should be fast if working)
			if submissionTimeout != "" {
				parsedTimeout, err := time.ParseDuration(submissionTimeout)
				if err != nil {
					return fmt.Errorf("invalid submission timeout duration: %w", err)
				}
				submissionTimeoutDuration = parsedTimeout
			}

			// Quick connectivity check
			fmt.Fprintf(cmd.OutOrStdout(), "🔍 Checking NeoPRISM connectivity at %s...\n", prismURL)
			if err := checkNeoPRISMConnectivity(prismURL, 5*time.Second); err != nil {
				return fmt.Errorf("NeoPRISM connectivity check failed: %w\n\nTroubleshooting:\n- Ensure NeoPRISM is running: docker ps | grep neoprism\n- Check NeoPRISM is accessible at %s\n- Verify NeoPRISM logs: docker logs neoprism --tail 50", err, prismURL)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✅ NeoPRISM is reachable\n\n")

			fmt.Fprintf(cmd.OutOrStdout(), "📤 Submitting operation to NeoPRISM (timeout: %s)...\n", submissionTimeoutDuration)
			// Submit to NeoPRISM
			txID, operationIDs, err := submitPRISMOperationToNeoPRISM(prismURL, signedOpHex, submissionTimeoutDuration, cmd.OutOrStdout())
			if err != nil {
				return fmt.Errorf("submit to NeoPRISM: %w\n\nTroubleshooting:\n- Ensure NeoPRISM is running at %s\n- Check NeoPRISM logs for errors\n- Verify NeoPRISM has sufficient funds for transaction fees\n- Try increasing --submission-timeout if the operation is large", err, prismURL)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✅ Operation submitted successfully\n\n")

			// Print PRISM identity information
			fmt.Fprintf(cmd.OutOrStdout(), "📋 PRISM Identity Information:\n")
			fmt.Fprintf(cmd.OutOrStdout(), "   DID: %s\n", prismDIDStr)
			fmt.Fprintf(cmd.OutOrStdout(), "   Key Name: %s\n", keyID)
			fmt.Fprintf(cmd.OutOrStdout(), "   Key Type: %s\n", keyType)
			if txID != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "   Transaction ID: %s\n", txID)
			}
			if len(operationIDs) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "   Operation IDs: %s\n", strings.Join(operationIDs, ", "))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n")

			// Optionally wait for confirmation
			if waitConfirmation {
				fmt.Fprintf(cmd.OutOrStdout(), "⏳ Waiting for blockchain confirmation (timeout: %s)...\n", timeout)
				ctx, cancel := context.WithTimeout(context.Background(), timeoutDuration)
				defer cancel()

				err = waitForDIDDocument(ctx, prismDID, prismURL, timeoutDuration, cmd.OutOrStdout())
				if err != nil {
					return fmt.Errorf("wait for DID document: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "✅ DID document confirmed on blockchain\n\n")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "⏭️  Skipping confirmation wait (use --wait-confirmation to enable)\n\n")
			}

			// Get config and file system
			cfg, err := dmsCli.Config()
			if err != nil {
				return fmt.Errorf("get dms config: %w", err)
			}
			fs := dmsCli.FS()

			// Setup keystore
			keyStoreDir := filepath.Join(cfg.General.UserDir, node.KeystoreDir)
			ks, err := keystore.New(fs, keyStoreDir, false)
			if err != nil {
				return fmt.Errorf("failed to create keystore: %w", err)
			}

			// Check if key exists
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

			// Marshal private key for storage
			rawPriv, err := crypto.PrivateKeyToBytes(privKey)
			if err != nil {
				return fmt.Errorf("marshal private key: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "💾 Importing identity into DMS...\n")
			// Get passphrase
			passphrase, err := dmsCli.NewPassphrase(keyID)
			if err != nil {
				return fmt.Errorf("get dms passphrase: %w", err)
			}

			// Import and store the private key
			priv, err := dms.ImportAndStorePrivKey(ks, rawPriv, passphrase, keyID)
			if err != nil {
				return fmt.Errorf("failed to import and store private key: %w", err)
			}

			// Verify we can create a PRISM provider
			provider, err := did.ProviderFromPRISMPrivateKey(prismDID, priv)
			if err != nil {
				return fmt.Errorf("failed to create PRISM provider: %w", err)
			}

			// Store the PRISM DID association
			if err := node.SetPrismDID(fs, cfg.General.UserDir, keyID, prismDIDStr); err != nil {
				return fmt.Errorf("failed to store PRISM DID association: %w", err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "✅ Identity imported successfully\n\n")

			// Output results
			fmt.Fprintf(cmd.OutOrStdout(), "✅ PRISM identity created successfully\n\n")
			fmt.Fprintf(cmd.OutOrStdout(), "DID: %s\n", provider.DID())
			fmt.Fprintf(cmd.OutOrStdout(), "Key name: %s\n", keyID)
			if txID != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "Transaction ID: %s\n", txID)
			}
			if len(operationIDs) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Operation IDs: %s\n", strings.Join(operationIDs, ", "))
			}
			fmt.Fprintf(cmd.OutOrStdout(), "\n⚠️  IMPORTANT: Your private key is stored in the keystore. Make sure to back up\n")
			fmt.Fprintf(cmd.OutOrStdout(), "   your keystore directory if you need to recover this identity.\n\n")
			fmt.Fprintf(cmd.OutOrStdout(), "The identity has been imported into DMS and is ready to use with UCAN.\n")

			return nil
		},
	}

	cmd.Flags().StringVar(&prismURL, "prism-url", "http://localhost:8080", "PRISM resolver/submitter URL")
	cmd.Flags().StringVar(&keyType, "key-type", "secp256k1", "Key type for PRISM operation (ed25519 or secp256k1)")
	cmd.Flags().BoolVar(&waitConfirmation, "wait-confirmation", true, "Wait for blockchain confirmation before returning")
	cmd.Flags().StringVar(&timeout, "timeout", "20m", "Timeout for blockchain confirmation")
	cmd.Flags().StringVar(&submissionTimeout, "submission-timeout", "2m", "Timeout for submitting operation to NeoPRISM")

	return cmd
}

// generatePRISMKeys generates a key pair for PRISM identity creation
func generatePRISMKeys(keyType string) (crypto.PrivKey, crypto.PubKey, error) {
	var keyTypeEnum int
	switch keyType {
	case "secp256k1":
		keyTypeEnum = crypto.Secp256k1
	case "ed25519":
		keyTypeEnum = crypto.Ed25519
	default:
		return nil, nil, fmt.Errorf("unsupported key type: %s", keyType)
	}

	privKey, pubKey, err := crypto.GenerateKeyPair(keyTypeEnum)
	if err != nil {
		return nil, nil, fmt.Errorf("generate key pair: %w", err)
	}

	return privKey, pubKey, nil
}

// checkNeoPRISMConnectivity performs a quick connectivity check to NeoPRISM
func checkNeoPRISMConnectivity(neoprismURL string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Try to reach a simple endpoint (health check or similar)
	// If no health endpoint, try the submission endpoint with invalid data to see if it responds
	testURL := fmt.Sprintf("%s/api/signed-operation-submissions", strings.TrimSuffix(neoprismURL, "/"))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, testURL, bytes.NewReader([]byte(`{"signed_operations":["test"]}`)))
	if err != nil {
		return fmt.Errorf("create test request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{
		Timeout: timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return fmt.Errorf("connection timeout - NeoPRISM may be unreachable or slow to respond")
		}
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	// Any response (even error) means NeoPRISM is reachable
	// 422 is expected for invalid test data, which confirms the endpoint works
	if resp.StatusCode == http.StatusUnprocessableEntity || resp.StatusCode == http.StatusOK {
		return nil
	}

	// Other status codes might indicate issues, but at least it's reachable
	return nil
}

// extractDIDFromSignedOperation extracts the PRISM DID from a signed operation
// The DID suffix is the hexadecimal-encoded SHA256 hash of the operation bytes
func extractDIDFromSignedOperation(signedOpHex string) (string, error) {
	// Decode hex
	signedOpBytes, err := hex.DecodeString(signedOpHex)
	if err != nil {
		return "", fmt.Errorf("decode hex: %w", err)
	}

	// Parse the SignedPrismOperation to get the operation
	var signedOp prismpb.SignedPrismOperation
	if err := proto.Unmarshal(signedOpBytes, &signedOp); err != nil {
		return "", fmt.Errorf("unmarshal signed operation: %w", err)
	}

	if signedOp.Operation == nil {
		return "", fmt.Errorf("operation is nil")
	}

	// Encode the operation to bytes
	operationBytes, err := proto.Marshal(signedOp.Operation)
	if err != nil {
		return "", fmt.Errorf("marshal operation: %w", err)
	}

	// Compute SHA256 hash
	hash := sha256.Sum256(operationBytes)

	// NeoPRISM expects canonical PRISM DIDs with hexadecimal suffix (64 chars)
	// Format: did:prism:{64-char-hex}
	didSuffix := hex.EncodeToString(hash[:])
	return fmt.Sprintf("did:prism:%s", didSuffix), nil
}

// submitPRISMOperationToNeoPRISM submits a signed PRISM operation to NeoPRISM
func submitPRISMOperationToNeoPRISM(neoprismURL string, signedOpHex string, timeout time.Duration, output io.Writer) (string, []string, error) {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// NeoPRISM API format
	submitURL := fmt.Sprintf("%s/api/signed-operation-submissions", strings.TrimSuffix(neoprismURL, "/"))

	reqBody := map[string]interface{}{
		"signed_operations": []string{signedOpHex},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, submitURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", nil, fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	// Use a client with timeout
	client := &http.Client{
		Timeout: timeout,
	}

	// Log that we're sending the request
	if output != nil {
		fmt.Fprintf(output, "   Sending request to %s...\n", submitURL)
	}

	startTime := time.Now()
	resp, err := client.Do(req)
	requestTime := time.Since(startTime)

	if err != nil {
		// Check if it's a context timeout
		if ctx.Err() == context.DeadlineExceeded {
			return "", nil, fmt.Errorf("submission timeout after %v: NeoPRISM did not respond in time. This usually means:\n- NeoPRISM is processing the transaction (can take time on blockchain)\n- NeoPRISM is unavailable or overloaded\n- Network connectivity issues\n\nTry: curl -X POST %s/api/signed-operation-submissions -H 'Content-Type: application/json' -d '{\"signed_operations\":[\"test\"]}' to test connectivity", requestTime, submitURL)
		}
		return "", nil, fmt.Errorf("submit operation (took %v): %w", requestTime, err)
	}
	defer resp.Body.Close()

	if output != nil {
		fmt.Fprintf(output, "   Received response (status %d) in %v\n", resp.StatusCode, requestTime)
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", nil, fmt.Errorf("submission failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse NeoPRISM response
	var submitResponse struct {
		TxID         string   `json:"tx_id"`
		OperationIDs []string `json:"operation_ids"`
	}

	if err := json.Unmarshal(respBody, &submitResponse); err != nil {
		return "", nil, fmt.Errorf("parse response: %w", err)
	}

	return submitResponse.TxID, submitResponse.OperationIDs, nil
}

// waitForDIDDocument waits for a PRISM DID document to be available on the resolver
func waitForDIDDocument(ctx context.Context, prismDID did.DID, prismURL string, timeout time.Duration, output io.Writer) error {
	// Configure resolver
	originalConfig := did.GetPRISMResolverConfig()
	defer did.SetPRISMResolverConfig(originalConfig)

	did.SetPRISMResolverConfig(did.PRISMResolverConfig{
		ResolverURL:                 prismURL,
		PreferredVerificationMethod: "authentication",
	})

	// Create a context with timeout
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Poll for DID document
	retryDelay := 2 * time.Second
	maxRetries := int(timeout / retryDelay)
	if maxRetries < 1 {
		maxRetries = 1
	}

	var lastErr error
	lastLogTime := time.Now()
	logInterval := 5 * time.Second // Log progress every 5 seconds

	for i := 0; i < maxRetries; i++ {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return fmt.Errorf("timeout waiting for DID document: %w", lastErr)
			}
			return fmt.Errorf("timeout waiting for DID document")
		default:
		}

		// Try to resolve the DID
		anchor, err := did.GetAnchorForDID(prismDID)
		if err == nil {
			// Successfully resolved - verify it has authentication methods
			// The anchor creation already verifies the DID document exists
			// We can trust that if GetAnchorForDID succeeds, the DID is valid
			_ = anchor // Use anchor to avoid unused variable
			return nil
		}

		lastErr = err

		// Log progress periodically
		now := time.Now()
		if now.Sub(lastLogTime) >= logInterval {
			fmt.Fprintf(output, "   Still waiting... (attempt %d/%d)\n", i+1, maxRetries)
			lastLogTime = now
		}

		// Wait before retrying (except on last attempt)
		if i < maxRetries-1 {
			select {
			case <-ctx.Done():
				errMsg := "timeout waiting for DID document"
				if lastErr != nil {
					return fmt.Errorf("%s: %w", errMsg, lastErr)
				}
				return fmt.Errorf("%s", errMsg)
			case <-time.After(retryDelay):
				// Continue to next iteration
			}
		}
	}

	errMsg := fmt.Sprintf("failed to resolve DID document after %d attempts", maxRetries)
	if lastErr != nil {
		return fmt.Errorf("%s: %w", errMsg, lastErr)
	}
	return fmt.Errorf("%s", errMsg)
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
