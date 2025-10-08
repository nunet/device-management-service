// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package dockercompose

import (
	"context"

	"github.com/compose-spec/compose-go/v2/loader"
	"github.com/compose-spec/compose-go/v2/types"
)

// Parse takes the content of a docker-compose.yml file and returns a parsed Project object.
func Parse(content []byte) (*types.Project, error) {
	configDetails := types.ConfigDetails{
		ConfigFiles: []types.ConfigFile{
			{
				Content: content,
			},
		},
	}

	// The loader.LoadWithContext function handles parsing and validation of the compose file.
	// NOTE: should we add context to the Parse function and pass it here?
	project, err := loader.LoadWithContext(context.Background(), configDetails, func(opts *loader.Options) { opts.SetProjectName("project", true) })
	if err != nil {
		return nil, err
	}

	return project, nil
}
