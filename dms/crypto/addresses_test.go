// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package crypto

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetCardanoAddressAndMnemonic(t *testing.T) {
	addrAndMnemonic, _ := GetCardanoAddressAndMnemonic()
	addr := addrAndMnemonic.Address
	mnemonic := addrAndMnemonic.Mnemonic

	t.Run("cardano address is 103 characters long", func(t *testing.T) {
		want := 103
		assert.Equal(t, len(addr), want)
	})

	t.Run("cardano mnemonic is 24 words long", func(t *testing.T) {
		want := 24
		got := len(strings.Split(mnemonic, " "))
		assert.Equal(t, want, got)
	})
}
