package docker

import (
	"encoding/json"
	"fmt"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils/validate"
)

const (
	EngineKeyImage            = "image"
	EngineKeyEntrypoint       = "entrypoint"
	EngineKeyCmd              = "cmd"
	EngineKeyEnvironment      = "environment"
	EngineKeyWorkingDirectory = "working_directory"
)

// EngineSpec contains necessary parameters to execute a docker job.
type EngineSpec struct {
	// Image this should be pullable by docker
	Image string `json:"image,omitempty"`
	// Entrypoint optionally override the default entrypoint
	Entrypoint []string `json:"entrypoint,omitempty"`
	// Cmd specifies the command to run in the container
	Cmd []string `json:"cmd,omitempty"`
	// EnvironmentVariables is a slice of env to run the container with
	Environment []string `json:"environment,omitempty"`
	// WorkingDirectory inside the container
	WorkingDirectory string `json:"working_directory,omitempty"`
}

// Validate checks if the engine spec is valid
func (c EngineSpec) Validate() error {
	if validate.IsBlank(c.Image) {
		return fmt.Errorf("invalid docker engine params: image cannot be empty")
	}
	return nil
}

// DecodeSpec decodes a spec config into a docker engine spec
// It converts the params into a docker EngineSpec struct and validates it
func DecodeSpec(spec *types.SpecConfig) (EngineSpec, error) {
	if !spec.IsType(types.ExecutorTypeDocker) {
		return EngineSpec{}, fmt.Errorf(
			"invalid docker engine type. expected %s, but received: %s",
			types.ExecutorTypeDocker,
			spec.Type,
		)
	}

	inputParams := spec.Params
	if inputParams == nil {
		return EngineSpec{}, fmt.Errorf("invalid docker engine params: params cannot be nil")
	}

	paramBytes, err := json.Marshal(inputParams)
	if err != nil {
		return EngineSpec{}, fmt.Errorf("failed to encode docker engine params: %w", err)
	}

	var dockerSpec *EngineSpec
	if err := json.Unmarshal(paramBytes, &dockerSpec); err != nil {
		return EngineSpec{}, fmt.Errorf("failed to decode docker engine params: %w", err)
	}

	return *dockerSpec, dockerSpec.Validate()
}

// EngineBuilder is a struct that is used for constructing an EngineSpec object
// specifically for Docker engines using the Builder pattern.
// It embeds an EngineBuilder object for handling the common builder methods.
type EngineBuilder struct {
	eb *types.SpecConfig
}

// NewDockerEngineBuilder function initializes a new DockerEngineBuilder instance.
// It sets the engine type to model.EngineDocker.String() and image as per the input argument.
func NewDockerEngineBuilder(image string) *EngineBuilder {
	eb := types.NewSpecConfig(types.ExecutorTypeDocker)
	eb.WithParam(EngineKeyImage, image)
	return &EngineBuilder{eb: eb}
}

// WithEntrypoint is a builder method that sets the Docker engine entrypoint.
// It returns the DockerEngineBuilder for further chaining of builder methods.
func (b *EngineBuilder) WithEntrypoint(e ...string) *EngineBuilder {
	b.eb.WithParam(EngineKeyEntrypoint, e)
	return b
}

// WithCmd is a builder method that sets the Docker engine's Command.
// It returns the DockerEngineBuilder for further chaining of builder methods.
func (b *EngineBuilder) WithCmd(c ...string) *EngineBuilder {
	b.eb.WithParam(EngineKeyCmd, c)
	return b
}

// WithEnvironment is a builder method that sets the Docker engine's environment variables.
// It returns the DockerEngineBuilder for further chaining of builder methods.
func (b *EngineBuilder) WithEnvironment(e ...string) *EngineBuilder {
	b.eb.WithParam(EngineKeyEnvironment, e)
	return b
}

// WithWorkingDirectory is a builder method that sets the Docker engine's working directory.
// It returns the DockerEngineBuilder for further chaining of builder methods.
func (b *EngineBuilder) WithWorkingDirectory(w string) *EngineBuilder {
	b.eb.WithParam(EngineKeyWorkingDirectory, w)
	return b
}

// Build method constructs the final SpecConfig object by calling the embedded EngineBuilder's Build method.
func (b *EngineBuilder) Build() *types.SpecConfig {
	return b.eb
}
