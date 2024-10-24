// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConvertNumericToFloat64(t *testing.T) {
	// positive examples
	f, ok := ConvertNumericToFloat64(10)
	assert.True(t, ok)
	assert.Equal(t, float64(10), f)

	f, ok = ConvertNumericToFloat64(int8(5))
	assert.True(t, ok)
	assert.Equal(t, float64(5), f)

	f, ok = ConvertNumericToFloat64(int16(100))
	assert.True(t, ok)
	assert.Equal(t, float64(100), f)

	f, ok = ConvertNumericToFloat64(int32(1000))
	assert.True(t, ok)
	assert.Equal(t, float64(1000), f)

	f, ok = ConvertNumericToFloat64(int64(10000))
	assert.True(t, ok)
	assert.Equal(t, float64(10000), f)

	f, ok = ConvertNumericToFloat64(uint(1))
	assert.True(t, ok)
	assert.Equal(t, float64(1), f)

	f, ok = ConvertNumericToFloat64(uint8(2))
	assert.True(t, ok)
	assert.Equal(t, float64(2), f)

	f, ok = ConvertNumericToFloat64(uint16(3))
	assert.True(t, ok)
	assert.Equal(t, float64(3), f)

	f, ok = ConvertNumericToFloat64(uint32(4))
	assert.True(t, ok)
	assert.Equal(t, float64(4), f)

	f, ok = ConvertNumericToFloat64(uint64(5))
	assert.True(t, ok)
	assert.Equal(t, float64(5), f)

	// this test passes when there are not decimal places
	// but fails with decimal places -- see corresponding negative example
	f, ok = ConvertNumericToFloat64(float32(3))
	assert.True(t, ok)
	assert.Equal(t, float64(3), f)

	f, ok = ConvertNumericToFloat64(float64(2.718))
	assert.True(t, ok)
	assert.Equal(t, float64(2.718), f)

	// negative examples
	f, ok = ConvertNumericToFloat64("test")
	assert.False(t, ok)
	assert.Equal(t, float64(0), f)

	f, ok = ConvertNumericToFloat64(true)
	assert.False(t, ok)
	assert.Equal(t, float64(0), f)

	f, ok = ConvertNumericToFloat64([]int{1, 2, 3})
	assert.False(t, ok)
	assert.Equal(t, float64(0), f)

	f, ok = ConvertNumericToFloat64(map[string]int{"a": 1, "b": 2})
	assert.False(t, ok)
	assert.Equal(t, float64(0), f)

	// interestingly, this is a negative example, with the actually converted value having
	// more decimal places than the expected value
	// however, for what this method is used it should not be a problem,
	// since both converted values are compared to each other
	// however, BEWARE of this behaviour
	f, ok = ConvertNumericToFloat64(float32(3.14))
	assert.True(t, ok)
	assert.NotEqual(t, float64(3.14), f)
}
