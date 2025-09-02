// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

const (
	DefaultContextName = "dms"
	UserContextName    = "user"
	KeystoreDir        = "key/"
	CapstoreDir        = "cap/"
	DMSPassphraseEnv   = "DMS_PASSPHRASE"
)

const ledger = "ledger"

// IsLedgerContext checks if the context is a ledger context.
func IsLedgerContext(context string) bool {
	return strings.HasPrefix(context, ledger)
}

// GetContextKey returns the key part of the context, if it has a prefix.
func GetContextKey(context string) string {
	parts := strings.Split(context, ":")
	if len(parts) != 2 {
		return context
	}

	return parts[1]
}

func CreateTrustContextFromKeyStore(
	fs afero.Fs, contextKey,
	passphrase, keyStorePath string,
) (did.TrustContext, crypto.PrivKey, error) {
	keyStoreDir := filepath.Join(keyStorePath, KeystoreDir)

	// support FS cache for E2E tests #1139
	fsCache := os.Getenv("DMS_E2E_CACHE_KEYS") == "1"
	ks, err := keystore.New(fs, keyStoreDir, fsCache)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open keystore: %w", err)
	}

	ksPrivKey, err := ks.Get(contextKey, passphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get key from keystore %s: %w", contextKey, err)
	}

	priv, err := ksPrivKey.PrivKey()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to convert key from keystore to private key: %w", err)
	}

	trustCtx, err := did.NewTrustContextWithPrivateKey(priv)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create trust context: %w", err)
	}

	return trustCtx, priv, nil
}

func LoadCapabilityContext(trustCtx did.TrustContext, fs afero.Fs, name string, capStorePath string) (ucan.CapabilityContext, error) {
	capStoreDir := filepath.Join(capStorePath, CapstoreDir)
	capStoreFile := filepath.Join(capStoreDir, fmt.Sprintf("%s.cap", name))

	f, err := fs.Open(capStoreFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open capability context file: %w", err)
	}
	defer f.Close()

	capCtx, err := ucan.LoadCapabilityContextWithName(name, trustCtx, f)
	if err != nil {
		return nil, fmt.Errorf("unable to load capability context: %w", err)
	}

	return capCtx, nil
}

func SaveCapabilityContext(capCtx ucan.CapabilityContext, fs afero.Fs, capStorePath string) error {
	name := capCtx.Name()
	capStoreDir := filepath.Join(capStorePath, CapstoreDir)
	capCtxFile := filepath.Join(capStoreDir, fmt.Sprintf("%s.cap", name))
	capCtxBackup := filepath.Join(capStoreDir, fmt.Sprintf("%s.cap.%d", name, time.Now().Unix()))

	// ensure the directory exists
	if err := fs.MkdirAll(capStoreDir, os.FileMode(0o700)); err != nil {
		return fmt.Errorf("creating capability context director: %w", err)
	}

	// first take a backup -- move the current context
	if _, err := fs.Stat(capCtxFile); err == nil {
		if err := fs.Rename(capCtxFile, capCtxBackup); err != nil {
			return fmt.Errorf("error backing up current capability context: %w", err)
		}
	}

	// now open for writing
	f, err := fs.Create(capCtxFile)
	if err != nil {
		return fmt.Errorf("error creating new capability context file: %w", err)
	}
	defer f.Close()

	if err := ucan.SaveCapabilityContext(capCtx, f); err != nil {
		return fmt.Errorf("error saving capability context: %w", err)
	}

	return nil
}

// GetTrustContext returns a DID TrustContext for the given logical context
// name. For ledger-backed contexts it now supports:
//
//   - “ledger”                 → account index 0
//   - “ledger:<index>”         → explicit ledger account index
//   - “ledger:<alias>”         → alias resolved via ledger_aliases.json
func GetTrustContext(
	fs afero.Fs,
	context, passphrase, userDir string,
) (did.TrustContext, error) {
	// Ledger path
	if IsLedgerContext(context) {
		idx, err := ResolveLedgerIndex(fs, userDir, GetContextKey(context))
		if err != nil {
			return nil, err
		}

		provider, err := did.NewLedgerWalletProvider(idx)
		if err != nil {
			return nil, err
		}

		return did.NewTrustContextWithProvider(provider), nil
	}

	// Keystore path
	trustCtx, _, err := CreateTrustContextFromKeyStore(
		fs, context, passphrase, userDir,
	)
	return trustCtx, err
}
