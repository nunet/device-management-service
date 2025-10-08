// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package dockercompose

import (
	"fmt"
	"slices"
	"strconv"
	"time"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/dms/translator/types"
	nunettypes "gitlab.com/nunet/device-management-service/types"
)

// DockerTranslator implements the translator.Translator interface for Docker Compose files.
type DockerTranslator struct{}

// NewDockerComposeTranslator creates a new DockerTranslator.
func NewDockerComposeTranslator() *DockerTranslator {
	return &DockerTranslator{}
}

// Translate converts a Docker Compose file content into a NuNet EnsembleConfig.
func (t *DockerTranslator) Translate(input []byte) (*types.Translation, error) {
	project, err := Parse(input)
	if err != nil {
		return nil, fmt.Errorf("failed to parse docker compose file: %w", err)
	}

	warnings := &WarningCollector{}
	allocations := make(map[string]jobtypes.AllocationConfig)
	nodes := make(map[string]jobtypes.NodeConfig)

	// Translate top-level unsupported features
	if len(project.Networks) > 1 {
		warnings.Add("", "networks", "multiple networks are not supported; all services will be joined into a single subnet.")
	}
	checkUnsupportedTopLevelFeatures(project, warnings)

	// Translate each service
	for _, service := range project.ServiceNames() {
		alloc, node := translateService(project, service, warnings)
		allocations[service] = alloc
		nodes[service] = node
	}

	ensemble := &jobtypes.EnsembleConfig{
		V1: &jobtypes.EnsembleConfigV1{
			Allocations: allocations,
			Nodes:       nodes,
			Subnet: jobtypes.SubnetConfig{
				Join: true, // All services join the same subnet by default
			},
			// Default escalation strategy
			EscalationStrategy: jobtypes.EscalationStrategyRedeploy,
		},
	}

	return &types.Translation{
		Config:   ensemble,
		Warnings: warnings.Get(),
	}, nil
}

// translateService converts a single Docker Compose service into a NuNet Allocation and Node.
func translateService(project *composetypes.Project, serviceName string, w *WarningCollector) (jobtypes.AllocationConfig, jobtypes.NodeConfig) {
	service, err := project.GetService(serviceName)
	if err != nil {
		w.Add("", "services", fmt.Sprintf("service '%s' not found", serviceName))
		return jobtypes.AllocationConfig{}, jobtypes.NodeConfig{}
	}

	executionParams := map[string]any{
		"image": service.Image,
	}
	if service.Command != nil {
		executionParams["command"] = service.Command
	}
	if service.Entrypoint != nil {
		executionParams["entrypoint"] = service.Entrypoint
	}
	if len(service.Environment) > 0 {
		executionParams["environment"] = service.Environment
	}
	if service.WorkingDir != "" {
		executionParams["working_directory"] = service.WorkingDir
	}

	// Set cpu and memory limits if specified in the service
	if service.Deploy == nil {
		service.Deploy = &composetypes.DeployConfig{}
	}
	if service.Deploy.Resources.Limits == nil {
		limits := composetypes.Resource{}
		if service.CPUS > 0 {
			limits.NanoCPUs = composetypes.NanoCPUs(service.CPUS)
		}
		if service.MemLimit > 0 {
			limits.MemoryBytes = service.MemLimit
		}
		service.Deploy.Resources.Limits = &limits
	}

	alloc := jobtypes.AllocationConfig{
		Executor:    jobtypes.ExecutorDocker,
		Type:        jobtypes.AllocationTypeService,
		Resources:   translateResources(serviceName, service.Deploy, w),
		Execution:   nunettypes.SpecConfig{Type: "docker", Params: executionParams},
		DNSName:     service.DomainName,
		Volume:      translateVolumes(serviceName, service.Volumes, project.Volumes, w),
		HealthCheck: translateHealthCheck(service.HealthCheck, w),
		DependsOn:   service.GetDependencies(),
	}

	node := jobtypes.NodeConfig{
		Allocations: []string{service.Name},
		Ports:       translatePorts(service.Ports, service.Name, w),
	}
	if service.Restart != "" {
		alloc.FailureRecovery, node.FailureRecovery = translateServiceRestart(serviceName, service.Restart, w)
	}

	checkForUnsupportedServiceFeatures(service, w)

	return alloc, node
}

