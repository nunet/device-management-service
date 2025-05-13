// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package keystore

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/utils"
)

// KeyStore manages a local keystore with lock and unlock functionalities.
type KeyStore interface {
	Save(id string, data []byte, passphrase string) (string, error)
	Get(keyID string, passphrase string) (*Key, error)
	Delete(keyID string, passphrase string) error
	ListKeys() ([]string, error)
}

// BasicKeyStore handles keypair storage.
// TODO: add cache?
type BasicKeyStore struct {
	fs      afero.Fs
	keysDir string
	mu      sync.RWMutex
}

var _ KeyStore = (*BasicKeyStore)(nil)

// New creates a new BasicKeyStore.
func New(fs afero.Fs, keysDir string) (*BasicKeyStore, error) {
	if keysDir == "" {
		return nil, ErrEmptyKeysDir
	}

	if err := fs.MkdirAll(keysDir, 0o700); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateKeysDir, err)
	}

	return &BasicKeyStore{
		fs:      fs,
		keysDir: keysDir,
	}, nil
}

// Save encrypts a key and writes it to a file.
func (ks *BasicKeyStore) Save(id string, data []byte, passphrase string) (string, error) {
	if passphrase == "" {
		return "", ErrEmptyPassphrase
	}

	key := &Key{
		ID:   id,
		Data: data,
	}

	keyDataJSON, err := key.MarshalToJSON(passphrase)
	if err != nil {
		return "", fmt.Errorf("failed to marshal key: %w", err)
	}

	filename, err := utils.WriteToFile(ks.fs, keyDataJSON, filepath.Join(ks.keysDir, key.ID+".json"))
	if err != nil {
		return "", fmt.Errorf("failed to write key to file: %w", err)
	}

	return filename, nil
}

// Get unlocks a key by keyID.
func (ks *BasicKeyStore) Get(keyID string, passphrase string) (*Key, error) {
	bts, err := afero.ReadFile(ks.fs, filepath.Join(ks.keysDir, keyID+".json"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to read keystore file: %w", err)
	}

	key, err := UnmarshalKey(bts, passphrase)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal keystore file: %w", err)
	}

	return key, err
}

// Exists returns whether a key is stored
func (ks *BasicKeyStore) Exists(key string) bool {
	keys, err := ks.ListKeys()
	if err != nil {
		return false
	}
	return slices.Contains(keys, key)
}

// Delete removes the file referencing the given key.
func (ks *BasicKeyStore) Delete(keyID string, passphrase string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	filePath := filepath.Join(ks.keysDir, keyID+".json")
	bts, err := afero.ReadFile(ks.fs, filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrKeyNotFound
		}
		return fmt.Errorf("failed to read keystore file: %w", err)
	}

	_, err = UnmarshalKey(bts, passphrase)
	if err != nil {
		return fmt.Errorf("invalid passphrase or corrupted key file: %w", err)
	}

	err = ks.fs.Remove(filePath)
	if err != nil {
		return fmt.Errorf("failed to delete key file: %w", err)
	}

	return nil
}

// ListKeys lists the keys in the keysDir.
func (ks *BasicKeyStore) ListKeys() ([]string, error) {
	keys := make([]string, 0)

	dirEntries, err := afero.ReadDir(ks.fs, ks.keysDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read keystore directory: %w", err)
	}

	for _, entry := range dirEntries {
		_, err := afero.ReadFile(ks.fs, filepath.Join(ks.keysDir, entry.Name()))
		if err != nil {
			continue
		}

		keys = append(keys, removeFileExtension(entry.Name()))
	}

	return keys, nil
}
