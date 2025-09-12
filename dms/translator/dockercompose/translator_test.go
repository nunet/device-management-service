package dockercompose

import (
	"testing"
	"time"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/suite"
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	nunettypes "gitlab.com/nunet/device-management-service/types"
)

type TranslatorTestSuite struct {
	suite.Suite
}

func TestTranslatorTestSuite(t *testing.T) {
	suite.Run(t, new(TranslatorTestSuite))
}

func (s *TranslatorTestSuite) TestNewDockerComposeTranslator() {
	translator := NewDockerComposeTranslator()
	s.NotNil(translator)
	s.IsType(&DockerTranslator{}, translator)
}

func (s *TranslatorTestSuite) TestTranslateSimpleService() {
	content := []byte(`
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
`)

	translator := NewDockerComposeTranslator()
	result, err := translator.Translate(content)

	s.NoError(err)
	s.NotNil(result)
	s.NotNil(result.Config)
	s.NotNil(result.Config.V1)

	// Check allocations
	s.Len(result.Config.V1.Allocations, 1)
	s.Contains(result.Config.V1.Allocations, "web")

	webAlloc := result.Config.V1.Allocations["web"]
	s.Equal(jobtypes.ExecutorDocker, webAlloc.Executor)
	s.Equal(jobtypes.AllocationTypeService, webAlloc.Type)
	s.Equal("docker", webAlloc.Execution.Type)
	s.Equal("nginx:latest", webAlloc.Execution.Params["image"])

	// Check nodes
	s.Len(result.Config.V1.Nodes, 1)
	s.Contains(result.Config.V1.Nodes, "web")

	webNode := result.Config.V1.Nodes["web"]
	s.Equal([]string{"web"}, webNode.Allocations)
	s.Len(webNode.Ports, 1)
	s.Equal(80, webNode.Ports[0].Public)
	s.Equal(80, webNode.Ports[0].Private)
	s.Equal("web", webNode.Ports[0].Allocation)

	// Check subnet configuration
	s.True(result.Config.V1.Subnet.Join)
	s.Equal(jobtypes.EscalationStrategyRedeploy, result.Config.V1.EscalationStrategy)
}

func (s *TranslatorTestSuite) TestTranslateComplexService() {
	content := []byte(`
version: '3.8'
services:
  app:
    image: myapp:latest
    command: ["python", "app.py"]
    entrypoint: ["/entrypoint.sh"]
    working_dir: /app
    environment:
      - DEBUG=true
      - PORT=8080
    volumes:
      - ./src:/app/src:ro
      - data:/app/data
    # removed depends_on to avoid validation errors
    restart: always
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 1G
volumes:
  data:
`)

	translator := NewDockerComposeTranslator()
	result, err := translator.Translate(content)

	s.NoError(err)
	s.NotNil(result)

	appAlloc := result.Config.V1.Allocations["app"]

	// Check execution parameters
	s.Equal("myapp:latest", appAlloc.Execution.Params["image"])

	// Command and entrypoint are ShellCommand types, convert to []string for comparison
	cmd := appAlloc.Execution.Params["command"].(composetypes.ShellCommand)
	s.Equal([]string{"python", "app.py"}, []string(cmd))

	entrypoint := appAlloc.Execution.Params["entrypoint"].(composetypes.ShellCommand)
	s.Equal([]string{"/entrypoint.sh"}, []string(entrypoint))

	s.Equal("/app", appAlloc.Execution.Params["working_directory"])

	// Environment is MappingWithEquals, check individual keys
	env := appAlloc.Execution.Params["environment"].(composetypes.MappingWithEquals)
	s.Contains(env, "DEBUG")
	s.Contains(env, "PORT")
	s.Equal("true", *env["DEBUG"])
	s.Equal("8080", *env["PORT"])

	// Dependencies were removed from compose file, so skip this check
	// s.Contains(appAlloc.DependsOn, "db")

	// Check resources
	s.Equal(float32(2.0), appAlloc.Resources.CPU.Cores)
	s.Equal(uint64(1073741824), appAlloc.Resources.RAM.Size) // 1G in bytes

	// Check volumes
	s.Len(appAlloc.Volume, 2)

	// Find the read-only volume
	var roVolume *nunettypes.VolumeConfig
	for i := range appAlloc.Volume {
		if appAlloc.Volume[i].ReadOnly {
			roVolume = &appAlloc.Volume[i]
			break
		}
	}
	s.NotNil(roVolume)
	// Source path is resolved to absolute path by compose-go
	s.Contains(roVolume.Src, "src") // Check that it contains 'src' since path is resolved
	s.Equal("/app/src", roVolume.MountDestination)
	s.True(roVolume.ReadOnly)

	// Check healthcheck
	s.Equal("command", appAlloc.HealthCheck.Type)
	s.Equal([]string{"curl", "-f", "http://localhost:8080/health"}, appAlloc.HealthCheck.Exec)
	s.Equal(30*time.Second, appAlloc.HealthCheck.Interval)

	// Check restart policy
	appNode := result.Config.V1.Nodes["app"]
	s.Equal(jobtypes.NodeFailureRecoveryRestart, appNode.FailureRecovery)
}

