// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestUseMessageOptsFlags(t *testing.T) {
	cmd := &cobra.Command{}

	useMessageOptsFlags(cmd, false)

	// Check flags are added
	contextFlag := cmd.Flags().Lookup(fnContext)
	assert.NotNil(t, contextFlag)
	assert.Equal(t, "c", contextFlag.Shorthand)

	destFlag := cmd.Flags().Lookup(fnDest)
	assert.NotNil(t, destFlag)
	assert.Equal(t, "d", destFlag.Shorthand)

	timeoutFlag := cmd.Flags().Lookup(fnTimeout)
	assert.NotNil(t, timeoutFlag)
	assert.Equal(t, "t", timeoutFlag.Shorthand)

	expiryFlag := cmd.Flags().Lookup(fnExpiry)
	assert.NotNil(t, expiryFlag)
	assert.Equal(t, "e", expiryFlag.Shorthand)
}

func TestUseNewMsgOptsFlags(t *testing.T) {
	cmd := &cobra.Command{}

	useNewMsgOptsFlags(cmd, false)

	// Check message opts flags
	contextFlag := cmd.Flags().Lookup(fnContext)
	assert.NotNil(t, contextFlag)

	// Check new flags
	broadcastFlag := cmd.Flags().Lookup(fnBroadcast)
	assert.NotNil(t, broadcastFlag)
	assert.Equal(t, "b", broadcastFlag.Shorthand)

	invokeFlag := cmd.Flags().Lookup(fnInvoke)
	assert.NotNil(t, invokeFlag)
	assert.Equal(t, "i", invokeFlag.Shorthand)
}
