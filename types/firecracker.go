package types

type BootSource struct {
	KernelImagePath string `json:"kernel_image_path"`
	BootArgs        string `json:"boot_args"`
}

type Drives struct {
	DriveID      string `json:"drive_id"`
	PathOnHost   string `json:"path_on_host"`
	IsRootDevice bool   `json:"is_root_device"`
	IsReadOnly   bool   `json:"is_read_only"`
}

type MachineConfig struct {
	VCPUCount  int `json:"vcpu_count"`
	MemSizeMib int `json:"mem_size_mib"`
}

type NetworkInterfaces struct {
	IfaceID     string `json:"iface_id"`
	GuestMac    string `json:"guest_mac"`
	HostDevName string `json:"host_dev_name"`
}

type MMDSConfig struct {
	NetworkInterface []string `json:"network_interfaces"`
}

type MMDSMsg struct {
	Latest struct {
		Metadata struct {
			MMDSMetadata
		} `json:"meta-data"`
	} `json:"latest"`
}

type MMDSMetadata struct {
	NodeId string `json:"node_id"`
	PKey   string `json:"pkey"`
}

type Actions struct {
	ActionType string `json:"action_type"`
}
type VirtualMachine struct {
	BaseDBModel
	SocketFile string `json:"socket_file"`
	BootSource string `json:"boot_source"`
	Filesystem string `json:"filesystem"`
	VCPUCount  int    `json:"vcpu_count"`
	MemSizeMib int    `json:"mem_size_mib"`
	TapDevice  string `json:"tap_device"`
	State      string `json:"state"`
}
