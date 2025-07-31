package utils

import (
	"fmt"
	"strings"

	incus "github.com/lxc/incus/client"
)

// DefaultDMSSuffix is the suffix added to DMS context names
const DefaultDMSSuffix = "-dms"

// Node represents a container instance.
type Node struct {
	Name     string
	Client   incus.InstanceServer
	Contexts map[string]*Context
}

// RunCMD runs a command inside the instance.
func (n *Node) RunCMD(cmd []string) (string, error) {
	return RunCommandInInstance(n.Client, n.Name, cmd)
}

// RunCMDBackground runs a command inside the instance in the background.
func (n *Node) RunCMDBackground(cmd []string) error {
	return RunBackgroundCommandInInstance(n.Client, n.Name, cmd)
}

// RunDMSCmd is a wrapper for running the DMS CLI
func (n *Node) RunDMSCmd(cmd string) (string, error) {
	fullCmd := fmt.Sprintf("DMS_PASSPHRASE=123 %s", cmd)
	return n.RunCMD([]string{"sh", "-c", fullCmd})
}

// RunDMSCmd is a wrapper for running the DMS CLI
func (n *Node) RunDMSCmdBackground(cmd string) error {
	fullCmd := fmt.Sprintf("DMS_PASSPHRASE=123 %s", cmd)
	return n.RunCMDBackground([]string{"sh", "-c", fullCmd})
}

// UploadFile uploads a local file to the instance.
func (n *Node) UploadFile(localPath, remotePath string, mode int) error {
	return UploadFileToInstance(n.Client, n.Name, localPath, remotePath, mode)
}

// Destroy deletes the container instance.
func (n *Node) Destroy() error {
	return DeleteInstance(n.Client, n.Name)
}

func (n *Node) CreateContext(name string) (*Context, error) {
	name = strings.ToLower(name)

	did, err := n.RunDMSCmd(fmt.Sprintf("nunet key new %s", name))
	if err != nil {
		return nil, err
	}

	_, err = n.RunDMSCmd(fmt.Sprintf("nunet cap new %s", name))
	if err != nil {
		return nil, err
	}

	context := &Context{
		Name: name,
		DID:  did,
		node: n,
	}

	n.Contexts[name] = context

	return context, nil
}

// GetTotalRAMGB returns the total RAM in GB available on the node.
func (n *Node) GetTotalRAMGB() (float64, error) {
	output, err := n.RunCMD([]string{"sh", "-c", "free -m | awk '/^Mem:/ {print $2}'"})
	if err != nil {
		return 0, fmt.Errorf("failed to get RAM info from node %s: %w", n.Name, err)
	}

	var memMB float64
	_, err = fmt.Sscanf(strings.TrimSpace(output), "%f", &memMB)
	if err != nil {
		return 0, fmt.Errorf("failed to parse RAM value '%s' from node %s: %w", output, n.Name, err)
	}

	return memMB / 1024.0, nil
}

// GetTotalDiskGB returns the total disk size in GB available on the node.
func (n *Node) GetTotalDiskGB() (float64, error) {
	output, err := n.RunCMD([]string{"sh", "-c", "df -BG / | awk 'NR==2 {print $2}'"})
	if err != nil {
		return 0, fmt.Errorf("failed to get disk info from node %s: %w", n.Name, err)
	}

	output = strings.TrimSpace(output)
	output = strings.TrimSuffix(output, "G")

	var diskGB float64
	_, err = fmt.Sscanf(output, "%f", &diskGB)
	if err != nil {
		return 0, fmt.Errorf("failed to parse disk value '%s' from node %s: %w", output, n.Name, err)
	}

	return diskGB, nil
}

// GetCPUCores returns the total number of CPU cores available on the node.
func (n *Node) GetCPUCores() (int, error) {
	output, err := n.RunCMD([]string{"sh", "-c", "nproc"})
	if err != nil {
		return 0, fmt.Errorf("failed to get CPU core count from node %s: %w", n.Name, err)
	}

	var cores int
	_, err = fmt.Sscanf(strings.TrimSpace(output), "%d", &cores)
	if err != nil {
		return 0, fmt.Errorf("failed to parse CPU core count '%s' from node %s: %w", output, n.Name, err)
	}

	return cores, nil
}

// GetOnboardingResources returns 10% of the node's resources (RAM, CPU cores, and disk)
// which is the necessary amount for onboarding.
func (n *Node) GetOnboardingResources() (ramGB float64, cpuCores float64, diskGB float64, err error) {
	totalRAM, err := n.GetTotalRAMGB()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get total RAM: %w", err)
	}

	totalCPUCores, err := n.GetCPUCores()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get total CPU cores: %w", err)
	}

	totalDisk, err := n.GetTotalDiskGB()
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to get total disk: %w", err)
	}

	ramGB = totalRAM * 0.2
	cpuCores = float64(totalCPUCores) * 0.2
	diskGB = totalDisk * 0.2

	return ramGB, cpuCores, diskGB, nil
}

