// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package crypto

import (
	"crypto/rand"
	"errors"
	"io"

	"golang.org/x/crypto/sha3"
)

// RandomEntropy bytes from rand.Reader
func RandomEntropy(length int) ([]byte, error) {
	if length < 0 {
		return nil, errors.New("length must be non-negative")
	}
	buf := make([]byte, length)
	n, err := io.ReadFull(rand.Reader, buf)
	if err != nil || n != length {
		return nil, errors.New("failed to read random bytes")
	}
	return buf, nil
}

// Sha3 return sha3 of a given byte array
func Sha3(data ...[]byte) ([]byte, error) {
	d := sha3.New256()
	for _, b := range data {
		_, err := d.Write(b)
		if err != nil {
			return nil, err
		}
	}
	return d.Sum(nil), nil
}