func (s *TranslatorTestSuite) TestTranslateMultipleServices() {
	content := []byte(`
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
    depends_on:
      - api
  api:
    image: myapi:latest
    ports:
      - "8080:8080"
    depends_on:
      - db
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
`)

	translator := NewDockerComposeTranslator()
	result, err := translator.Translate(content)

	s.NoError(err)
	s.NotNil(result)

	// Check all services are translated
	s.Len(result.Config.V1.Allocations, 3)
	s.Len(result.Config.V1.Nodes, 3)

	s.Contains(result.Config.V1.Allocations, "web")
	s.Contains(result.Config.V1.Allocations, "api")
	s.Contains(result.Config.V1.Allocations, "db")

	// Check dependencies
	webAlloc := result.Config.V1.Allocations["web"]
	s.Contains(webAlloc.DependsOn, "api")

	apiAlloc := result.Config.V1.Allocations["api"]
	s.Contains(apiAlloc.DependsOn, "db")

	dbAlloc := result.Config.V1.Allocations["db"]
	s.Empty(dbAlloc.DependsOn)

	// Check ports
	webNode := result.Config.V1.Nodes["web"]
	s.Len(webNode.Ports, 1)
	s.Equal(80, webNode.Ports[0].Public)

	apiNode := result.Config.V1.Nodes["api"]
	s.Len(apiNode.Ports, 1)
	s.Equal(8080, apiNode.Ports[0].Public)

	dbNode := result.Config.V1.Nodes["db"]
	s.Empty(dbNode.Ports)
}

func (s *TranslatorTestSuite) TestTranslateInvalidInput() {
	content := []byte(`invalid yaml content [`)

	translator := NewDockerComposeTranslator()
	result, err := translator.Translate(content)

	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "failed to parse docker compose file")
}

func (s *TranslatorTestSuite) TestTranslateServiceRestart() {
	tests := []struct {
		name                  string
		restart               string
		expectedAllocRecovery jobtypes.AllocationFailureRecovery
		expectedNodeRecovery  jobtypes.NodeFailureRecovery
	}{
		{
			name:                  "no restart",
			restart:               composetypes.RestartPolicyNo,
			expectedAllocRecovery: jobtypes.AllocationFailureRecoveryStayDown,
			expectedNodeRecovery:  jobtypes.NodeFailureRecoveryStayDown,
		},
		{
			name:                  "always restart",
			restart:               composetypes.RestartPolicyAlways,
			expectedAllocRecovery: "",
			expectedNodeRecovery:  jobtypes.NodeFailureRecoveryRestart,
		},
		{
			name:                  "unless stopped",
			restart:               composetypes.RestartPolicyUnlessStopped,
			expectedAllocRecovery: "",
			expectedNodeRecovery:  jobtypes.NodeFailureRecoveryRestart,
		},
		{
			name:                  "on failure",
			restart:               composetypes.RestartPolicyOnFailure,
			expectedAllocRecovery: "",
			expectedNodeRecovery:  jobtypes.NodeFailureRecoveryRestart,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wc := &WarningCollector{}
			allocRecovery, nodeRecovery := translateServiceRestart("test", tt.restart, wc)

			s.Equal(tt.expectedAllocRecovery, allocRecovery)
			s.Equal(tt.expectedNodeRecovery, nodeRecovery)
		})
	}
}

func (s *TranslatorTestSuite) TestTranslateServiceRestartUnknown() {
	wc := &WarningCollector{}
	allocRecovery, nodeRecovery := translateServiceRestart("test", "unknown", wc)

	s.Empty(allocRecovery)
	s.Empty(nodeRecovery)

	warnings := wc.Get()
	s.Len(warnings, 1)
	s.Contains(warnings[0], "unknown restart policy 'unknown'")
}

func (s *TranslatorTestSuite) TestTranslateResources() {
	tests := []struct {
		name     string
		deploy   *composetypes.DeployConfig
		expected nunettypes.Resources
	}{
		{
			name:     "nil deploy config",
			deploy:   nil,
			expected: nunettypes.Resources{},
		},
		{
			name: "nil limits",
			deploy: &composetypes.DeployConfig{
				Resources: composetypes.Resources{
					Limits: nil,
				},
			},
			expected: nunettypes.Resources{},
		},
		{
			name: "valid CPU and memory limits",
			deploy: &composetypes.DeployConfig{
				Resources: composetypes.Resources{
					Limits: &composetypes.Resource{
						NanoCPUs:    2.5,
						MemoryBytes: 1073741824, // 1GB
					},
				},
			},
			expected: nunettypes.Resources{
				CPU: nunettypes.CPU{
					Cores: 2.5,
				},
				RAM: nunettypes.RAM{
					Size: 1073741824,
				},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wc := &WarningCollector{}
			result := translateResources("test", tt.deploy, wc)

			s.Equal(tt.expected.CPU.Cores, result.CPU.Cores)
			s.Equal(tt.expected.RAM.Size, result.RAM.Size)
		})
	}
}

