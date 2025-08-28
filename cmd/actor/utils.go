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
)

func displayResponse(w io.Writer, resp any) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(resp)
}

func ProcessEnsembleYaml(fs afero.Afero, path string) (
	*jobtypes.EnsembleConfig, error,
) {
	data, err := fs.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &jobtypes.EnsembleConfig{}
	err = parser.Parse(parser.SpecTypeEnsembleV1, data, &cfg)
	if err != nil {
		return nil, err
	}

	for name, script := range cfg.V1.Scripts {
		scriptData, err := fs.ReadFile(string(script))
		if err != nil {
			return nil, fmt.Errorf("failed to read script file: %w", err)
		}
		cfg.V1.Scripts[name] = scriptData
	}

	for aIdx, alloc := range cfg.V1.Allocations {
		// handle reading public key files
		if alloc.Keys != nil {
			for kIdx, key := range alloc.Keys {
				keyData, err := fs.ReadFile(key.File)
				if err != nil {
					return nil, fmt.Errorf("failed to read key file: %w", err)
				}
				cfg.V1.Allocations[aIdx].Keys[kIdx].File = string(keyData)
			}
		}

		// handle reading client certificate files for volumes
		if len(alloc.Volume) > 0 {
			for i, v := range alloc.Volume {
				if v.Type != "glusterfs" {
					continue
				}

				if v.ClientPrivateKey != "" {
					pvkeyData, err := fs.ReadFile(v.ClientPrivateKey)
					if err != nil {
						return nil, fmt.Errorf("failed to read pvkeyData data: %w", err)
					}
					cfg.V1.Allocations[aIdx].Volume[i].ClientPrivateKey = string(pvkeyData)
				}

				if v.ClientPEM != "" {
					pemData, err := fs.ReadFile(v.ClientPEM)
					if err != nil {
						return nil, fmt.Errorf("failed to read pem data: %w", err)
					}
					cfg.V1.Allocations[aIdx].Volume[i].ClientPEM = string(pemData)
				}

				if v.ClientCA != "" {
					caData, err := fs.ReadFile(v.ClientCA)
					if err != nil {
						return nil, fmt.Errorf("failed to read ca data: %w", err)
					}
					cfg.V1.Allocations[aIdx].Volume[i].ClientCA = string(caData)
				}
			}
		}
	}

	return cfg, nil
}
