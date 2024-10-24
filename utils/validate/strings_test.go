// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package validate

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsLiteral(t *testing.T) {
	intValue := int(2)
	float32Value := float32(3.456)
	float64Value := float64(45.59736)

	stringValue := "some string"

	// negative assertions
	assert.False(t, IsLiteral(intValue))
	assert.False(t, IsLiteral(float32Value))
	assert.False(t, IsLiteral(float64Value))

	// negative assertions
	assert.True(t, IsLiteral(stringValue))
}

func TestIsBlank(t *testing.T) {
	// positive assertions
	assert.True(t, IsBlank("   "))
	assert.True(t, IsBlank(""))
	assert.True(t, IsBlank(" "))

	// negative assertions
	assert.False(t, IsBlank("  a  "))
	assert.False(t, IsBlank("a"))
}

func TestIsNotBlank(t *testing.T) {
	// positive assertions
	assert.True(t, IsNotBlank("  a  "))
	assert.True(t, IsNotBlank("a"))

	// negative assertions
	assert.False(t, IsNotBlank("   "))
	assert.False(t, IsNotBlank(""))
	assert.False(t, IsNotBlank(" "))
}
