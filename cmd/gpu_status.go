package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"syscall"
	"time"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
	"gitlab.com/nunet/device-management-service/dms/resources"
	"gitlab.com/nunet/device-management-service/types"
)

var gpuStatusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Check GPU status in real time",
	Long:    ``,
	PreRunE: isDMSRunning(networkService),
	Run: func(cmd *cobra.Command, args []string) {
		vendors, err := resources.ManagerInstance.SystemSpecs().GetGPUVendors()
		if err != nil {
			fmt.Println("Error trying to detect GPU(s):", err)
			return
		}

		hasAMD := containsVendor(vendors, types.GPUVendorAMDATI)
		hasNVIDIA := containsVendor(vendors, types.GPUVendorNvidia)
		hasIntel := containsVendor(vendors, types.GPUVendorIntel)

		if hasNVIDIA || hasAMD || hasIntel {
			if hasNVIDIA {
				// NVML initialization
				retNVML := nvml.Init()
				if retNVML != nvml.SUCCESS {
					fmt.Println("Failed to initialize NVML:", nvml.ErrorString(retNVML))
				}
				defer func() {
					retNVML := nvml.Shutdown()
					if retNVML != nvml.SUCCESS {
						fmt.Println("Failed to shutdown NVML:", nvml.ErrorString(retNVML))
					}
				}()
			}

			countNVML, retNVML := nvml.DeviceGetCount()
			if retNVML != nvml.SUCCESS {
				fmt.Println("Failed to count NVIDIA GPU devices:", nvml.ErrorString(retNVML))
				countNVML = 0
			}

			countROCM, err := getCountAMD()
			if err != nil {
				fmt.Println("Failed to count AMD GPU devices:", err)
				countROCM = 0
			}

			countIntel, err := getCountIntel()
			if err != nil {
				fmt.Println("Failed to count Intel GPU devices:", err)
				countIntel = 0
			}

			// Initialize GPU slices
			nvidiaGPUs := make([]nvidiaGPU, countNVML)
			for i := 0; i < countNVML; i++ {
				nvidiaGPUs[i] = nvidiaGPU{index: i}
			}

			amdGPUs := make([]amdGPU, countROCM)
			for i := 0; i < countROCM; i++ {
				amdGPUs[i] = amdGPU{index: (i + 1)}
			}

			intelGPUs := make([]intelGPU, countIntel)
			for i := 0; i < countIntel; i++ {
				intelGPUs[i] = intelGPU{index: (i + 1)}
			}

			// Define channel for receiving interrupt signal and closing the real-time loop
			interrupt := make(chan os.Signal, 1)
			signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)
			exit := make(chan struct{})
			go func() {
				<-interrupt
				close(exit)
			}()

			for {
				select {
				case <-exit:
					fmt.Println("signal: interrupt")
					return
				default:
					// Clear screen (not reliable, maybe implement something ncurses-like for future)
					fmt.Print("\033[H\033[2J")

					fmt.Println("========== NuNet GPU Status ==========")

					fmt.Println("========== GPU Utilization ==========")
					for _, n := range nvidiaGPUs {
						fmt.Printf("%d %s: %d%%\n", n.index, n.name(), n.utilizationRate())
					}
					for _, a := range amdGPUs {
						fmt.Printf("%d AMD %s: %d%%\n", a.index, a.name(), a.utilizationRate())
					}
					for _, i := range intelGPUs {
						fmt.Printf("%d %s: %d%%\n", i.index, i.name(), i.utilizationRate())
					}

					fmt.Println("========== Memory Capacity ==========")
					for _, n := range nvidiaGPUs {
						fmt.Printf("%d %s: %s\n", n.index, n.name(), humanize.IBytes(n.memory().total))
					}
					for _, a := range amdGPUs {
						fmt.Printf("%d AMD %s: %s\n", a.index, a.name(), humanize.IBytes(a.memory().total))
					}
					for _, i := range intelGPUs {
						fmt.Printf("%d %s: %s\n", i.index, i.name(), humanize.IBytes(i.memory().total))
					}

					fmt.Println("========== Memory Used ==========")
					for _, n := range nvidiaGPUs {
						fmt.Printf("%d %s: %s\n", n.index, n.name(), humanize.IBytes(n.memory().used))
					}
					for _, a := range amdGPUs {
						fmt.Printf("%d AMD %s: %s\n", a.index, a.name(), humanize.IBytes(a.memory().used))
					}
					for _, i := range intelGPUs {
						fmt.Printf("%d %s: %s\n", i.index, i.name(), humanize.IBytes(i.memory().used))
					}

					fmt.Println("========== Memory Free ==========")
					for _, n := range nvidiaGPUs {
						fmt.Printf("%d %s: %s\n", n.index, n.name(), humanize.IBytes(n.memory().free))
					}
					for _, a := range amdGPUs {
						fmt.Printf("%d AMD %s: %s\n", a.index, a.name(), humanize.IBytes(a.memory().free))
					}
					for _, i := range intelGPUs {
						fmt.Printf("%d %s: %s\n", i.index, i.name(), humanize.IBytes(i.memory().free))
					}

					fmt.Println("========== Temperature ==========")
					for _, n := range nvidiaGPUs {
						fmt.Printf("%d %s: %.0f°C\n", n.index, n.name(), n.temperature())
					}
					for _, a := range amdGPUs {
						fmt.Printf("%d AMD %s: %.0f°C\n", a.index, a.name(), a.temperature())
					}

					fmt.Println("========== Power Usage ==========")
					for _, n := range nvidiaGPUs {
						fmt.Printf("%d %s: %dW\n", n.index, n.name(), n.powerUsage())
					}
					for _, a := range amdGPUs {
						fmt.Printf("%d AMD %s: %dW\n", a.index, a.name(), a.powerUsage())
					}
					for _, i := range intelGPUs {
						fmt.Printf("%d %s: %dW\n", i.index, i.name(), i.powerUsage())
					}

					fmt.Println("")
					fmt.Println("Press CTRL+C to exit...")
					fmt.Println("Refreshing status in a few seconds...")

					time.Sleep(2 * time.Second)
				}
			}
		} else {
			fmt.Println("No AMD, NVIDIA or Intel GPU(s) detected...")
			return
		}
	},
}

func runShellCmd(command string) (string, error) {
	cmd := exec.Command("sh", "-c", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("unable to get combined output from command %s: %v", command, err)
	}
	return string(output), nil
}

func getCountAMD() (int, error) {
	rocmOutput, err := runShellCmd("rocm-smi --showid")
	if err != nil {
		return 0, fmt.Errorf("cannot run shell command: %v", err)
	}

	pattern := `GPU\[(\d+)\]`
	re := regexp.MustCompile(pattern)

	matches := re.FindAllStringSubmatch(rocmOutput, -1)

	var ids []string
	for _, match := range matches {
		ids = append(ids, match[1])
	}

	return len(ids), nil
}

// GetCountIntel returns the number of discrete Intel GPUs
func getCountIntel() (int, error) {
	cmd := exec.Command("xpu-smi", "health", "-l")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("xpu-smi not installed, initialized, or configured: %s", err)
	}

	outputStr := string(output)

	// Use regex to find all instances of Device ID
	deviceIDRegex := regexp.MustCompile(`(?i)\| Device ID\s+\|\s+(\d+)\s+\|`)
	deviceIDMatches := deviceIDRegex.FindAllStringSubmatch(outputStr, -1)

	return len(deviceIDMatches), nil
}
