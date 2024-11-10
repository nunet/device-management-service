// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"errors"

	"github.com/fivebinaries/go-cardano-serialization/address"
)

// ValidateAddress checks if the wallet address is a valid cardano address
func ValidateAddress(addr string) error {
	validCardano := false
	isValidCardano(addr, &validCardano)
	if validCardano {
		return nil
	}

	return errors.New("invalid cardano wallet address")
}

// isValidCardano checks if the cardano address is valid
func isValidCardano(addr string, valid *bool) {
	defer func() {
		if r := recover(); r != nil {
			*valid = false
		}
	}()
	if _, err := address.NewAddress(addr); err == nil {
		*valid = true
	}
}
