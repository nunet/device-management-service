// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package tun

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJoinedNetworks(t *testing.T) {
	addrs, err := JoinedNetworks()
	require.NoError(t, err)

	for _, addr := range addrs {
		fmt.Println(addr)
		assert.True(t, strings.Contains(addr, "/"))
		assert.True(t, strings.Contains(addr, "."))
		assert.True(t, !strings.Contains(addr, ":"))
	}
}
