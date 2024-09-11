package cap

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms"
	"gitlab.com/nunet/device-management-service/internal/config"
	"gitlab.com/nunet/device-management-service/lib/crypto"
	"gitlab.com/nunet/device-management-service/lib/crypto/keystore"
	"gitlab.com/nunet/device-management-service/lib/did"
	"gitlab.com/nunet/device-management-service/lib/ucan"
)

func CreateTrustContextFromKeyStore(afs afero.Afero, contextKey string) (did.TrustContext, crypto.PrivKey, error) {
	keyStoreDir := filepath.Join(config.GetConfig().General.UserDir, dms.KeystoreDir)

	ks, err := keystore.New(afs.Fs, keyStoreDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open keystore: %w", err)
	}

	passphrase, err := utils.PromptForPassphrase(false)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get passphrase: %w", err)
	}

	ksPrivKey, err := ks.Get(contextKey, passphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get key from keystore: %w", err)
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

func LoadCapabilityContext(trustCtx did.TrustContext, name string) (ucan.CapabilityContext, error) {
	capStoreDir := filepath.Join(config.GetConfig().General.UserDir, dms.CapstoreDir)
	capStoreFile := filepath.Join(capStoreDir, fmt.Sprintf("%s.cap", name))

	f, err := os.Open(capStoreFile)
	if err != nil {
		return nil, fmt.Errorf("unable to open capability context file: %w", err)
	}
	defer f.Close()

	capCtx, err := ucan.LoadCapabilityContext(trustCtx, f)
	if err != nil {
		return nil, fmt.Errorf("unable to load capability context: %w", err)
	}

	return capCtx, nil
}

func SaveCapabilityContext(capCtx ucan.CapabilityContext, name string) error {
	capStoreDir := filepath.Join(config.GetConfig().General.UserDir, dms.CapstoreDir)
	capCtxFile := filepath.Join(capStoreDir, fmt.Sprintf("%s.cap", name))
	capCtxBackup := filepath.Join(capStoreDir, fmt.Sprintf("%s.cap.%d", name, time.Now().Unix()))

	// first take a backup -- move the current context
	if _, err := os.Stat(capCtxFile); err == nil {
		if err := os.Rename(capCtxFile, capCtxBackup); err != nil {
			return fmt.Errorf("error backing up current capability context: %w", err)
		}
	}

	// now open for writing
	f, err := os.Create(capCtxFile)
	if err != nil {
		return fmt.Errorf("error creating new capability context file: %w", err)
	}
	defer f.Close()

	if err := ucan.SaveCapabilityContext(capCtx, f); err != nil {
		return fmt.Errorf("error saving capability context: %w", err)
	}

	return nil
}
