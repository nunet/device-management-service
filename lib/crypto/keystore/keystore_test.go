// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package keystore

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type keystoreTestSuite struct {
	fs       afero.Fs
	keysDir  string
	keystore *BasicKeyStore
}

func newKeystoreTestSuite(t *testing.T) *keystoreTestSuite {
	t.Helper()
	fs := afero.NewMemMapFs()
	keysDir := "/tmp/dms/keystore"

	keystore, err := New(fs, keysDir)
	require.NoError(t, err)
	require.NotNil(t, keystore)

	return &keystoreTestSuite{
		fs:       fs,
		keysDir:  keysDir,
		keystore: keystore,
	}
}

func TestNew(t *testing.T) {
	t.Parallel()
	cases := map[string]struct {
		fs      afero.Fs
		keysDir string
		expErr  error
	}{
		"keysDir empty": {
			fs:      afero.NewMemMapFs(),
			keysDir: "",
			expErr:  ErrEmptyKeysDir,
		},
		"mkdir error": {
			fs:      afero.NewReadOnlyFs(afero.NewMemMapFs()),
			keysDir: "/tmp/dms/keystore",
			expErr:  ErrCreateKeysDir,
		},
		"success": {
			fs:      afero.NewMemMapFs(),
			keysDir: "/tmp/dms/keystore",
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			keystore, err := New(tt.fs, tt.keysDir)
			if tt.expErr != nil {
				assert.Nil(t, keystore)
				assert.ErrorIs(t, err, tt.expErr)
			} else {
				assert.NotNil(t, keystore)
			}
		})
	}
}

func TestBasicKeyStoreSave(t *testing.T) {
	t.Parallel()

	suite := newKeystoreTestSuite(t)

	t.Run("invalid passphrase", func(t *testing.T) {
		t.Parallel()
		path, err := suite.keystore.Save("id123", []byte("hello world"), "")
		assert.ErrorIs(t, err, ErrEmptyPassphrase)
		assert.Empty(t, path)
	})

	t.Run("valid passphrase", func(t *testing.T) {
		t.Parallel()
		path, err := suite.keystore.Save("id123", []byte("hello world"), "1234")
		assert.NoError(t, err)
		assert.NotEmpty(t, path)

		exists, err := afero.Exists(suite.fs, path)
		assert.NoError(t, err)
		assert.True(t, exists)
	})
}

func TestBasicKeyStoreGet(t *testing.T) {
	t.Parallel()
	suite := newKeystoreTestSuite(t)

	id := "keyid test get"
	data := []byte("hello world")
	passphrase := "1234"

	path, err := suite.keystore.Save(id, data, passphrase)
	require.NoError(t, err)
	require.NotEmpty(t, path)

	t.Run("wrong passphrase", func(t *testing.T) {
		t.Parallel()
		_, err := suite.keystore.Get(id, "wrong")
		assert.ErrorIs(t, err, ErrMACMismatch)
	})

	t.Run("non-existent keyID", func(t *testing.T) {
		t.Parallel()
		_, err := suite.keystore.Get("non-existent", passphrase)
		assert.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		key, err := suite.keystore.Get(id, passphrase)
		assert.NoError(t, err)
		assert.NotNil(t, key)
		assert.Equal(t, id, key.ID)
		assert.Equal(t, data, key.Data)
	})
}

func TestBasicKeyStoreDelete(t *testing.T) {
	t.Parallel()
	suite := newKeystoreTestSuite(t)

	id := "key id test delete"
	data := []byte("hello world")
	passphrase := "1234"

	path, err := suite.keystore.Save(id, data, passphrase)
	require.NoError(t, err)
	require.NotEmpty(t, path)

	t.Run("wrong passphrase", func(t *testing.T) {
		t.Parallel()
		err := suite.keystore.Delete(id, "wrong")
		assert.Error(t, err)
	})

	t.Run("non-existent keyID", func(t *testing.T) {
		t.Parallel()
		err := suite.keystore.Delete("non-existent", passphrase)
		assert.ErrorIs(t, err, ErrKeyNotFound)
	})

	t.Run("valid", func(t *testing.T) {
		t.Parallel()
		err := suite.keystore.Delete(id, passphrase)
		assert.NoError(t, err)

		exists, err := afero.Exists(suite.fs, path)
		assert.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestBasicKeyStoreListKeys(t *testing.T) {
	t.Parallel()

	suite := newKeystoreTestSuite(t)

	t.Run("empty keystore", func(t *testing.T) {
		t.Parallel()

		keys, err := suite.keystore.ListKeys()
		assert.NoError(t, err)
		assert.Empty(t, keys)
	})

	t.Run("with keys", func(t *testing.T) {
		t.Parallel()

		ids := []string{"id1", "id2", "id3"}
		for _, id := range ids {
			_, err := suite.keystore.Save(id, []byte("data"), "pass")
			require.NoError(t, err)
		}

		keys, err := suite.keystore.ListKeys()
		assert.NoError(t, err)
		assert.ElementsMatch(t, ids, keys)
	})
}

func TestBasicKeyStoreExists(t *testing.T) {
	t.Parallel()
	suite := newKeystoreTestSuite(t)

	exists := suite.keystore.Exists("nonexistentkey")
	assert.False(t, exists)

	testKey := "testkey"
	_, err := suite.keystore.Save(testKey, []byte("testdata"), "pass")
	assert.NoError(t, err)

	exists = suite.keystore.Exists(testKey)
	assert.True(t, exists)
}
