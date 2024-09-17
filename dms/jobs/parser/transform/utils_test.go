package transform

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

func TestMapToSlice(t *testing.T) {
	data := map[string]interface{}{
		"job1": map[string]interface{}{"key": "value"},
		"job2": map[string]interface{}{"key": "value"},
	}

	expected := []map[string]interface{}{
		{"name": "job1", "key": "value"},
		{"name": "job2", "key": "value"},
	}

	result, err := MapToSlice(data)
	assert.NoError(t, err)
	assert.Equal(t, Normalize(expected), Normalize(result))
}
