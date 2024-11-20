// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalize(t *testing.T) {
	data := map[string]interface{}{
		"jobs": []map[string]interface{}{
			{"name": "job2"},
			{"name": "job1"},
		},
	}

	expected := map[string]interface{}{
		"jobs": []interface{}{
			map[string]interface{}{
				"name": "job1",
			},
			map[string]interface{}{
				"name": "job2",
			},
		},
	}

	result := Normalize(data)
	assert.Equal(t, expected, result)
}

func TestToAnySlice(t *testing.T) {
	data := []string{"a", "b", "c"}
	expected := []any{"a", "b", "c"}

	result, err := ToAnySlice(data)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestGetConfigAtPath(t *testing.T) {
	data := map[string]interface{}{
		"jobs": []map[string]interface{}{
			{"name": "job1"},
			{"name": "job2"},
		},
	}

	expected := map[string]interface{}{
		"name": "job1",
	}

	result, err := GetConfigAtPath(data, "jobs.[0]")
	assert.NoError(t, err)
	assert.Equal(t, expected, result)

	_, err = GetConfigAtPath(data, "jobs.[x]")
	assert.Error(t, err)
}