func (n *Node) IsDMSRunning(port int) bool {
	out, err := n.RunCMD([]string{"ss", "-tnlp"})
	if err != nil {
		return false
	}
	return strings.Contains(out, fmt.Sprintf(":%d", port))
}

// InitialCaps creates user and DMS contexts with proper capabilities
func (n *Node) InitialCaps(name string) (userCtx, dmsCtx *Context, err error) {
	userCtx, err = n.CreateContext(name)
	if err != nil {
		return nil, nil, err
	}
	dmsCtx, err = n.CreateContext(name + DefaultDMSSuffix)
	if err != nil {
		return nil, nil, err
	}
	err = dmsCtx.Anchor("root", userCtx.DID)
	if err != nil {
		return nil, nil, err
	}
	return userCtx, dmsCtx, nil
}

func (n *Node) InstallDocker() error {
	_, err := n.RunCMD([]string{"apt-get", "update"})
	if err != nil {
		return fmt.Errorf("failed to update apt at node %s: %w", n.Name, err)
	}

	_, err = n.RunCMD([]string{"apt-get", "install", "-y", "ca-certificates", "curl"})
	if err != nil {
		return fmt.Errorf("failed to install prerequisites at node %s: %w", n.Name, err)
	}

	_, err = n.RunCMD([]string{"install", "-m", "0755", "-d", "/etc/apt/keyrings"})
	if err != nil {
		return fmt.Errorf("failed to create keyring directory at node %s: %w", n.Name, err)
	}

	_, err = n.RunCMD([]string{"curl", "-fsSL", "https://download.docker.com/linux/ubuntu/gpg", "-o", "/etc/apt/keyrings/docker.asc"})
	if err != nil {
		return fmt.Errorf("failed to download Docker GPG key at node %s: %w", n.Name, err)
	}

	_, err = n.RunCMD([]string{"chmod", "a+r", "/etc/apt/keyrings/docker.asc"})
	if err != nil {
		return fmt.Errorf("failed to set permissions on Docker GPG key at node %s: %w", n.Name, err)
	}

	_, err = n.RunCMD([]string{"sh", "-c", "echo \"deb [arch=$(dpkg --print-architecture) signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu $(. /etc/os-release && echo ${UBUNTU_CODENAME:-$VERSION_CODENAME}) stable\" | tee /etc/apt/sources.list.d/docker.list > /dev/null"})
	if err != nil {
		return fmt.Errorf("failed to add Docker repository at node %s: %w", n.Name, err)
	}

	_, err = n.RunCMD([]string{"apt-get", "update"})
	if err != nil {
		return fmt.Errorf("failed to update apt after adding Docker repository at node %s: %w", n.Name, err)
	}

	_, err = n.RunCMD([]string{"apt-get", "install", "-y", "docker-ce", "docker-ce-cli", "containerd.io", "docker-buildx-plugin", "docker-compose-plugin"})
	if err != nil {
		return fmt.Errorf("failed to install Docker packages at node %s: %w", n.Name, err)
	}

	_, err = n.RunCMD([]string{"systemctl", "start", "docker"})
	if err != nil {
		return fmt.Errorf("failed to start Docker daemon %s: %w", n.Name, err)
	}

	return nil
}

func (n *Node) PruneResolved() error {
	dest := "/root/netplan.sh"
	if err := n.UploadFile(FindTestdata("netplan.sh"), dest, 0o755); err != nil {
		return err
	}
	_, err := n.RunCMD([]string{"bash", "-c", dest})
	if err != nil {
		return err
	}
	dest = "/root/resolv.sh"
	if err := n.UploadFile(FindTestdata("resolv.sh"), dest, 0o755); err != nil {
		return err
	}
	_, err = n.RunCMD([]string{"bash", "-c", dest})
	if err != nil {
		return err
	}
	return nil
}

func (n *Node) InstallYQ() error {
	_, err := n.RunCMD([]string{"apt-get", "update"})
	if err != nil {
		return err
	}
	_, err = n.RunCMD([]string{"apt-get", "install", "-y", "wget"})
	if err != nil {
		return err
	}
	_, err = n.RunCMD([]string{"wget", "https://github.com/mikefarah/yq/releases/latest/download/yq_linux_amd64", "-O", "/usr/local/bin/yq"})
	if err != nil {
		return err
	}
	_, err = n.RunCMD([]string{"chmod", "+x", "/usr/local/bin/yq"})
	if err != nil {
		return err
	}
	return nil
}
