// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/nunet/device-management-service/client"
	"gitlab.com/nunet/device-management-service/cmd/cli"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils/convert"
)

// onboardingInput used for command line onboarding parameters input
type onboardingInput struct {
	NoGPU    bool
	GPUsStr  string
	RAMSize  string
	DiskSize string
	CPUCores float32
	CPUCLock float64
	GPUs     types.GPUs
}

func processOnboardInput(ctx context.Context, dmsClient client.DmsClient, opts actorCmdOptions) error {
	p, ok := opts.Payload.(*onboardingInput)
	if !ok {
		return ErrInvalidArgument
	}

	r, err := dmsClient.HardwareSpec(ctx, opts.MsgOpts...)
	if err != nil || !r.OK {
		return fmt.Errorf("could not get machine resourcs: %w", err)
	}
	res := r.Resources

	// Set the CPU clock speed
	p.CPUCLock = res.CPU.ClockSpeed

	if p.NoGPU {
		fmt.Println("Skipping GPU selection.")
		return nil
	}

	// Handle GPU onboarding
	//
	// If no GPUs are found, skip GPU selection
	if len(res.GPUs) == 0 {
		fmt.Println("No usable GPUs detected; prerequisites may not be met. Skipping GPU selection.\n" +
			"Read more: https://gitlab.com/nunet/device-management-service#gpu-machines")
		return nil
	}

	// Check if GPUs are specified in the command line
	if p.GPUsStr != "" {
		p.GPUs, err = commandLineGPUOnboarding(res, p.GPUsStr, opts.Streams)
		if err != nil {
			return fmt.Errorf("onboard GPUs: %w", err)
		}

		return nil
	}

	// Interactive GPU onboarding
	r, err = dmsClient.HardwareUsage(ctx, opts.MsgOpts...)
	if err != nil || !r.OK {
		return fmt.Errorf("could not get machine resource usage: %w", err)
	}
	usage := r.Resources

	p.GPUs, err = interactiveGPUOnboarding(res, usage, opts.Streams)
	if err != nil {
		return fmt.Errorf("interactive GPU onboarding: %w", err)
	}
	return nil
}

// commandLineGPUOnboarding parses the GPU arguments from the command line and allocates VRAM for each selected GPU
// The GPU arguments are in the format "index:VRAM,index:VRAM,..."
func commandLineGPUOnboarding(machineResources types.Resources, gpuArgs string, _ cli.Streams) (types.GPUs, error) {
	var gpus types.GPUs
	gpuIndices := strings.Split(gpuArgs, ",")
	for _, gpuIndex := range gpuIndices {
		gpuIndexSplit := strings.Split(gpuIndex, ":")
		if len(gpuIndexSplit) != 2 {
			return nil, fmt.Errorf("invalid GPU format: %s", gpuIndex)
		}

		index, err := strconv.Atoi(gpuIndexSplit[0])
		if err != nil {
			return nil, fmt.Errorf("invalid GPU index: %w", err)
		}

		gpu, err := machineResources.GPUs.GetWithIndex(index)
		if err != nil {
			return nil, fmt.Errorf("invalid GPU index: %w", err)
		}

		gpu.VRAM, err = convert.ParseBytesWithDefaultUnit(gpuIndexSplit[1], "GiB")
		if err != nil {
			return nil, fmt.Errorf("invalid GPU VRAM: %w", err)
		}
		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// interactiveGPUOnboarding prompts the user to select GPUs and allocate VRAM for each selected GPU
func interactiveGPUOnboarding(machineResources types.Resources, machineResourceUsage types.Resources, streams cli.Streams) (types.GPUs, error) {
	var (
		gpuMap         = make(map[string]types.GPU)
		gpuPromptItems = make([]*selectPromptItem, 0, len(machineResources.GPUs))
		selectedGPUs   types.GPUs
	)
	for _, gpu := range machineResources.GPUs {
		gpuMap[gpu.Model] = gpu
		gpuPromptItems = append(gpuPromptItems, &selectPromptItem{
			Label: gpu.Model,
		})
	}

	// Prompt for GPU selection
	res, err := selectPromptMultiple("Select GPU", gpuPromptItems, streams)
	if err != nil {
		return nil, fmt.Errorf("could not select GPU: %w", err)
	}

	// Validate VRAM input
	vramValidator := func(input string) error {
		if _, err := strconv.ParseFloat(input, 64); err != nil {
			return fmt.Errorf("invalid input: %w", err)
		}
		return nil
	}

	// Update the VRAM allocation for each selected GPU
	for _, gpuName := range res {
		gpu := gpuMap[gpuName]
		fmt.Printf("-----------------------------------\n")
		fmt.Printf("Selected GPU: %s\n", gpuName)
		fmt.Printf("Total VRAM: %d GB\n", gpu.VRAMInGB())
		gpuUsage, err := machineResourceUsage.GPUs.GetWithIndex(gpu.Index)
		if err != nil {
			return nil, fmt.Errorf("could not get GPU usage: %w", err)
		}
		fmt.Printf("Used VRAM: %d GB\n", gpuUsage.VRAMInGB())
		fmt.Printf("Available VRAM: %d GB\n", types.ConvertBytesToGB(gpu.VRAM-gpuUsage.VRAM))

		// Prompt for VRAM allocation
		input, err := prompt("Enter new VRAM allocation in GB", vramValidator, streams)
		if err != nil {
			return nil, fmt.Errorf("could not prompt for VRAM: %w", err)
		}
		fmt.Println("-----------------------------------")

		// Update the GPU with the new VRAM allocation
		vram, err := strconv.ParseUint(input, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse VRAM: %w", err)
		}
		gpu.VRAM = types.ConvertGBToBytes(vram)
		selectedGPUs = append(selectedGPUs, gpu)
	}

	return selectedGPUs, nil
}
