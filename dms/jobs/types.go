package jobs

import "gitlab.com/nunet/device-management-service/models"

// JobConfig represents the root of the job configuration
type JobSpec struct {
	Version string `json:"version" description:"Version of the job configuration"`
	Jobs    []*Job `json:"jobs" description:"List of jobs"`
}

// JobLink represents a link between two jobs
type JobLink struct {
	Type   string `json:"type" description:"Type of the link"`
	Target string `json:"target" description:"Name of the target job"`
	Dependency string `json:"dependency" description:"Dependency of the link"`
}

// Job represents a single job in the configuration
type Job struct {
	Instances int                       `json:"instances" description:"Number of instances"`
	Name      string                    `json:"name" description:"Name of the job"`
	Metadata  JobMetadata               `json:"metadata" description:"Metadata of the job"`
	Locality  []string                  `json:"locality" description:"Deployment locality"`
	Execution models.SpecConfig         `json:"execution" description:"Execution Engine configuration"`
	Resources models.ExecutionResources `json:"resources" description:"Resources required"`
	Volumes   []VolumeConfig           `json:"volumes" description:"List of volumes"`
	Networks  []NetworkConfig          `json:"networks" description:"List of networks"`
	Libraries []Library                 `json:"libraries" description:"List of required libraries"`
	Links     []JobLink                `json:"links" description:"List of links"`
	Children  []Job                    `json:"children" description:"List of tasks"`
}

// Metadata contains job metadata
type JobMetadata struct {
	Namespace string `json:"namespace" description:"Namespace of the job"`
}

// Volume represents a volume configuration
type VolumeConfig struct {
	Name       string            `json:"name" description:"Name of the volume"`
	Type       string            `json:"type" description:"Type of the volume"`
	Remote     models.SpecConfig `json:"remote" description:"Remote volume configuration"`
	Mountpoint string            `json:"mountpoint" description:"Mountpoint of the volume"`
}

// NetworkConfig represents a network configuration
type NetworkConfig struct {
	Name    string         `json:"name" description:"Name of the network"`
	Type    string         `json:"type" description:"Type of the network"`
	PortMap []NetworkPortMap `json:"port_map" description:"Port mapping"`
}

// NetworkPortMap represents a port mapping
type NetworkPortMap struct {
	Protocol      string `json:"protocol" description:"Protocol"`
	ContainerPort int    `json:"container_port" description:"Container port"`
	HostPort      int    `json:"host_port" description:"Host port"`
}

// Library represents a library configuration
type Library struct {
	Name    string `json:"name" description:"Name of the library"`
	Version string `json:"version" description:"Version of the library"`
}
