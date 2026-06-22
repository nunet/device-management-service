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
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"

	"github.com/spf13/afero"
)

// RandomString generates a random string of length n
func RandomString(n int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	sb := strings.Builder{}
	sb.Grow(n)
	for i := 0; i < n; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			return "", err
		}

		sb.WriteByte(charset[n.Int64()])
	}
	return sb.String(), nil
}

// SliceContains checks if a string exists in a slice
func SliceContains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// IsStrictlyContained checks if all elements of rightSlice are contained in leftSlice
func IsStrictlyContained(leftSlice, rightSlice []interface{}) bool {
	result := false // the default result is false
	for _, subElement := range rightSlice {
		if !slices.Contains(leftSlice, subElement) {
			result = false
			break
		}
		result = true
	}
	return result
}

// IsStrictlyContainedInt checks if all elements of rightSlice are contained in leftSlice
func IsStrictlyContainedInt(leftSlice, rightSlice []int) bool {
	result := false // the default result is false
	for _, subElement := range rightSlice {
		if !slices.Contains(leftSlice, subElement) {
			result = false
			break
		}
		result = true
	}
	return result
}

// Sanitize archive file pathing from "G305: Zip Slip vulnerability"
func SanitizeArchivePath(d, t string) (v string, err error) {
	v = filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	return "", fmt.Errorf("%s: %s", "content filepath is tainted", t)
}

// CreateTarArchive creates a tar archive of the source directory
func CreateTarArchive(afs afero.Afero, sourceDir, targetFile string) error {
	tarFile, err := afs.Create(targetFile)
	if err != nil {
		return fmt.Errorf("failed to create tar file: %w", err)
	}
	defer tarFile.Close()

	tw := tar.NewWriter(tarFile)
	defer tw.Close()

	// Walk through the source directory
	return afs.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get the relative path
		relPath, err := filepath.Rel(filepath.Dir(sourceDir), path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		// Skip the source directory itself
		if path == sourceDir {
			return nil
		}

		// Create a tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		// Set the name to the relative path
		header.Name = relPath

		// Write the header
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// If it's a regular file, write the content
		if info.Mode().IsRegular() {
			file, err := afs.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file: %w", err)
			}
			defer file.Close()

			// copy in chunks to avoid decompression bomb
			for {
				_, err := io.CopyN(tw, file, 1024)
				if err != nil {
					if err == io.EOF {
						break
					}
					return fmt.Errorf("failed to copy file content to tar: %w", err)
				}
			}
		}

		return nil
	})
}

func IsSameShallowType(a, b interface{}) bool {
	aType := reflect.TypeOf(a)
	bType := reflect.TypeOf(b)
	result := aType == bType
	return result
}

func ConvertTypedSliceToUntypedSlice(typedSlice interface{}) []interface{} {
	s := reflect.ValueOf(typedSlice)
	if s.Kind() != reflect.Slice {
		return nil
	}
	result := make([]interface{}, s.Len())
	for i := 0; i < s.Len(); i++ {
		result[i] = s.Index(i).Interface()
	}
	return result
}

func MapKeysToSlice[R comparable, T any](m map[R]T) []R {
	keys := make([]R, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
