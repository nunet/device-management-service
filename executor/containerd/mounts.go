// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package containerd

import (
	"fmt"

	specs "github.com/opencontainers/runtime-spec/specs-go"

	"gitlab.com/nunet/device-management-service/types"
)

// makeMounts builds OCI bind mounts from execution inputs and outputs
func makeMounts(
	inputs []*types.StorageVolumeExecutor,
	outputs []*types.StorageVolumeExecutor,
	resultsDir string,
) ([]specs.Mount, error) {
	mounts := make([]specs.Mount, 0, len(inputs)+len(outputs))

	for _, input := range inputs {
		mnt, err := bindMount(input)
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, mnt)
	}

	for _, output := range outputs {
		if output == nil {
			return nil, fmt.Errorf("output volume is nil")
		}
		if output.Source == "" {
			return nil, fmt.Errorf("output source is empty")
		}
		if resultsDir == "" {
			return nil, fmt.Errorf("results directory is empty")
		}

		mnt, err := bindMount(&types.StorageVolumeExecutor{
			Type:     output.Type,
			Source:   output.Source,
			Target:   output.Target,
			ReadOnly: false,
		})
		if err != nil {
			return nil, err
		}
		mounts = append(mounts, mnt)
	}

	return mounts, nil
}

func bindMount(vol *types.StorageVolumeExecutor) (specs.Mount, error) {
	if vol == nil {
		return specs.Mount{}, fmt.Errorf("volume is nil")
	}
	if vol.Target == "" {
		return specs.Mount{}, fmt.Errorf("volume target is empty")
	}

	opts := []string{"rbind"}
	if vol.ReadOnly {
		opts = append(opts, "ro")
	}

	return specs.Mount{
		Type:        "bind",
		Source:      vol.Source,
		Destination: vol.Target,
		Options:     opts,
	}, nil
}
