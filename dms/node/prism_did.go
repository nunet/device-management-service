// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/lib/did"
)

const PrismDIDMapFile = "prism_did_map.json"

// prismDIDMapFilePath returns the path to the PRISM DID mapping file
func prismDIDMapFilePath(userDir string) string {
	return filepath.Join(userDir, PrismDIDMapFile)
}

// loadPrismDIDMap loads the PRISM DID to key name mapping
// Missing file returns empty map (no error)
func loadPrismDIDMap(fs afero.Fs, userDir string) (map[string]string, error) {
	path := prismDIDMapFilePath(userDir)

	mapping := make(map[string]string)

	f, err := fs.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return mapping, nil
		}
		return nil, fmt.Errorf("open PRISM DID map: %w", err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&mapping); err != nil {
		return nil, fmt.Errorf("decode PRISM DID map: %w", err)
	}
	return mapping, nil
}

// savePrismDIDMap writes the PRISM DID mapping atomically
func savePrismDIDMap(fs afero.Fs, userDir string, mapping map[string]string) error {
	path := prismDIDMapFilePath(userDir)

	// Create backup if file exists
	if _, err := fs.Stat(path); err == nil {
		backupPath := path + ".bak"
		_ = fs.Remove(backupPath) // ignore error
		// Read existing file and write to backup
		existingData, err := afero.ReadFile(fs, path)
		if err == nil {
			if err := afero.WriteFile(fs, backupPath, existingData, 0o600); err != nil {
				return fmt.Errorf("backup PRISM DID map: %w", err)
			}
		}
	}

	// Write to temp file first
	tmpPath := path + ".tmp"
	data, err := json.MarshalIndent(mapping, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal PRISM DID map: %w", err)
	}

	if err := afero.WriteFile(fs, tmpPath, data, 0o600); err != nil {
		return fmt.Errorf("write PRISM DID map: %w", err)
	}

	// Atomic rename
	if err := fs.Rename(tmpPath, path); err != nil {
		_ = fs.Remove(tmpPath) // cleanup
		return fmt.Errorf("rename PRISM DID map: %w", err)
	}

	return nil
}

// SetPrismDID associates a key name with a PRISM DID
func SetPrismDID(fs afero.Fs, userDir, keyName, prismDID string) error {
	mapping, err := loadPrismDIDMap(fs, userDir)
	if err != nil {
		return err
	}

	// Validate PRISM DID
	didObj, err := did.FromString(prismDID)
	if err != nil {
		return fmt.Errorf("invalid PRISM DID: %w", err)
	}

	if didObj.Method() != "prism" {
		return fmt.Errorf("expected PRISM DID (did:prism:...), got %s", didObj.Method())
	}

	mapping[keyName] = prismDID
	return savePrismDIDMap(fs, userDir, mapping)
}

// GetPrismDID retrieves the PRISM DID for a key name
// Returns empty string if not found
func GetPrismDID(fs afero.Fs, userDir, keyName string) (string, error) {
	mapping, err := loadPrismDIDMap(fs, userDir)
	if err != nil {
		return "", err
	}

	return mapping[keyName], nil
}

// RemovePrismDID removes the PRISM DID association for a key name
func RemovePrismDID(fs afero.Fs, userDir, keyName string) error {
	mapping, err := loadPrismDIDMap(fs, userDir)
	if err != nil {
		return err
	}

	delete(mapping, keyName)
	return savePrismDIDMap(fs, userDir, mapping)
}
