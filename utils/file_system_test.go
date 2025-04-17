// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetDirectorySize(t *testing.T) {
	tests := []struct {
		name          string
		setupFs       func(fs afero.Fs) error
		path          string
		expectedSize  int64
		expectedError bool
	}{
		{
			name: "empty directory",
			setupFs: func(fs afero.Fs) error {
				return fs.MkdirAll("testdir", os.ModePerm)
			},
			path:          "testdir",
			expectedSize:  0,
			expectedError: false,
		},
		{
			name: "directory with files",
			setupFs: func(fs afero.Fs) error {
				if err := fs.MkdirAll("testdir", os.ModePerm); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "testdir/file1.txt", []byte("hello"), os.ModePerm); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "testdir/file2.txt", []byte("world"), os.ModePerm); err != nil {
					return err
				}
				return nil
			},
			path:          "testdir",
			expectedSize:  10, // "hello" (5) + "world" (5) = 10 bytes
			expectedError: false,
		},
		{
			name: "nested directories",
			setupFs: func(fs afero.Fs) error {
				if err := fs.MkdirAll("testdir/subdir", os.ModePerm); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "testdir/file1.txt", []byte("hello"), os.ModePerm); err != nil {
					return err
				}
				if err := afero.WriteFile(fs, "testdir/subdir/file2.txt", []byte("world"), os.ModePerm); err != nil {
					return err
				}
				return nil
			},
			path:          "testdir",
			expectedSize:  10, // "hello" (5) + "world" (5) = 10 bytes
			expectedError: false,
		},
		{
			name: "non-existent directory",
			setupFs: func(_ afero.Fs) error {
				return nil
			},
			path:          "nonexistent",
			expectedSize:  0,
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			err := tt.setupFs(fs)
			require.NoError(t, err, "Failed to setup filesystem")

			size, err := GetDirectorySize(fs, tt.path)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedSize, size)
			}
		})
	}
}

func TestWriteToFile(t *testing.T) {
	tests := []struct {
		name          string
		data          []byte
		filePath      string
		setupFs       func(fs afero.Fs) error
		expectedPath  string
		expectedError bool
		validateFile  func(t *testing.T, fs afero.Fs, path string)
	}{
		{
			name:     "write to new file",
			data:     []byte("test data"),
			filePath: "test.txt",
			setupFs: func(_ afero.Fs) error {
				return nil
			},
			expectedPath:  "test.txt",
			expectedError: false,
			validateFile: func(t *testing.T, fs afero.Fs, path string) {
				content, err := afero.ReadFile(fs, path)
				assert.NoError(t, err)
				assert.Equal(t, []byte("test data"), content)
			},
		},
		{
			name:     "write to file in new directory",
			data:     []byte("test data"),
			filePath: "newdir/test.txt",
			setupFs: func(_ afero.Fs) error {
				return nil
			},
			expectedPath:  "newdir/test.txt",
			expectedError: false,
			validateFile: func(t *testing.T, fs afero.Fs, path string) {
				content, err := afero.ReadFile(fs, path)
				assert.NoError(t, err)
				assert.Equal(t, []byte("test data"), content)

				// Verify directory was created
				info, err := fs.Stat(filepath.Dir(path))
				assert.NoError(t, err)
				assert.True(t, info.IsDir())
			},
		},
		{
			name:     "overwrite existing file",
			data:     []byte("new data"),
			filePath: "existing.txt",
			setupFs: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "existing.txt", []byte("old data"), os.ModePerm)
			},
			expectedPath:  "existing.txt",
			expectedError: false,
			validateFile: func(t *testing.T, fs afero.Fs, path string) {
				content, err := afero.ReadFile(fs, path)
				assert.NoError(t, err)
				assert.Equal(t, []byte("new data"), content)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			err := tt.setupFs(fs)
			require.NoError(t, err, "Failed to setup filesystem")

			path, err := WriteToFile(fs, tt.data, tt.filePath)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPath, path)
				tt.validateFile(t, fs, path)
			}
		})
	}
}

func TestFileExists(t *testing.T) {
	tests := []struct {
		name           string
		setupFs        func(fs afero.Fs) error
		filename       string
		expectedExists bool
	}{
		{
			name: "file exists",
			setupFs: func(fs afero.Fs) error {
				return afero.WriteFile(fs, "test.txt", []byte("test"), os.ModePerm)
			},
			filename:       "test.txt",
			expectedExists: true,
		},
		{
			name: "file does not exist",
			setupFs: func(_ afero.Fs) error {
				return nil
			},
			filename:       "nonexistent.txt",
			expectedExists: false,
		},
		{
			name: "path is a directory",
			setupFs: func(fs afero.Fs) error {
				return fs.MkdirAll("testdir", os.ModePerm)
			},
			filename:       "testdir",
			expectedExists: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			err := tt.setupFs(fs)
			require.NoError(t, err, "Failed to setup filesystem")

			exists := FileExists(fs, tt.filename)
			assert.Equal(t, tt.expectedExists, exists)
		})
	}
}
