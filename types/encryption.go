// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

type EncryptionType int

const (
	EncryptionTypeNull EncryptionType = iota
)

// Encryptor (DRAFT)
//
// Warning: this is just a draft. And it might be moved to an Encryption package
//
// TODO: it must support encryption of files/directories, otherwise we have to
// create another interface for the usecase
type Encryptor interface {
	Encrypt([]byte) ([]byte, error)
}

// Decryptor (DRAFT)
//
// Warning: this is just a draft. And it might be moved to an Encryption package
//
// TODO: it must support decryption of files/directories, otherwise we have to
// create another interface for the usecase
type Decryptor interface {
	Decrypt([]byte) ([]byte, error)
}
