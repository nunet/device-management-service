// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package node

import (
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"
)

func TestResolveLedgerIndexAndAliasPersistence(t *testing.T) {
	t.Parallel()

	userDir := "/tmp/dms/user"
	fs := afero.NewMemMapFs()

	// bare "ledger" & empty key should be 0
	idx, err := ResolveLedgerIndex(fs, userDir, "")
	require.NoError(t, err)
	require.Equal(t, 0, idx)

	idx, err = ResolveLedgerIndex(fs, userDir, "ledger")
	require.NoError(t, err)
	require.Equal(t, 0, idx)

	// numeric string
	idx, err = ResolveLedgerIndex(fs, userDir, "7")
	require.NoError(t, err)
	require.Equal(t, 7, idx)

	// unknown alias must error before we create it
	_, err = ResolveLedgerIndex(fs, userDir, "business")
	require.Error(t, err)

	// persist alias "business" - 3
	err = SetLedgerAlias(fs, userDir, "business", 3)
	require.NoError(t, err)

	// file should exist under cap/ledger_aliases.json
	aliasJSON := filepath.Join(userDir, LedgerAliasFile)
	_, err = fs.Stat(aliasJSON)
	require.NoError(t, err)

	// resolver should now pick up the alias
	idx, err = ResolveLedgerIndex(fs, userDir, "business")
	require.NoError(t, err)
	require.Equal(t, 3, idx)
}

func TestSetLedgerAliasValidation(t *testing.T) {
	t.Parallel()

	userDir := testUserDir
	fs := afero.NewMemMapFs()

	// purely numeric alias is rejected
	err := SetLedgerAlias(fs, userDir, "123", 0)
	require.Error(t, err)

	// negative index is rejected
	err = SetLedgerAlias(fs, userDir, "neg", -1)
	require.Error(t, err)

	// alias containing colon is rejected
	err = SetLedgerAlias(fs, userDir, "bad:alias", 1)
	require.Error(t, err)
}
