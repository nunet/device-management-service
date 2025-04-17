// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"archive/tar"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"gitlab.com/nunet/device-management-service/types"
)

func TestRandomString(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		numChars int
		expErr   string
	}{
		"generate with zero characters": {
			numChars: 0,
		},
		"generate with two characters": {
			numChars: 2,
		},
		"generate with 1024 characters": {
			numChars: 1024,
		},
	}

	for name, tt := range cases {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			chars, err := RandomString(tt.numChars)
			if tt.expErr != "" {
				assert.EqualError(t, err, tt.expErr)
			} else {
				assert.Len(t, chars, tt.numChars)
			}
		})
	}
}

func TestIsExecutorStrictlyContained(t *testing.T) {
	tests := []struct {
		name       string
		leftSlice  []interface{}
		rightSlice []interface{}
		expected   bool
	}{
		{
			name:       "left contains right (multiple elements)",
			leftSlice:  []interface{}{types.ExecutorTypeDocker, types.ExecutorTypeFirecracker, types.ExecutorTypeWasm},
			rightSlice: []interface{}{types.ExecutorTypeDocker, types.ExecutorTypeFirecracker},
			expected:   true,
		},
		{
			name:       "left contains right (single element)",
			leftSlice:  []interface{}{types.ExecutorTypeDocker, types.ExecutorTypeFirecracker},
			rightSlice: []interface{}{types.ExecutorTypeDocker},
			expected:   true,
		},
		{
			name:       "right contains elements not in left",
			leftSlice:  []interface{}{types.ExecutorTypeDocker, types.ExecutorTypeFirecracker},
			rightSlice: []interface{}{types.ExecutorTypeDocker, types.ExecutorTypeFirecracker, types.ExecutorTypeWasm},
			expected:   false,
		},
		{
			name:       "left equals right",
			leftSlice:  []interface{}{types.ExecutorTypeDocker, types.ExecutorTypeFirecracker},
			rightSlice: []interface{}{types.ExecutorTypeDocker, types.ExecutorTypeFirecracker},
			expected:   true,
		},
		{
			name:       "empty right slice",
			leftSlice:  []interface{}{types.ExecutorTypeDocker, types.ExecutorTypeFirecracker},
			rightSlice: []interface{}{},
			expected:   false, // Default is false when rightSlice is empty
		},
		{
			name:       "empty left slice",
			leftSlice:  []interface{}{},
			rightSlice: []interface{}{types.ExecutorTypeDocker},
			expected:   false,
		},
		{
			name:       "different element types",
			leftSlice:  []interface{}{types.ExecutorTypeDocker, types.ExecutorTypeFirecracker, "string"},
			rightSlice: []interface{}{types.ExecutorTypeDocker, 123},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStrictlyContained(tt.leftSlice, tt.rightSlice)
			assert.Equal(t, tt.expected, result, fmt.Sprintf("IsStrictlyContained(%v, %v)", tt.leftSlice, tt.rightSlice))
		})
	}
}