func translateServiceRestart(serviceName, restart string, w *WarningCollector) (jobtypes.AllocationFailureRecovery, jobtypes.NodeFailureRecovery) {
	switch restart {
	case composetypes.RestartPolicyNo:
		return jobtypes.AllocationFailureRecoveryStayDown, jobtypes.NodeFailureRecoveryStayDown
	case composetypes.RestartPolicyAlways, composetypes.RestartPolicyUnlessStopped, composetypes.RestartPolicyOnFailure:
		return "", jobtypes.NodeFailureRecoveryRestart
	default:
		w.Add(serviceName, "restart", fmt.Sprintf("unknown restart policy '%s'", restart))
		return "", ""
	}
}

func translateResources(serviceName string, deploy *composetypes.DeployConfig, w *WarningCollector) nunettypes.Resources {
	// Set default resources
	res := nunettypes.Resources{}

	if deploy == nil || deploy.Resources.Limits == nil {
		return res
	}

	limits := deploy.Resources.Limits
	if limits.NanoCPUs > 0 {
		res.CPU.Cores = limits.NanoCPUs.Value()
	}

	if limits.MemoryBytes > 0 {
		res.RAM.Size = uint64(limits.MemoryBytes)
	}

	if deploy.Resources.Reservations != nil {
		w.Add(serviceName, "deploy.resources.reservations", "resource reservations are not supported and will be ignored. Using limits instead.")
	}

	return res
}

func translateVolumes(serviceName string, volumes []composetypes.ServiceVolumeConfig, projectVolumes composetypes.Volumes, w *WarningCollector) []nunettypes.VolumeConfig {
	nunetVolumes := make([]nunettypes.VolumeConfig, 0)
	for _, vol := range volumes {
		v := nunettypes.VolumeConfig{
			Type:             "local",
			MountDestination: vol.Target,
			ReadOnly:         vol.ReadOnly,
		}

		switch vol.Type {
		case "bind":
			v.Src = vol.Source
		case "volume":
			v.Src = vol.Source
			volume, ok := projectVolumes[vol.Source]
			if !ok {
				w.Add("", "volumes", fmt.Sprintf("volume '%s' not found", vol.Source))
			}

			if volume.Driver != "" && volume.Driver != "local" {
				w.Add(serviceName, "volumes.volume.driver", "only 'local' volumes are supported.")
				continue
			}

			if dev, ok := volume.DriverOpts["device"]; ok && dev != "" {
				v.Src = dev
			}

			if vol.Volume != nil && vol.Volume.NoCopy {
				w.Add("", "volumes.volume.nocopy", "'nocopy' is not supported and will be ignored.")
			}
		default:
			w.Add("", fmt.Sprintf("volumes type '%s'", vol.Type), "only 'bind' and 'volume' types are supported.")
			continue
		}
		nunetVolumes = append(nunetVolumes, v)
	}
	return nunetVolumes
}

func translatePorts(ports []composetypes.ServicePortConfig, serviceName string, w *WarningCollector) []jobtypes.PortConfig {
	nunetPorts := make([]jobtypes.PortConfig, 0)
	for _, port := range ports {
		targetPort := int(port.Target)
		publishedPort, _ := strconv.Atoi(port.Published)

		if port.Protocol != "tcp" && port.Protocol != "" {
			w.Add(serviceName, fmt.Sprintf("ports.protocol: %s", port.Protocol), "only TCP ports are supported. UDP will be ignored.")
			continue
		}

		nunetPorts = append(nunetPorts, jobtypes.PortConfig{
			Public:     publishedPort,
			Private:    targetPort,
			Allocation: serviceName,
		})
	}
	return nunetPorts
}

func translateHealthCheck(hc *composetypes.HealthCheckConfig, w *WarningCollector) nunettypes.HealthCheckManifest {
	if hc == nil || hc.Disable || len(hc.Test) == 0 {
		return nunettypes.HealthCheckManifest{}
	}

	cmd := hc.Test
	if slices.Contains([]string{"CMD", "CMD-SHELL"}, hc.Test[0]) {
		if len(hc.Test) < 2 {
			w.Add("", "healthcheck.test", "healthcheck test must be a command if test type is CMD or CMD-SHELL.")
		}
		cmd = hc.Test[1:]
	}

	manifest := nunettypes.HealthCheckManifest{
		Type: "command",
		Exec: cmd,
	}

	if hc.Interval != nil {
		manifest.Interval = time.Duration(*hc.Interval)
	}

	if hc.Timeout != nil {
		w.Add("", "healthcheck.timeout", "'timeout' is not supported and will be ignored.")
	}
	if hc.Retries != nil {
		w.Add("", "healthcheck.retries", "'retries' is not supported and will be ignored.")
	}
	if hc.StartPeriod != nil {
		w.Add("", "healthcheck.start_period", "'start_period' is not supported and will be ignored.")
	}

	return manifest
}
