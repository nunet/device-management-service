// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package resolve

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/afero"
	"gitlab.com/nunet/device-management-service/lib/env"
)

type Handler interface {
	Resolve(path string) ([]byte, error)
}

// FileResolver implements FileResolver for the local filesystem.
type FileResolver struct {
	Fs         afero.Afero
	BasePath   string
	WorkingDir string
}

func NewFileResolver(fs afero.Fs, basePath string) Handler {
	return &FileResolver{Fs: afero.Afero{Fs: fs}, BasePath: basePath}
}

func (l *FileResolver) Resolve(path string) ([]byte, error) {
	joinedPath := filepath.Join(l.BasePath, path)
	content, err := l.Fs.ReadFile(joinedPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrNotExist
		}
		return nil, err
	}
	return content, nil
}

// EnvResolver implements EnvResolver for the environment variables
type EnvResolver struct {
	Env env.EnvironmentProvider
}

func NewEnvResolver(env env.EnvironmentProvider) Handler {
	return &EnvResolver{Env: env}
}

func (r *EnvResolver) Resolve(key string) ([]byte, error) {
	val := r.Env.Getenv(key)
	if val == "" {
		return nil, ErrNotExist
	}
	return []byte(val), nil
}

type Resolver struct {
	SourceHandlers  map[string]Handler
	expressionRegex *regexp.Regexp
}

func NewResolver(sourceHandlers map[string]Handler, expressionRegex *regexp.Regexp) *Resolver {
	if expressionRegex == nil {
		expressionRegex = regexp.MustCompile(`\${([^{}]+?)}`)
	}
	return &Resolver{
		SourceHandlers:  sourceHandlers,
		expressionRegex: expressionRegex,
	}
}

func (r *Resolver) Process(input string) (string, error) {
	result := []byte(input)
	for {
		matches := r.expressionRegex.FindSubmatchIndex(result)
		if matches == nil {
			break
		}

		expression := result[matches[0]:matches[1]]
		content := string(result[matches[2]:matches[3]])

		resolvedValue, err := r.resolveContent(content)
		if err != nil {
			return "", fmt.Errorf("failed to resolve expression '%s': %w", expression, err)
		}
		result = bytes.Replace(result, expression, resolvedValue, 1)
	}
	return string(result), nil
}

func (r *Resolver) resolveContent(content string) ([]byte, error) {
	parts := strings.SplitN(content, ":-", 2)
	mainPart := parts[0]
	var defaultValue []byte
	hasDefault := len(parts) > 1
	if hasDefault {
		defaultValue = []byte(parts[1])
	}

	modifierParts := strings.Split(mainPart, "|")
	sourceAndKey := modifierParts[0]

	source, key, found := strings.Cut(sourceAndKey, ":")
	if !found {
		// Default to 'env' source if no source is specified
		source = "env"
		key = sourceAndKey
	}

	handler, exists := r.SourceHandlers[source]
	if !exists {
		return nil, fmt.Errorf("unknown source: '%s'", source)
	}

	value, err := handler.Resolve(key)
	if err != nil {
		if hasDefault {
			return defaultValue, nil
		}
		return nil, err
	}
	return value, err
}
