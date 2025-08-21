// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package keystore

import (
	"bytes"
	"encoding/gob"
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
	Exists(key string) bool
	Dir() string
}

// BasicKeyStore handles keypair storage.
// TODO: add cache?
type BasicKeyStore struct {
	fs      afero.Fs
	keysDir string
	mu      sync.RWMutex
	cache   map[string]*Key
	fsCache bool
}

var _ KeyStore = (*BasicKeyStore)(nil)

// New creates a new BasicKeyStore.
//
// fsCache: keeps unmarshalled keys in the file system. Insecure, only for tests.
func New(fs afero.Fs, keysDir string, fsCache bool) (*BasicKeyStore, error) {
	if keysDir == "" {
		return nil, ErrEmptyKeysDir
	}

	if err := fs.MkdirAll(keysDir, 0o700); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrCreateKeysDir, err)
	}

	return &BasicKeyStore{
		fs:      fs,
		keysDir: keysDir,
		cache:   make(map[string]*Key),
		fsCache: fsCache,
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

	// cache
	delete(ks.cache, key.ID)
	if ks.fsCache {
		_ = ks.fs.Remove(filepath.Join(ks.keysDir, id+".gob"))
	}

	return filename, nil
}

// Get unlocks a key by keyID.
func (ks *BasicKeyStore) Get(keyID string, passphrase string) (*Key, error) {
	// read cache?
	fsCachePath := filepath.Join(ks.keysDir, keyID+".gob")
	if key, ok := ks.cache[keyID]; ok {
		return key, nil
	}
	if ks.fsCache {
		if b, err := afero.ReadFile(ks.fs, fsCachePath); err == nil {
			key := &Key{}
			if err := gob.NewDecoder(bytes.NewReader(b)).Decode(key); err == nil {
				ks.cache[keyID] = key
				return key, nil
			}
		}
	}

	// read & unmarshall
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

	// save cache
	ks.cache[keyID] = key
	if ks.fsCache {
		var buf bytes.Buffer
		if err := gob.NewEncoder(&buf).Encode(key); err != nil {
			return key, nil
		}
		if err := afero.WriteFile(ks.fs, fsCachePath, buf.Bytes(), 0o600); err != nil {
			return key, nil
		}
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

	// cache
	delete(ks.cache, keyID)
	if ks.fsCache {
		_ = ks.fs.Remove(filepath.Join(ks.keysDir, keyID+".gob"))
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

func (ks *BasicKeyStore) Dir() string {
	return ks.keysDir
}
