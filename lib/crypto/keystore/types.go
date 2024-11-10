// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package keystore

type cipherparamsJSON struct {
	IV string `json:"iv"`
}

// ScryptParams
type ScryptParams struct {
	N          int    `json:"n"`
	R          int    `json:"r"`
	P          int    `json:"p"`
	DKeyLength int    `json:"dklen"`
	Salt       string `json:"salt"`
}

type cryptoJSON struct {
	Cipher       string           `json:"cipher"`
	CipherText   string           `json:"ciphertext"`
	CipherParams cipherparamsJSON `json:"cipherparams"`
	KDF          string           `json:"kdf"`
	KDFParams    ScryptParams     `json:"kdfparams"`
	MAC          string           `json:"mac"`
}

type encryptedKeyJSON struct {
	ID      string     `json:"id"`
	Crypto  cryptoJSON `json:"crypto"`
	Version int        `json:"version"`
}