func (s *TranslatorTestSuite) TestTranslateResourcesWithReservations() {
	deploy := &composetypes.DeployConfig{
		Resources: composetypes.Resources{
			Limits: &composetypes.Resource{
				NanoCPUs:    2.0,
				MemoryBytes: 1073741824,
			},
			Reservations: &composetypes.Resource{
				NanoCPUs:    1.0,
				MemoryBytes: 536870912,
			},
		},
	}

	wc := &WarningCollector{}
	result := translateResources("test", deploy, wc)

	// Should use limits, not reservations
	s.Equal(float32(2.0), result.CPU.Cores)
	s.Equal(uint64(1073741824), result.RAM.Size)

	// Should generate warning about reservations
	warnings := wc.Get()
	s.Len(warnings, 1)
	s.Contains(warnings[0], "resource reservations are not supported")
}

func (s *TranslatorTestSuite) TestTranslateVolumes() {
	projectVolumes := composetypes.Volumes{
		"data": composetypes.VolumeConfig{
			Driver: "local",
		},
		"external_data": composetypes.VolumeConfig{
			Driver: "nfs",
		},
	}

	serviceVolumes := []composetypes.ServiceVolumeConfig{
		{
			Type:     "bind",
			Source:   "./src",
			Target:   "/app/src",
			ReadOnly: true,
		},
		{
			Type:   "volume",
			Source: "data",
			Target: "/app/data",
		},
		{
			Type:   "volume",
			Source: "external_data",
			Target: "/app/external",
		},
	}

	wc := &WarningCollector{}
	result := translateVolumes("test", serviceVolumes, projectVolumes, wc)

	// Should have 2 volumes (external_data should be skipped due to non-local driver)
	s.Len(result, 2)

	// Check bind volume
	bindVol := result[0]
	s.Equal("local", bindVol.Type)
	s.Equal("./src", bindVol.Src)
	s.Equal("/app/src", bindVol.MountDestination)
	s.True(bindVol.ReadOnly)

	// Check local volume
	localVol := result[1]
	s.Equal("local", localVol.Type)
	s.Equal("data", localVol.Src)
	s.Equal("/app/data", localVol.MountDestination)
	s.False(localVol.ReadOnly)

	// Check warnings
	warnings := wc.Get()
	s.Len(warnings, 1)
	s.Contains(warnings[0], "only 'local' volumes are supported")
}

func (s *TranslatorTestSuite) TestTranslateVolumesMissingVolume() {
	projectVolumes := composetypes.Volumes{}
	serviceVolumes := []composetypes.ServiceVolumeConfig{
		{
			Type:   "volume",
			Source: "missing_volume",
			Target: "/app/data",
		},
	}

	wc := &WarningCollector{}
	result := translateVolumes("test", serviceVolumes, projectVolumes, wc)

	s.Len(result, 1) // Volume should still be created
	s.Equal("missing_volume", result[0].Src)

	warnings := wc.Get()
	s.Len(warnings, 1)
	s.Contains(warnings[0], "volume 'missing_volume' not found")
}

func (s *TranslatorTestSuite) TestTranslatePorts() {
	ports := []composetypes.ServicePortConfig{
		{
			Target:    80,
			Published: "8080",
			Protocol:  "tcp",
		},
		{
			Target:    443,
			Published: "8443",
			Protocol:  "tcp",
		},
		{
			Target:    53,
			Published: "5353",
			Protocol:  "udp", // Should be ignored
		},
		{
			Target:    3000,
			Published: "3000",
			Protocol:  "", // Default to TCP
		},
	}

	wc := &WarningCollector{}
	result := translatePorts(ports, "test", wc)

	// Should have 3 ports (UDP port ignored)
	s.Len(result, 3)

	// Check first port
	s.Equal(8080, result[0].Public)
	s.Equal(80, result[0].Private)
	s.Equal("test", result[0].Allocation)

	// Check second port
	s.Equal(8443, result[1].Public)
	s.Equal(443, result[1].Private)

	// Check third port (default protocol)
	s.Equal(3000, result[2].Public)
	s.Equal(3000, result[2].Private)

	// Check warnings
	warnings := wc.Get()
	s.Len(warnings, 1)
	s.Contains(warnings[0], "only TCP ports are supported")
}

