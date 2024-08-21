package cmd

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils"
)

const (
	containerPath    = "maint-scripts/install_container_runtime"
	amdDriverPath    = "maint-scripts/install_amd_drivers"
	nvidiaDriverPath = "maint-scripts/install_nvidia_drivers"
)

var gpuOnboardCmd = &cobra.Command{
	Use:     "onboard",
	Short:   "Install GPU drivers and Container Runtime",
	Long:    ``,
	PreRunE: isDMSRunning(networkService),
	Run: func(cmd *cobra.Command, args []string) {
		wsl, err := utils.CheckWSL()
		if err != nil {
			zlog.Sugar().Errorf("Error checking WSL: %v", err)
			return
		}

		mining, err := checkMiningOS()
		if err != nil {
			zlog.Sugar().Errorf("Error checking Mining OS: %v", err)
			return
		}

		vendors, err := resources.ManagerInstance.SystemSpecs().GetGPUVendors()
		if err != nil {
			zlog.Sugar().Errorf("Error trying to detect GPU(s): %v", err)
			return
		}

		hasAMD := containsVendor(vendors, types.GPUVendorAMDATI)
		hasNVIDIA := containsVendor(vendors, types.GPUVendorNvidia)
		hasIntel := containsVendor(vendors, types.GPUVendorIntel)

		if !hasAMD && !hasNVIDIA && !hasIntel {
			zlog.Sugar().Error("No NVIDIA/AMD/Intel GPU(s) detected...")
			return
		}

		switch {
		case wsl:
			zlog.Warn("You are running on Windows Subsystem for Linux (WSL). AMD GPUs are not supported.")
			if !hasNVIDIA {
				zlog.Warn("No NVIDIA GPU(s) detected")
				return
			}

			if err := promptContainer(cmd.InOrStdin(), cmd.OutOrStdout(), containerPath); err != nil {
				fmt.Println("Error during Container Runtime installation:", err)
				return
			}

		case mining:
			zlog.Warn("You are likely running a Mining OS. Skipping driver installation...")

			if err := promptContainer(cmd.InOrStdin(), cmd.OutOrStdout(), containerPath); err != nil {
				zlog.Sugar().Errorf("Error during Container Runtime installation: %v", err)
				return
			}

		default:
			if hasNVIDIA {
				nvidiaGPUs, err := resources.ManagerInstance.SystemSpecs().GetGPUs(types.GPUVendorNvidia)
				if err != nil {
					fmt.Println("Error while fetching NVIDIA info:", err)
					return
				}

				printGPUs(nvidiaGPUs)

				if err := promptContainer(cmd.InOrStdin(), cmd.OutOrStdout(), containerPath); err != nil {
					zlog.Sugar().Errorf("Error during Container Runtime installation: %v", err)
					return
				}

				if err := promptDriverInstallation(cmd.InOrStdin(), cmd.OutOrStdout(), types.GPUVendorNvidia, nvidiaDriverPath); err != nil {
					zlog.Sugar().Errorf("Error during NVIDIA drivers installation: %v", err)
					return
				}
			}

			if hasAMD {
				AMDGPUs, err := resources.ManagerInstance.SystemSpecs().GetGPUs(types.GPUVendorAMDATI)
				if err != nil {
					zlog.Sugar().Errorf("Error while fetching AMD info: %v", err)
					return
				}

				printGPUs(AMDGPUs)

				if err := promptDriverInstallation(cmd.InOrStdin(), cmd.OutOrStdout(), types.GPUVendorAMDATI, amdDriverPath); err != nil {
					zlog.Sugar().Errorf("Error during AMD drivers installation: %v", err)
					return
				}
			}
		}
	},
}

// containsVendor takes a slice of GPUVendor structs that were detected in the system
// and look for a specific vendor, returning true if it is found.
func containsVendor(vendors []types.GPUVendor, target types.GPUVendor) bool {
	for _, v := range vendors {
		if v == target {
			return true
		}
	}

	return false
}

// runScript executes a bash script from a given path.
// It takes the script's path as input and tries to run it, if successfull it prints the output.
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
		return fmt.Errorf("could not read answer from prompt: %v", err)
	}

	if proceed {
		err := runScript(containerPath)
		if err != nil {
			return fmt.Errorf("cannot run container runtime installation script: %v", err)
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
		return fmt.Errorf("could not read answer from prompt: %v", err)
	}

	if proceed {
		err := runScript(scriptPath)
		if err != nil {
			return fmt.Errorf("cannot run driver installation script: %v", err)
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
func checkMiningOS() (bool, error) {
	miningOSes := []string{"Hive", "Rave", "PiMP", "Minerstat", "SimpleMining", "NH", "Miner", "SM", "MMP"}
	osFile := "/etc/os-release"

	info, err := os.ReadFile(osFile)
	if err != nil {
		return false, fmt.Errorf("cannot read file %s: %v", osFile, err)
	}

	infoStr := string(info)
	for _, os := range miningOSes {
		if strings.Contains(infoStr, os) {
			return true, nil
		}
	}

	return false, nil
}
