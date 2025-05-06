// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package did

import (
	"fmt"
	"strings"
)

type DID struct {
	URI string `json:"uri,omitempty"`
}

func (did DID) Equal(other DID) bool {
	return did.URI == other.URI
}

func (did DID) Empty() bool {
	return did.URI == ""
}

func (did DID) String() string {
	return did.URI
}

func (did DID) Method() string {
	parts := strings.Split(did.URI, ":")
	if len(parts) == 3 {
		return parts[1]
	}

	return ""
}

func (did DID) Identifier() string {
	parts := strings.Split(did.URI, ":")
	if len(parts) == 3 {
		return parts[2]
	}

	return ""
}

func FromString(s string) (DID, error) {
	if s != "" {
		parts := strings.Split(s, ":")
		if len(parts) != 3 {
			return DID{},
				fmt.Errorf("%w: %s", ErrInvalidDID, s)
		}

		for _, part := range parts {
			if part == "" {
				return DID{},
					fmt.Errorf("%w: %s", ErrInvalidDID, s)
			}
		}

		// TODO validate parts according to spec: https://www.w3.org/TR/did-core/
	}

	return DID{URI: s}, nil
}
