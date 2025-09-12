package dockercompose

import (
	"testing"

	composetypes "github.com/compose-spec/compose-go/v2/types"
	"github.com/stretchr/testify/suite"
)

type ParserTestSuite struct {
	suite.Suite
}

func TestParserTestSuite(t *testing.T) {
	suite.Run(t, new(ParserTestSuite))
}

func (s *ParserTestSuite) TestParse() {
	tests := []struct {
		name        string
		content     []byte
		expectError bool
	}{
		{
			name: "valid simple compose file",
			content: []byte(`
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
`),
			expectError: false,
		},
		{
			name: "valid complex compose file",
			content: []byte(`
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
      - "443:443"
    environment:
      - ENV_VAR=value
    volumes:
      - ./data:/var/www/html
    depends_on:
      - db
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
      POSTGRES_USER: user
      POSTGRES_PASSWORD: password
    volumes:
      - db_data:/var/lib/postgresql/data
volumes:
  db_data:
`),
			expectError: false,
		},
		{
			name: "compose file with resources",
			content: []byte(`
version: '3.8'
services:
  web:
    image: nginx:latest
    deploy:
      resources:
        limits:
          cpus: '2.0'
          memory: 1G
        reservations:
          cpus: '1.0'
          memory: 512M
`),
			expectError: false,
		},
		{
			name: "compose file with healthcheck",
			content: []byte(`
version: '3.8'
services:
  web:
    image: nginx:latest
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
`),
			expectError: false,
		},
		{
			name:        "invalid yaml",
			content:     []byte(`invalid: yaml: content: [`),
			expectError: true,
		},
		{
			name:        "empty content",
			content:     []byte(``),
			expectError: true,
		},
		{
			name: "invalid compose version",
			content: []byte(`
version: '1.0'
services:
  web:
    image: nginx
`),
			expectError: false, // compose-go library is lenient with versions
		},
		{
			name: "missing services section",
			content: []byte(`
version: '3.8'
volumes:
  data:
`),
			expectError: false, // valid compose file without services
		},
		{
			name: "compose file with networks",
			content: []byte(`
version: '3.8'
services:
  web:
    image: nginx:latest
    networks:
      - frontend
networks:
  frontend:
    driver: bridge
`),
			expectError: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			project, err := Parse(tt.content)

			if tt.expectError {
				s.Error(err)
				s.Nil(project)
			} else {
				s.NoError(err)
				s.NotNil(project)
				s.Equal("project", project.Name) // Default project name set by parser
			}
		})
	}
}

func (s *ParserTestSuite) TestParseProjectStructure() {
	content := []byte(`
version: '3.8'
services:
  web:
    image: nginx:latest
    ports:
      - "80:80"
    environment:
      ENV_VAR: value
  db:
    image: postgres:13
    environment:
      POSTGRES_DB: myapp
volumes:
  data:
networks:
  frontend:
`)

	project, err := Parse(content)
	s.NoError(err)
	s.NotNil(project)

	// Verify project structure
	s.Equal("project", project.Name)
	s.Len(project.Services, 2)
	s.Contains(project.ServiceNames(), "web")
	s.Contains(project.ServiceNames(), "db")

	// Verify web service
	webService, err := project.GetService("web")
	s.NoError(err)
	s.Equal("nginx:latest", webService.Image)
	s.Len(webService.Ports, 1)
	s.Equal(uint32(80), webService.Ports[0].Target)
	s.Equal("80", webService.Ports[0].Published)

	// Verify db service
	dbService, err := project.GetService("db")
	s.NoError(err)
	s.Equal("postgres:13", dbService.Image)
	// Environment is MappingWithEquals, check individual keys
	s.Contains(dbService.Environment, "POSTGRES_DB")
	s.Equal("myapp", *dbService.Environment["POSTGRES_DB"])

	// Verify volumes and networks
	s.Len(project.Volumes, 1)
	s.Contains(project.Volumes, "data")
	s.Len(project.Networks, 2) // default network + frontend
	s.Contains(project.Networks, "frontend")
}

func (s *ParserTestSuite) TestParseServiceFeatures() {
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
    restart: always
    healthcheck:
      test: ["CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"]
      interval: 30s
    deploy:
      resources:
        limits:
          cpus: '1.5'
          memory: 2G
volumes:
  data:
`)

	project, err := Parse(content)
	s.NoError(err)
	s.NotNil(project)

	service, err := project.GetService("app")
	s.NoError(err)

	// Verify service configuration
	s.Equal("myapp:latest", service.Image)

	// Command and entrypoint are ShellCommand types
	s.Equal([]string{"python", "app.py"}, []string(service.Command))
	s.Equal([]string{"/entrypoint.sh"}, []string(service.Entrypoint))
	s.Equal("/app", service.WorkingDir)

	// Environment is MappingWithEquals, check individual keys
	s.Contains(service.Environment, "DEBUG")
	s.Contains(service.Environment, "PORT")
	s.Equal("true", *service.Environment["DEBUG"])
	s.Equal("8080", *service.Environment["PORT"])
	s.Equal("always", service.Restart)

	// Verify volumes
	s.Len(service.Volumes, 2)
	// Source path is resolved to absolute path by compose-go
	s.Contains(service.Volumes[0].Source, "src")
	s.Equal("/app/src", service.Volumes[0].Target)
	s.True(service.Volumes[0].ReadOnly)

	// Verify healthcheck
	s.NotNil(service.HealthCheck)
	// Test is HealthCheckTest type
	s.Equal([]string{"CMD-SHELL", "curl -f http://localhost:8080/health || exit 1"}, []string(service.HealthCheck.Test))

	// Verify resource limits
	s.NotNil(service.Deploy)
	s.NotNil(service.Deploy.Resources.Limits)
	s.Equal(composetypes.NanoCPUs(1.5), service.Deploy.Resources.Limits.NanoCPUs)
	// MemoryBytes is UnitBytes type
	s.Equal(composetypes.UnitBytes(2147483648), service.Deploy.Resources.Limits.MemoryBytes) // 2G in bytes
}
