package cmd

import (
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"

	"gitlab.com/nunet/device-management-service/cmd/utils"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/types"
	dmsUtils "gitlab.com/nunet/device-management-service/utils"
)

const (
	containerPath    = "maint-scripts/install_container_runtime"
	amdDriverPath    = "maint-scripts/install_amd_drivers"
	nvidiaDriverPath = "maint-scripts/install_nvidia_drivers"
)

func newGPUOnboardCmd(afs afero.Afero) *cobra.Command {
	return &cobra.Command{
		Use:   "onboard",
		Short: "Install GPU drivers and Container Runtime",
		Long:  ``,
		RunE: func(cmd *cobra.Command, _ []string) error {
			wsl, err := dmsUtils.CheckWSL(afs)
			if err != nil {
				return fmt.Errorf("could not check WSL: %w", err)
			}

			mining, err := checkMiningOS(afs)
			if err != nil {
				return fmt.Errorf("couldn't check if Mining OS: %w", err)
			}

			machineResources, err := resources.ManagerInstance.SystemSpecs().GetMachineResources()
			if err != nil {
				return fmt.Errorf("get machine resources: %v", err)
			}

			var nvidiaGPUs, amdGPUs, intelGPUs []types.GPU
			for _, gpu := range machineResources.GPUs {
				switch gpu.Vendor {
				case types.GPUVendorNvidia:
					nvidiaGPUs = append(nvidiaGPUs, gpu)
				case types.GPUVendorAMDATI:
					amdGPUs = append(amdGPUs, gpu)
				case types.GPUVendorIntel:
					intelGPUs = append(intelGPUs, gpu)
				}
			}

			if len(nvidiaGPUs) == 0 && len(amdGPUs) == 0 && len(intelGPUs) == 0 {
				return fmt.Errorf("no NVIDIA/AMD/Intel GPU(s) detected")
			}

			switch {
			case wsl:
				fmt.Fprintf(cmd.OutOrStdout(), "You are running on Windows Subsystem for Linux (WSL). AMD GPUs are not supported.")

				if len(nvidiaGPUs) == 0 {
					return fmt.Errorf("no NVIDIA GPU(s) detected")
				}

				if err := promptContainer(cmd.InOrStdin(), cmd.OutOrStdout(), containerPath); err != nil {
					return fmt.Errorf("couldn't install container runtime: %w", err)
				}

			case mining:
				fmt.Fprintf(cmd.OutOrStdout(), "You are likely running a Mining OS. Skipping driver installation...")

				if err := promptContainer(cmd.InOrStdin(), cmd.OutOrStdout(), containerPath); err != nil {
					return fmt.Errorf("couldn't install container runtime: %w", err)
				}

			default:
				if len(nvidiaGPUs) > 0 {
					printGPUs(nvidiaGPUs)

					if err := promptContainer(cmd.InOrStdin(), cmd.OutOrStdout(), containerPath); err != nil {
						return fmt.Errorf("couldn't install container runtime: %w", err)
					}

					if err := promptDriverInstallation(cmd.InOrStdin(), cmd.OutOrStdout(), types.GPUVendorNvidia, nvidiaDriverPath); err != nil {
						return fmt.Errorf("couldn't install Nvidia driver: %w", err)
					}
				}

				if len(amdGPUs) > 0 {
					printGPUs(amdGPUs)

					if err := promptDriverInstallation(cmd.InOrStdin(), cmd.OutOrStdout(), types.GPUVendorAMDATI, amdDriverPath); err != nil {
						return fmt.Errorf("couldn't install AMD driver: %w", err)
					}
				}
			}
			return nil
		},
	}
}

// runScript executes a bash script from a given path.
// It takes the script's path as input and tries to run it, if successful it prints the output.
func runScript(scriptPath string) error {
	script := exec.Command("/bin/bash", scriptPath)

	output, err := script.CombinedOutput()
	if err != nil {
		return fmt.Errorf("script failed with error: %w", err)
	}

	fmt.Printf("%s\n", output)

	return nil
}

// promptContainer takes container runtime script path as input and prompts the user for confirmation.
// If confirmed, it runs the script.
func promptContainer(in io.Reader, out io.Writer, containerPath string) error {
	proceed, err := utils.PromptYesNo(in, out, "Do you want to proceed with Container Runtime installation? (y/N)")
	if err != nil {
		return fmt.Errorf("could not read answer from prompt: %w", err)
	}

	if proceed {
		err := runScript(containerPath)
		if err != nil {
			return fmt.Errorf("cannot run container runtime installation script: %w", err)
		}
	}

	return nil
}

// promptDriverInstallation takes GPUVendor (for printing) and the installation script as inputs.
// It prompts the user for confirmation and if confirmed it runs the script.
func promptDriverInstallation(in io.Reader, out io.Writer, vendor types.GPUVendor, scriptPath string) error {
	prompt := fmt.Sprintf("Do you want to proceed with %s driver installation? (y/N)", vendor)

	proceed, err := utils.PromptYesNo(in, out, prompt)
	if err != nil {
		return fmt.Errorf("could not read answer from prompt: %w", err)
	}

	if proceed {
		err := runScript(scriptPath)
		if err != nil {
			return fmt.Errorf("cannot run driver installation script: %w", err)
		}
	}

	return nil
}

// printGPUs display a list of detected GPUs in the machine.
// It takes a slice of GPUInfo structs as input, get the vendor from the first element
// and then iterate over each element to display the GPU card series.
func printGPUs(gpus []types.GPU) {
	var vendor string

	if len(gpus) == 0 {
		return
	}

	vendor = string(gpus[0].Vendor)

	fmt.Printf("Available %s GPU(s):", vendor)

	for _, gpu := range gpus {
		fmt.Printf("- %s\n", gpu.Model)
	}
}

// checkMiningOS detects if host is running a mining OS.
// It reads from /etc/os-release file and look for common distros inside of it, if any is found it returns true.
func checkMiningOS(afs afero.Afero) (bool, error) {
	miningOSes := []string{"Hive", "Rave", "PiMP", "Minerstat", "SimpleMining", "NH", "Miner", "SM", "MMP"}
	osFile := "/etc/os-release"

	info, err := afs.ReadFile(osFile)
	if err != nil {
		return false, fmt.Errorf("cannot read file %s: %w", osFile, err)
	}

	infoStr := string(info)
	for _, os := range miningOSes {
		if strings.Contains(infoStr, os) {
			return true, nil
		}
	}

	return false, nil
}
