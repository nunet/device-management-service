// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package actor

import (
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/nunet/device-management-service/dms/hardware"
	"gitlab.com/nunet/device-management-service/dms/node"
	"gitlab.com/nunet/device-management-service/types"
)

func onboardBehaviorPreRun(_ *Command, payload any) error {
	p, ok := payload.(*node.OnboardRequest)
	if !ok {
		return ErrInvalidArgument
	}

	hardwareManager := hardware.NewHardwareManager()
	machineResources, err := hardwareManager.GetMachineResources()
	if err != nil {
		return fmt.Errorf("could not get machine resources: %w", err)
	}

	// Set the CPU clock speed
	p.Config.OnboardedResources.CPU.ClockSpeed = machineResources.CPU.ClockSpeed

	if p.NoGPU {
		fmt.Println("Skipping GPU selection.")
		return nil
	}

	// Handle GPU onboarding
	//
	// If no GPUs are found, skip GPU selection
	if len(machineResources.GPUs) == 0 {
		fmt.Println("No usable GPUs detected; prerequisites may not be met. Skipping GPU selection.\n" +
			"Read more: https://gitlab.com/nunet/device-management-service#gpu-machines")
		return nil
	}

	// Check if GPUs are specified in the command line
	if p.GPUs != "" {
		p.Config.OnboardedResources.GPUs, err = commandLineGPUOnboarding(machineResources, p.GPUs)
		if err != nil {
			return fmt.Errorf("onboard GPUs: %w", err)
		}

		return nil
	}

	// Interactive GPU onboarding
	machineResourceUsage, err := hardwareManager.GetUsage()
	if err != nil {
		return fmt.Errorf("could not get machine resource usage: %w", err)
	}

	p.Config.OnboardedResources.GPUs, err = interactiveGPUOnboarding(machineResources, machineResourceUsage)
	if err != nil {
		return fmt.Errorf("interactive GPU onboarding: %w", err)
	}
	return nil
}

// commandLineGPUOnboarding parses the GPU arguments from the command line and allocates VRAM for each selected GPU
// The GPU arguments are in the format "index:VRAM,index:VRAM,..."
func commandLineGPUOnboarding(machineResources types.MachineResources, gpuArgs string) (types.GPUs, error) {
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

		vram, err := strconv.ParseFloat(gpuIndexSplit[1], 64)
		if err != nil {
			return nil, fmt.Errorf("invalid GPU VRAM: %w", err)
		}

		gpu, err := machineResources.GPUs.GetWithIndex(index)
		if err != nil {
			return nil, fmt.Errorf("invalid GPU index: %w", err)
		}

		gpu.VRAM = types.ConvertGBToBytes(vram)
		gpus = append(gpus, gpu)
	}

	return gpus, nil
}

// interactiveGPUOnboarding prompts the user to select GPUs and allocate VRAM for each selected GPU
func interactiveGPUOnboarding(machineResources types.MachineResources, machineResourceUsage types.Resources) (types.GPUs, error) {
	var (
		gpuMap         = make(map[string]types.GPU)
		gpuPromptItems = make([]*selectPromptItem, 0)
		selectedGPUs   types.GPUs
	)
	for _, gpu := range machineResources.GPUs {
		gpuMap[gpu.Model] = gpu
		gpuPromptItems = append(gpuPromptItems, &selectPromptItem{
			Label: gpu.Model,
		})
	}

	// Prompt for GPU selection
	res, err := selectPrompt("Select GPU", gpuPromptItems)
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
		fmt.Printf("Total VRAM: %.2f GB\n", gpu.VRAMInGB())
		gpuUsage, err := machineResourceUsage.GPUs.GetWithIndex(gpu.Index)
		if err != nil {
			return nil, fmt.Errorf("could not get GPU usage: %w", err)
		}
		fmt.Printf("Used VRAM: %.2f GB\n", gpuUsage.VRAMInGB())
		fmt.Printf("Available VRAM: %.2f GB\n", types.ConvertBytesToGB(gpu.VRAM-gpuUsage.VRAM))

		// Prompt for VRAM allocation
		input, err := prompt("Enter new VRAM allocation in GB", vramValidator)
		if err != nil {
			return nil, fmt.Errorf("could not prompt for VRAM: %w", err)
		}
		fmt.Println("-----------------------------------")

		// Update the GPU with the new VRAM allocation
		vram, err := strconv.ParseFloat(input, 64)
		if err != nil {
			return nil, fmt.Errorf("could not parse VRAM: %w", err)
		}
		gpu.VRAM = types.ConvertGBToBytes(vram)
		selectedGPUs = append(selectedGPUs, gpu)
	}

	return selectedGPUs, nil
}
