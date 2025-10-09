// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/spf13/afero"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/lib/env"
)

func displayResponse(w io.Writer, resp any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(resp)
}

func ProcessEnsembleYaml(fs afero.Afero, env env.EnvironmentProvider, path string) (
	*jobtypes.EnsembleConfig, error,
) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &jobtypes.EnsembleConfig{}
	err = parser.Decode(parser.SpecTypeEnsembleV1, data, &cfg, &parser.Options{
		Env:        env,
		Fs:         fs,
		WorkingDir: "",
	})
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