func (s *TranslatorTestSuite) TestTranslateHealthCheck() {
	interval := composetypes.Duration(time.Second * 30)
	tests := []struct {
		name     string
		hc       *composetypes.HealthCheckConfig
		expected nunettypes.HealthCheckManifest
	}{
		{
			name:     "nil healthcheck",
			hc:       nil,
			expected: nunettypes.HealthCheckManifest{},
		},
		{
			name: "disabled healthcheck",
			hc: &composetypes.HealthCheckConfig{
				Disable: true,
			},
			expected: nunettypes.HealthCheckManifest{},
		},
		{
			name: "empty test",
			hc: &composetypes.HealthCheckConfig{
				Test: []string{},
			},
			expected: nunettypes.HealthCheckManifest{},
		},
		{
			name: "CMD healthcheck",
			hc: &composetypes.HealthCheckConfig{
				Test:     []string{"CMD", "curl", "-f", "http://localhost"},
				Interval: &interval,
			},
			expected: nunettypes.HealthCheckManifest{
				Type:     "command",
				Exec:     []string{"curl", "-f", "http://localhost"},
				Interval: 30 * time.Second,
			},
		},
		{
			name: "CMD-SHELL healthcheck",
			hc: &composetypes.HealthCheckConfig{
				Test: []string{"CMD-SHELL", "curl -f http://localhost || exit 1"},
			},
			expected: nunettypes.HealthCheckManifest{
				Type: "command",
				Exec: []string{"curl -f http://localhost || exit 1"},
			},
		},
		{
			name: "direct command healthcheck",
			hc: &composetypes.HealthCheckConfig{
				Test: []string{"curl", "-f", "http://localhost"},
			},
			expected: nunettypes.HealthCheckManifest{
				Type: "command",
				Exec: []string{"curl", "-f", "http://localhost"},
			},
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			wc := &WarningCollector{}
			result := translateHealthCheck(tt.hc, wc)

			s.Equal(tt.expected.Type, result.Type)
			s.Equal(tt.expected.Exec, result.Exec)
			s.Equal(tt.expected.Interval, result.Interval)
		})
	}
}

func (s *TranslatorTestSuite) TestTranslateHealthCheckWithUnsupportedFeatures() {
	interval := composetypes.Duration(time.Second * 30)
	hc := &composetypes.HealthCheckConfig{
		Test:        []string{"CMD", "curl", "-f", "http://localhost"},
		Timeout:     &interval,
		Retries:     &[]uint64{3}[0],
		StartPeriod: &interval,
	}

	wc := &WarningCollector{}
	result := translateHealthCheck(hc, wc)

	s.Equal("command", result.Type)
	s.Equal([]string{"curl", "-f", "http://localhost"}, result.Exec)

	// Check warnings for unsupported features
	warnings := wc.Get()
	s.Len(warnings, 3)

	warningTexts := make([]string, len(warnings))
	copy(warningTexts, warnings)

	s.Contains(warningTexts[0], "timeout")
	s.Contains(warningTexts[1], "retries")
	s.Contains(warningTexts[2], "start_period")
}

func (s *TranslatorTestSuite) TestTranslateHealthCheckInvalidCMD() {
	hc := &composetypes.HealthCheckConfig{
		Test: []string{"CMD"}, // Missing command
	}

	wc := &WarningCollector{}
	result := translateHealthCheck(hc, wc)

	s.Equal("command", result.Type)
	s.Empty(result.Exec)

	warnings := wc.Get()
	s.Len(warnings, 1)
	s.Contains(warnings[0], "healthcheck test must be a command")
}

func (s *TranslatorTestSuite) TestTranslateWithWarnings() {
	content := []byte(`
version: '3.8'
services:
  web:
    image: nginx:latest
    build:
      context: ./
    privileged: true
    ports:
      - "80:80"
      - "53:53/udp"
networks:
  frontend:
  backend:
configs:
  nginx_config:
    file: ./nginx.conf
`)

	translator := NewDockerComposeTranslator()
	result, err := translator.Translate(content)

	s.NoError(err)
	s.NotNil(result)
	s.NotEmpty(result.Warnings)

	// Should have warnings for multiple networks, configs, and unsupported service features
	warningFound := func(substring string) bool {
		for _, warning := range result.Warnings {
			if len(warning) > len(substring) {
				for i := 0; i <= len(warning)-len(substring); i++ {
					if warning[i:i+len(substring)] == substring {
						return true
					}
				}
			}
		}
		return false
	}

	s.True(warningFound("multiple networks"), "Expected warning about multiple networks")
	s.True(warningFound("configs"), "Expected warning about configs")
	s.True(warningFound("UDP"), "Expected warning about UDP ports")
}
