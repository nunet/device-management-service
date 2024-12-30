// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package cmd

import (
	"context"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/types"
	"go.uber.org/mock/gomock"
)

func TestGPUCommand(t *testing.T) {
	t.Parallel()

	t.Run("must be able to initialise the gpu command", func(t *testing.T) {
		t.Parallel()

		gpuCmd := newGPUCommand()
		require.NotNil(t, gpuCmd)
	})

	t.Run("must be able to list gpus", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		gpuManager := NewMockGPUManager(ctrl)
		gpuListCmd := newGPUListCommand(gpuManager)
		require.NotNil(t, gpuListCmd)

		gpus, gpuUsage := getMockGPUsAndUsage()
		gpuManager.EXPECT().GetGPUs().Return(gpus, nil).Times(1)
		gpuManager.EXPECT().GetGPUUsage().Return(gpuUsage, nil).Times(1)
		err := gpuListCmd.RunE(nil, nil)
		require.NoError(t, err)
	})

	t.Run("must be able to test gpu deployment", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		gpuManager := NewMockGPUManager(ctrl)
		dockerClient := NewMockClientInterface(ctrl)

		gpuTestCmd := newGPUTestCommand(gpuManager, dockerClient)
		require.NotNil(t, gpuTestCmd)

		gpus, _ := getMockGPUsAndUsage()
		gpuManager.EXPECT().GetGPUs().Return(gpus, nil).Times(1)
		dockerClient.EXPECT().IsInstalled(gomock.Any()).Return(true).Times(1)
		dockerClient.EXPECT().CreateContainer(gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
			gomock.Any(),
		).Return("mockContainerID", nil).Times(1)
		dockerClient.EXPECT().StartContainer(gomock.Any(), gomock.Any()).Return(nil).Times(1)
		statusCh := make(chan container.WaitResponse)
		errCh := make(chan error)
		dockerClient.EXPECT().WaitContainer(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, _ string) (<-chan container.WaitResponse, chan error) {
				go func() {
					time.Sleep(1 * time.Second)
					fmt.Println("Sending status")
					statusCh <- container.WaitResponse{
						StatusCode: 0,
					}
				}()
				return statusCh, errCh
			})
		dockerClient.EXPECT().GetOutputStream(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(dataToReadCloser("test data\n"), nil).Times(1)
		err := gpuTestCmd.RunE(nil, nil)
		require.NoError(t, err)
	})

	t.Run("must return error if there is no gpu", func(t *testing.T) {
		t.Parallel()

		ctrl := gomock.NewController(t)
		gpuManager := NewMockGPUManager(ctrl)
		dockerClient := NewMockClientInterface(ctrl)

		gpuTestCmd := newGPUTestCommand(gpuManager, dockerClient)
		require.NotNil(t, gpuTestCmd)

		gpuManager.EXPECT().GetGPUs().Return([]types.GPU{}, nil).Times(1)
		err := gpuTestCmd.RunE(nil, nil)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no GPUs found")
	})
}

func getMockGPUsAndUsage() (types.GPUs, types.GPUs) {
	gpus := []types.GPU{
		{
			Index:      0,
			Model:      "Tesla V100",
			Vendor:     types.GPUVendorAMDATI,
			VRAM:       16000000000,
			PCIAddress: "0000:00:00.0",
		},
		{
			Index:      1,
			Model:      "An AMD GPU",
			Vendor:     types.GPUVendorAMDATI,
			VRAM:       8000000000,
			PCIAddress: "0000:00:00.1",
		},
		{
			Index:      2,
			Model:      "An Intel GPU",
			Vendor:     types.GPUVendorIntel,
			VRAM:       4000000000,
			PCIAddress: "0000:00:00.2",
		},
	}
	gpuUsage := []types.GPU{
		{
			Index:      0,
			Model:      "Tesla V100",
			Vendor:     types.GPUVendorAMDATI,
			VRAM:       8000000000,
			PCIAddress: "0000:00:00.0",
		},
		{
			Index:      1,
			Model:      "An AMD GPU",
			Vendor:     types.GPUVendorAMDATI,
			VRAM:       4000000000,
			PCIAddress: "0000:00:00.1",
		},
		{
			Index:      2,
			Model:      "An Intel GPU",
			Vendor:     types.GPUVendorIntel,
			VRAM:       2000000000,
			PCIAddress: "0000:00:00.2",
		},
	}

	return gpus, gpuUsage
}

// dataToReadCloser converts a string into an io.ReadCloser
func dataToReadCloser(data string) io.ReadCloser {
	return io.NopCloser(strings.NewReader(data))
}
