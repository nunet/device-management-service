// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package utils

import (
	"fmt"
	"strings"
	"time"

	incus "github.com/lxc/incus/client"
)

// DefaultDMSSuffix is the suffix added to DMS context names
const DefaultDMSSuffix = "-dms"

// TODO: Define `Instance` as interface and rename this to `IncusInstance`
// Instance represents a container instance.
type Instance struct {
	Name     string
	Client   incus.InstanceServer
	Contexts map[string]*Context
}

type NetInfo struct {
	NetworkName string
	IPAddress   string
}

// RunCMD runs a command inside the instance.
func (i *Instance) RunCMD(cmd []string) (string, error) {
	return RunCommandInInstance(i.Client, i.Name, cmd)
}

// RunCMDBackground runs a command inside the instance in the background.
func (i *Instance) RunCMDBackground(cmd []string) error {
	return RunBackgroundCommandInInstance(i.Client, i.Name, cmd)
}

// RunDMSCmd is a wrapper for running the DMS CLI
func (i *Instance) RunDMSCmd(cmd string) (string, error) {
	fullCmd := fmt.Sprintf("DMS_PASSPHRASE=123 %s", cmd)
	return i.RunCMD([]string{"sh", "-c", fullCmd})
}

// RunDMSCmd is a wrapper for running the DMS CLI
func (i *Instance) RunDMSCmdBackground(cmd string) error {
	fullCmd := fmt.Sprintf("DMS_PASSPHRASE=123 %s", cmd)
	return i.RunCMDBackground([]string{"sh", "-c", fullCmd})
}

// WaitForInstanceReady waits a instance to be running and ready to be used
func (i *Instance) WaitForInstanceReady() error {
	return WaitForInstanceReady(i.Client, i.Name, 60*time.Second)
}

// UploadFile uploads a local file to the instance.
func (i *Instance) UploadFile(localPath, remotePath string, mode int) error {
	return UploadFileToInstance(i.Client, i.Name, localPath, remotePath, mode)
}

// Destroy deletes the container instance.
func (i *Instance) Destroy() error {
	return DeleteInstance(i.Client, i.Name)
}

// TotalRAMGB returns the total RAM in GB available on the instance.
func (i *Instance) TotalRAMGB() (float64, error) {
	output, err := i.RunCMD([]string{"sh", "-c", "free -m | awk '/^Mem:/ {print $2}'"})
	if err != nil {
		return 0, fmt.Errorf("failed to get RAM info from node %s: %w", i.Name, err)
	}

	var memMB float64
	_, err = fmt.Sscanf(strings.TrimSpace(output), "%f", &memMB)
	if err != nil {
		return 0, fmt.Errorf("failed to parse RAM value '%s' from node %s: %w", output, i.Name, err)
	}

	return memMB / 1024.0, nil
}

// TotalDiskGB returns the total disk size in GB available on the instance.
func (i *Instance) TotalDiskGB() (float64, error) {
	output, err := i.RunCMD([]string{"sh", "-c", "df -BG / | awk 'NR==2 {print $2}'"})
	if err != nil {
		return 0, fmt.Errorf("failed to get disk info from node %s: %w", i.Name, err)
	}

	output = strings.TrimSpace(output)
	output = strings.TrimSuffix(output, "G")

	var diskGB float64
	_, err = fmt.Sscanf(output, "%f", &diskGB)
	if err != nil {
		return 0, fmt.Errorf("failed to parse disk value '%s' from node %s: %w", output, i.Name, err)
	}

	return diskGB, nil
}

// CPUCores returns the total number of CPU cores available on the instance.
func (i *Instance) CPUCores() (int, error) {
	output, err := i.RunCMD([]string{"sh", "-c", "nproc"})
	if err != nil {
		return 0, fmt.Errorf("failed to get CPU core count from node %s: %w", i.Name, err)
	}

	var cores int
	_, err = fmt.Sscanf(strings.TrimSpace(output), "%d", &cores)
	if err != nil {
		return 0, fmt.Errorf("failed to parse CPU core count '%s' from node %s: %w", output, i.Name, err)
	}

	return cores, nil
}

func (i *Instance) GetNetInfo() (NetInfo, error) {
	instState, _, err := i.Client.GetInstanceState(i.Name)
	if err != nil {
		return NetInfo{}, fmt.Errorf("failed to get instance state for %s: %w", i.Name, err)
	}

	netI := NetInfo{}

	for netName, netInfo := range instState.Network {
		for _, addr := range netInfo.Addresses {
			if addr.Family == "inet" && addr.Scope == "global" && (strings.HasPrefix(netName, "eth") || strings.HasPrefix(netName, "enp")) {
				netI.NetworkName = netName
				netI.IPAddress = addr.Address
				return netI, nil
			}
		}
	}

	return netI, nil
}

// OnboardingResources returns an amount of the instance's resources (RAM, CPU cores, and disk)
// which is the necessary amount for onboarding.
func (i *Instance) OnboardingResources() (ramGB float64, cpuCores float64, diskGB float64, err error) {
	totalRAM, err := i.TotalRAMGB()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get total RAM: %w", err)
	}

	totalCPUCores, err := i.CPUCores()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get total CPU cores: %w", err)
	}

	totalDisk, err := i.TotalDiskGB()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get total disk: %w", err)
	}

	ramGB = totalRAM * 0.7
	cpuCores = float64(totalCPUCores) * 0.7
	diskGB = totalDisk * 0.7

	return ramGB, cpuCores, diskGB, nil
}

func (i *Instance) IsDMSRunning(port int) bool {
	out, err := i.RunCMD([]string{"ss", "-tnlp"})
	if err != nil {
		return false
	}
	return strings.Contains(out, fmt.Sprintf(":%d", port))
}

func (i *Instance) PruneResolved() error {
	dest := "/root/netplan.sh"
	if err := i.UploadFile(FindTestdata("scripts/netplan.sh"), dest, 0o755); err != nil {
		return err
	}
	_, err := i.RunCMD([]string{"bash", "-c", dest})
	if err != nil {
		return err
	}
	dest = "/root/resolv.sh"
	if err := i.UploadFile(FindTestdata("scripts/resolv.sh"), dest, 0o755); err != nil {
		return err
	}
	_, err = i.RunCMD([]string{"bash", "-c", dest})
	if err != nil {
		return err
	}
	return nil
}