func TestIsSameShallowType(t *testing.T) {
	// Setup test values
	mapStringInt1 := map[string]int{"1": 1, "2": 2, "3": 3}
	mapStringInt2 := map[string]int{"1": 5, "2": 6, "3": 7}

	mapStringInterface1 := map[string]interface{}{"1": 1, "2": 3, "3": 3}
	mapStringInterface2 := map[string]interface{}{"1": 5, "2": 6, "3": 7}

	str1 := "string"
	str2 := "another string"

	float1 := float32(6.629)
	float2 := float32(7.9790)

	int1 := 42
	int2 := 100

	slice1 := []int{1, 2, 3}
	slice2 := []int{4, 5, 6}

	array1 := [3]int{1, 2, 3}
	array2 := [3]int{4, 5, 6}

	struct1 := struct{ Name string }{"test"}
	struct2 := struct{ Name string }{"other"}

	var nilValue1, nilValue2 interface{} = nil, nil

	tests := []struct {
		name     string
		a        interface{}
		b        interface{}
		expected bool
	}{
		{
			name:     "same type - map[string]int",
			a:        mapStringInt1,
			b:        mapStringInt2,
			expected: true,
		},
		{
			name:     "same type - map[string]interface{}",
			a:        mapStringInterface1,
			b:        mapStringInterface2,
			expected: true,
		},
		{
			name:     "same type - string",
			a:        str1,
			b:        str2,
			expected: true,
		},
		{
			name:     "same type - float32",
			a:        float1,
			b:        float2,
			expected: true,
		},
		{
			name:     "same type - int",
			a:        int1,
			b:        int2,
			expected: true,
		},
		{
			name:     "same type - slice",
			a:        slice1,
			b:        slice2,
			expected: true,
		},
		{
			name:     "same type - array",
			a:        array1,
			b:        array2,
			expected: true,
		},
		{
			name:     "same type - struct",
			a:        struct1,
			b:        struct2,
			expected: true,
		},
		{
			name:     "same type - nil",
			a:        nilValue1,
			b:        nilValue2,
			expected: true,
		},
		{
			name:     "different types - map vs string",
			a:        mapStringInt1,
			b:        str1,
			expected: false,
		},
		{
			name:     "different types - string vs float",
			a:        str1,
			b:        float1,
			expected: false,
		},
		{
			name:     "different types - int vs slice",
			a:        int1,
			b:        slice1,
			expected: false,
		},
		{
			name:     "different types - slice vs array",
			a:        slice1,
			b:        array1,
			expected: false,
		},
		{
			name:     "different types - struct vs map",
			a:        struct1,
			b:        mapStringInt1,
			expected: false,
		},
		{
			name:     "different types - nil vs non-nil",
			a:        nilValue1,
			b:        int1,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSameShallowType(tt.a, tt.b)
			assert.Equal(t, tt.expected, result, "IsSameShallowType(%T, %T)", tt.a, tt.b)
		})
	}
}

func TestConvertTypedSliceToUntypedSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected []interface{}
	}{
		{
			name:     "JobTypes with Batch and SingleRun",
			input:    types.JobTypes{types.Batch, types.SingleRun},
			expected: []interface{}{types.Batch, types.SingleRun},
		},
		{
			name:     "JobTypes with Batch and LongRunning",
			input:    types.JobTypes{types.Batch, types.LongRunning},
			expected: []interface{}{types.Batch, types.LongRunning},
		},
		{
			name:     "JobTypes with Recurring and SingleRun",
			input:    types.JobTypes{types.Recurring, types.SingleRun},
			expected: []interface{}{types.Recurring, types.SingleRun},
		},
		{
			name:     "Empty JobTypes",
			input:    types.JobTypes{},
			expected: []interface{}{},
		},
		{
			name:     "String slice",
			input:    []string{"a", "b", "c"},
			expected: []interface{}{"a", "b", "c"},
		},
		{
			name:     "Int slice",
			input:    []int{1, 2, 3},
			expected: []interface{}{1, 2, 3},
		},
		{
			name:     "Mixed interface slice",
			input:    []interface{}{1, "string", true, 3.14},
			expected: []interface{}{1, "string", true, 3.14},
		},
		{
			name:     "Non-slice input",
			input:    "not a slice",
			expected: nil,
		},
		{
			name:     "Nil input",
			input:    nil,
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertTypedSliceToUntypedSlice(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSliceContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		str      string
		expected bool
	}{
		{
			name:     "empty slice",
			slice:    []string{},
			str:      "test",
			expected: false,
		},
		{
			name:     "slice contains string",
			slice:    []string{"test", "example", "sample"},
			str:      "example",
			expected: true,
		},
		{
			name:     "slice does not contain string",
			slice:    []string{"test", "example", "sample"},
			str:      "missing",
			expected: false,
		},
		{
			name:     "case sensitive match",
			slice:    []string{"Test", "Example", "Sample"},
			str:      "test",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SliceContains(tt.slice, tt.str)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsStrictlyContainedInt(t *testing.T) {
	tests := []struct {
		name       string
		leftSlice  []int
		rightSlice []int
		expected   bool
	}{
		{
			name:       "left contains right",
			leftSlice:  []int{1, 2, 3, 4, 5},
			rightSlice: []int{2, 3},
			expected:   true,
		},
		{
			name:       "left equals right",
			leftSlice:  []int{1, 2, 3},
			rightSlice: []int{1, 2, 3},
			expected:   true,
		},
		{
			name:       "left does not contain right",
			leftSlice:  []int{1, 2, 3},
			rightSlice: []int{3, 4, 5},
			expected:   false,
		},
		{
			name:       "empty right slice",
			leftSlice:  []int{1, 2, 3},
			rightSlice: []int{},
			expected:   false, // Default is false when rightSlice is empty
		},
		{
			name:       "empty left slice",
			leftSlice:  []int{},
			rightSlice: []int{1, 2},
			expected:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsStrictlyContainedInt(tt.leftSlice, tt.rightSlice)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMapKeysToSlice(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]int
		expected []string
	}{
		{
			name:     "empty map",
			input:    map[string]int{},
			expected: []string{},
		},
		{
			name: "map with keys",
			input: map[string]int{
				"one":   1,
				"two":   2,
				"three": 3,
			},
			expected: []string{"one", "two", "three"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MapKeysToSlice(tt.input)
			// Sort both slices to ensure consistent comparison
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestCreateTarArchive(t *testing.T) {
	// Create an in-memory filesystem
	memFS := afero.NewMemMapFs()
	afs := afero.Afero{Fs: memFS}

	// Setup test directory structure
	testDir := "/test-dir"
	err := afs.MkdirAll(testDir, 0o755)
	assert.NoError(t, err)

	// Create test files with content
	files := map[string]string{
		"/test-dir/file1.txt":        "This is file 1 content",
		"/test-dir/file2.txt":        "This is file 2 content",
		"/test-dir/subdir/file3.txt": "This is file 3 in subdirectory",
	}

	// Create the files in the in-memory filesystem
	for path, content := range files {
		// Ensure parent directory exists
		dir := filepath.Dir(path)
		err := afs.MkdirAll(dir, 0o755)
		assert.NoError(t, err)

		// Create and write to the file
		err = afs.WriteFile(path, []byte(content), 0o644)
		assert.NoError(t, err)
	}

	// Target tar file
	tarFile := "/output.tar"

	// Call the function to create tar archive
	err = CreateTarArchive(afs, testDir, tarFile)
	assert.NoError(t, err)

	// Verify the tar file was created
	exists, err := afs.Exists(tarFile)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Open and verify the contents of the tar file
	file, err := afs.Open(tarFile)
	assert.NoError(t, err)
	defer file.Close()

	// Read the tar archive
	tr := tar.NewReader(file)

	// Map to track which files we've found
	foundFiles := make(map[string]bool)

	// Iterate through the tar archive
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		assert.NoError(t, err)

		// Read the file content
		var buf strings.Builder
		var gotErr error
		for {
			_, err := io.CopyN(&buf, tr, 1024)
			if err == io.EOF {
				break
			}
			assert.NoError(t, gotErr)
		}

		// Check if this is one of our expected files
		expectedPath, err := SanitizeArchivePath("/", header.Name)
		assert.NoError(t, err)

		expectedContent, exists := files[expectedPath]

		if exists {
			// Verify the content matches
			assert.Equal(t, expectedContent, buf.String())
			foundFiles[expectedPath] = true
		}
	}

	// Verify all expected files were found in the archive
	for path := range files {
		assert.True(t, foundFiles[path], "File not found in archive: %s", path)
	}
}
