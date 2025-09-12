// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package ensemblev1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

func TestFormatSpec(t *testing.T) {
	t.Parallel()

	t.Run("invalid when data is non-map", func(t *testing.T) {
		t.Parallel()
		_, err := FormatSpec(nil, 123, tree.NewPath())
		assert.Error(t, err)
	})

	t.Run("invalid when v1 is missing", func(t *testing.T) {
		t.Parallel()
		_, err := FormatSpec(nil, map[string]any{}, tree.NewPath())
		assert.Error(t, err)
	})

	t.Run("invalid when v1 is not a map", func(t *testing.T) {
		t.Parallel()
		_, err := FormatSpec(nil, map[string]any{"v1": "oops"}, tree.NewPath())
		assert.Error(t, err)
	})

	t.Run("success returns v1 payload and sets version", func(t *testing.T) {
		t.Parallel()
		data := map[string]any{
			"v1": map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{"type": "service"},
				},
			},
		}

		res, err := FormatSpec(nil, data, tree.NewPath())
		assert.NoError(t, err)

		m, ok := res.(map[string]any)
		if assert.True(t, ok, "expected map result") {
			assert.Equal(t, "v1", m["version"])
			assert.NotNil(t, m["allocations"]) // returns inner `v1` map
		}
	})
}

func TestNewEnsemblev1Encoder_TransformsSpec(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"v1": map[string]any{
			"allocations": map[string]any{
				"alloc1": map[string]any{
					"execution": map[string]any{
						"type": "container",
						"params": map[string]any{
							"image":   "busybox",
							"command": []string{"sh", "-c", "echo hi"},
						},
					},
					"volumes": []any{
						map[string]any{
							"name": "data",
							"remote": map[string]any{
								"type":   "s3",
								"params": map[string]any{"bucket": "b1", "region": "eu"},
							},
						},
						map[string]any{
							"name": "cache",
							"remote": map[string]any{
								"type":   "sftp",
								"params": map[string]any{"host": "h", "path": "p"},
							},
						},
						map[string]any{
							// missing name -> will become volume_3
							"remote": map[string]any{
								"type":   "s3",
								"params": map[string]any{"bucket": "b2"},
							},
						},
					},
				},
				"alloc2": map[string]any{
					// ensure default naming for single unnamed volume
					"volumes": []any{
						map[string]any{
							"remote": map[string]any{
								"type":   "s3",
								"params": map[string]any{"bucket": "only"},
							},
						},
					},
				},
			},
			"nodes": map[string]any{},
		},
	}

	enc := NewEnsemblev1Encoder()
	got, err := enc.Transform(&raw)
	assert.NoError(t, err)

	root, ok := got.(map[string]any)
	if !assert.True(t, ok, "root should be a map") {
		return
	}

	// Top-level expectations
	assert.Equal(t, "v1", root["version"]) // set by FormatSpec
	_, hasV1 := root["v1"]
	assert.False(t, hasV1, "root should be replaced by inner v1 content")

	allocs, ok := root["allocations"].(map[string]any)
	if !assert.True(t, ok) {
		return
	}

	// alloc1 execution flatten
	alloc1, ok := allocs["alloc1"].(map[string]any)
	if !assert.True(t, ok) {
		return
	}

	exec, ok := alloc1["execution"].(map[string]any)
	if assert.True(t, ok) {
		assert.Equal(t, "container", exec["type"]) // kept
		assert.Equal(t, "busybox", exec["image"])  // flattened from params
		cmd, ok := exec["command"].([]string)
		if assert.True(t, ok) {
			assert.Equal(t, []string{"sh", "-c", "echo hi"}, cmd)
		}
	}

	// volumes slice -> map with name/default key and remote flatten
	vols, ok := alloc1["volumes"].(map[string]any)
	if assert.True(t, ok) {
		// named volume "data"
		dataV, ok := vols["data"].(map[string]any)
		if assert.True(t, ok) {
			remote, ok := dataV["remote"].(map[string]any)
			if assert.True(t, ok) {
				assert.Equal(t, "s3", remote["type"])   // kept
				assert.Equal(t, "b1", remote["bucket"]) // from params
				assert.Equal(t, "eu", remote["region"]) // from params
			}
		}

		// named volume "cache"
		cacheV, ok := vols["cache"].(map[string]any)
		if assert.True(t, ok) {
			remote, ok := cacheV["remote"].(map[string]any)
			if assert.True(t, ok) {
				assert.Equal(t, "sftp", remote["type"]) // kept
				assert.Equal(t, "h", remote["host"])    // from params
				assert.Equal(t, "p", remote["path"])    // from params
			}
		}

		// default-named volume
		v3, ok := vols["volumes_3"].(map[string]any)
		if assert.True(t, ok) {
			remote, ok := v3["remote"].(map[string]any)
			if assert.True(t, ok) {
				assert.Equal(t, "s3", remote["type"])   // kept
				assert.Equal(t, "b2", remote["bucket"]) // from params
			}
		}
	}

	// alloc2 volumes default naming starts from 1 within its own list
	alloc2, ok := allocs["alloc2"].(map[string]any)
	if assert.True(t, ok) {
		vols2, ok := alloc2["volumes"].(map[string]any)
		if assert.True(t, ok) {
			v1, ok := vols2["volumes_1"].(map[string]any)
			if assert.True(t, ok) {
				remote, ok := v1["remote"].(map[string]any)
				if assert.True(t, ok) {
					assert.Equal(t, "s3", remote["type"])     // kept
					assert.Equal(t, "only", remote["bucket"]) // from params
				}
			}
		}
	}
}
