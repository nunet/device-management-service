// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package did

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// create fake ledger-cli in a temp dir and prepend to PATH
func fakeLedgerCLI(t *testing.T, script string) func() {
	tmp := t.TempDir()
	bin := filepath.Join(tmp, "ledger-cli")
	require.NoError(t, os.WriteFile(bin, []byte(script), 0o755))

	orig := os.Getenv("PATH")
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+orig)

	return func() { t.Setenv("PATH", orig) }
}

// valid compressed secp256k1 generator point (33 bytes → 66 hex chars)
const generatorHex = "0279BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798"

// happy-path up to Sign() returning bytes (we don’t verify the signature)
func TestLedgerStubHappyPath(t *testing.T) {
	restore := fakeLedgerCLI(t, `#!/bin/sh
case "$1" in
  key)
    echo '{"key":"`+generatorHex+`","address":"0x00"}' > "$3"
    ;;
  sign)
    # minimal parseable signature: r=1, s=1
    echo '{"ecdsa":{"v":27,"r":"01","s":"01"}}' > "$3"
    ;;
esac
`)
	defer restore()

	prov, err := NewLedgerWalletProvider(0)
	require.NoError(t, err)

	_, err = prov.Sign([]byte("payload"))
	require.NoError(t, err)
}

// CLI missing → LookPath error
func TestLedgerCLIMissing(t *testing.T) {
	orig := os.Getenv("PATH")
	t.Setenv("PATH", "")
	defer t.Setenv("PATH", orig)

	_, err := NewLedgerWalletProvider(0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "can't find ledger-cli")
}

// Invalid hex in key JSON → decode ledger key error
func TestLedgerInvalidHexKey(t *testing.T) {
	restore := fakeLedgerCLI(t, `#!/bin/sh
echo '{"key":"ZZZ","address":"x"}' > "$3"
`)
	defer restore()

	_, err := NewLedgerWalletProvider(0)
	require.Error(t, err)
	require.Contains(t, err.Error(), "decode ledger key")
}

// Malformed JSON from “sign” → parse ledger output error

func TestLedgerMalformedSignJSON(t *testing.T) {
	restore := fakeLedgerCLI(t, `#!/bin/sh
case "$1" in
  key)
    echo '{"key":"`+generatorHex+`","address":"0x00"}' > "$3"
    ;;
  sign)
    echo 'not-json' > "$3"
    ;;
esac
`)
	defer restore()

	prov, err := NewLedgerWalletProvider(0)
	require.NoError(t, err)

	_, err = prov.Sign([]byte("data"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "parse ledger output")
}
