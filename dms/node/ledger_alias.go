// Copyright 2025, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions
// and limitations under the License.

package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/afero"
)

const LedgerAliasFile = "ledger_aliases.json"

// $USERDIR/ledger_aliases.json  (new – root of the user dir)
func aliasFilePath(userDir string) string {
	return filepath.Join(userDir, LedgerAliasFile)
}

// loadLedgerAliases parses the on-disk alias table
// Missing file  empty map (no error)
func loadLedgerAliases(fs afero.Fs, userDir string) (map[string]int, error) {
	path := aliasFilePath(userDir)

	aliases := make(map[string]int)

	f, err := fs.Open(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return aliases, nil
		}
		return nil, fmt.Errorf("open alias table: %w", err)
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&aliases); err != nil {
		return nil, fmt.Errorf("decode alias table: %w", err)
	}
	return aliases, nil
}

// saveLedgerAliases writes aliases atomically, backing up the old file first.
func saveLedgerAliases(fs afero.Fs, userDir string, aliases map[string]int) error {
	dir := userDir
	if err := fs.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create alias dir: %w", err)
	}

	path := aliasFilePath(userDir)
	backup := fmt.Sprintf("%s.%d", path, time.Now().Unix())

	// Backup existing file, if present
	if _, err := fs.Stat(path); err == nil {
		if err := fs.Rename(path, backup); err != nil {
			return fmt.Errorf("backup alias table: %w", err)
		}
	}

	// Write to temp file, then rename
	tmp, err := afero.TempFile(fs, dir, "aliases.*.tmp")
	if err != nil {
		return fmt.Errorf("create temp alias file: %w", err)
	}
	tmpName := tmp.Name()

	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(aliases); err != nil {
		tmp.Close()
		return fmt.Errorf("encode alias table: %w", err)
	}
	tmp.Close()

	// Best-effort: set permissions to 0600. Ignore if FS doesn’t support it.
	_ = fs.Chmod(tmpName, 0o600)

	if err := fs.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename alias file: %w", err)
	}
	return nil
}

// ResolveLedgerIndex turns <key> from “ledger:<key>” into the actual
// account index:
//
//   - ""  or "ledger"  → 0
//   - all-digits       → parsed integer (must be ≥0)
//   - otherwise        → look up alias in the on-disk table
func ResolveLedgerIndex(fs afero.Fs, userDir, key string) (int, error) {
	k := strings.TrimSpace(key)
	if k == "" || k == "ledger" {
		return 0, nil
	}

	if n, err := strconv.Atoi(k); err == nil {
		if n < 0 {
			return 0, fmt.Errorf("ledger index cannot be negative: %d", n)
		}
		return n, nil
	}

	aliases, err := loadLedgerAliases(fs, userDir)
	if err != nil {
		return 0, err
	}
	idx, ok := aliases[k]
	if !ok {
		return 0, fmt.Errorf("ledger alias not found: %q", k)
	}
	return idx, nil
}

func SetLedgerAlias(fs afero.Fs, userDir, alias string, index int) error {
	alias = strings.TrimSpace(alias)

	if alias == "" {
		return fmt.Errorf("alias cannot be empty")
	}
	if strings.Contains(alias, ":") {
		return fmt.Errorf("alias cannot contain ':'")
	}
	if _, err := strconv.Atoi(alias); err == nil {
		return fmt.Errorf("alias cannot be purely numeric")
	}
	if index < 0 {
		return fmt.Errorf("ledger index cannot be negative")
	}

	aliases, err := loadLedgerAliases(fs, userDir)
	if err != nil {
		return err
	}
	aliases[alias] = index
	return saveLedgerAliases(fs, userDir, aliases)
}
