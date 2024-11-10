// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package keystore

import "errors"

var (
	ErrEmptyPassphrase   = errors.New("passphrase is empty")
	ErrVersionMismatch   = errors.New("version mismatch")
	ErrCipherMismatch    = errors.New("cipher mismatch")
	ErrMACMismatch       = errors.New("mac mismatch")
	ErrEmptyKeysDir      = errors.New("keysDir is empty")
	ErrEmptyNodeIdentity = errors.New("nodeIdentityData is empty")
	ErrKeyNotFound       = errors.New("key not found on this node")
	ErrTokenInvalid      = errors.New("token is invalid")
	ErrNotUnlockedKey    = errors.New("key is not unlocked")
)
