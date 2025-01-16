// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsSameShallowType(t *testing.T) {
	v1 := make(map[string]int)
	v1["1"] = 1
	v1["2"] = 2
	v1["3"] = 3

	v4 := make(map[string]int)
	v4["1"] = 5
	v4["2"] = 6
	v4["3"] = 7

	v2 := "string"
	v5 := "another string"
	v3 := float32(6.629)
	v6 := float32(7.9790)

	v7 := make(map[string]interface{})
	v7["1"] = 1
	v7["2"] = 3
	v7["3"] = 3

	v8 := make(map[string]interface{})
	v8["1"] = 5
	v8["2"] = 6
	v8["3"] = 7

	assert.True(t, IsSameShallowType(v1, v4))
	assert.True(t, IsSameShallowType(v2, v5))
	assert.True(t, IsSameShallowType(v3, v6))
	assert.True(t, IsSameShallowType(v7, v8))

	assert.False(t, IsSameShallowType(v1, v2))
	assert.False(t, IsSameShallowType(v2, v3))
}

func TestIsExecutorStrictlyContained(t *testing.T) {
	executors1 := []interface{}{ExecutorTypeDocker, ExecutorTypeFirecracker, ExecutorTypeWasm}
	executors2 := []interface{}{ExecutorTypeDocker, ExecutorTypeFirecracker}
	executors3 := []interface{}{ExecutorTypeDocker}

	// possitive assertions
	assert.True(t,
		IsStrictlyContained(executors1, executors2),
		fmt.Sprintf("Executors %s strictly contains executors %s",
			executors1,
			executors2,
		),
	)
	assert.True(t,
		IsStrictlyContained(executors2, executors3),
		fmt.Sprintf("Executors %s strictly contains executors %s",
			executors2,
			executors3,
		),
	)

	// negative assertions
	assert.False(t,
		IsStrictlyContained(executors2, executors1),
		len(fmt.Sprintf("Executors %s strictly contains executors %s",
			executors2,
			executors1,
		)),
	)
}
