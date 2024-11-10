// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux
// +build linux

package firecracker

import (
	"encoding/json"
	"fmt"

	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils/validate"
)

const (
	EngineKeyKernelImage    = "kernel_image"
	EngineKeyKernelArgs     = "kernel_args"
	EngineKeyRootFileSystem = "root_file_system"
	EngineKeyInitrd         = "initrd"
	EngineKeyMMDSMessage    = "mmds_message"
)

// EngineSpec contains necessary parameters to execute a firecracker job.
type EngineSpec struct {
	// KernelImage is the path to the kernel image file.
	KernelImage string `json:"kernel_image,omitempty"`
	// InitrdPath is the path to the initial ramdisk file.
	Initrd string `json:"initrd_path,omitempty"`
	// KernelArgs is the kernel command line arguments.
	KernelArgs string `json:"kernel_args,omitempty"`
	// RootFileSystem is the path to the root file system.
	RootFileSystem string `json:"root_file_system,omitempty"`
	// MMDSMessage is the MMDS message to be sent to the Firecracker VM.
	MMDSMessage string `json:"mmds_message,omitempty"`
}

// Validate checks if the engine spec is valid
func (c EngineSpec) Validate() error {
	if validate.IsBlank(c.RootFileSystem) {
		return fmt.Errorf("invalid firecracker engine params: root_file_system cannot be empty")
	}
	if validate.IsBlank(c.KernelImage) {
		return fmt.Errorf("invalid firecracker engine params: kernel_image cannot be empty")
	}
	return nil
}

// DecodeSpec decodes a spec config into a firecracker engine spec
// It converts the params into a firecracker EngineSpec struct and validates it
func DecodeSpec(spec *types.SpecConfig) (EngineSpec, error) {
	if !spec.IsType(types.ExecutorTypeFirecracker.String()) {
		return EngineSpec{}, fmt.Errorf(
			"invalid firecracker engine type. expected %s, but received: %s",
			types.ExecutorTypeFirecracker,
			spec.Type,
		)
	}

	inputParams := spec.Params
	if inputParams == nil {
		return EngineSpec{}, fmt.Errorf("invalid firecracker engine params: params cannot be nil")
	}

	paramBytes, err := json.Marshal(inputParams)
	if err != nil {
		return EngineSpec{}, fmt.Errorf("failed to encode firecracker engine params: %w", err)
	}

	var firecrackerSpec *EngineSpec
	err = json.Unmarshal(paramBytes, &firecrackerSpec)
	if err != nil {
		return EngineSpec{}, fmt.Errorf("failed to decode firecracker engine params: %w", err)
	}

	return *firecrackerSpec, firecrackerSpec.Validate()
}

// EngineBuilder is a struct that is used for constructing an EngineSpec object
// specifically for Firecracker engines using the Builder pattern.
// It embeds an EngineBuilder object for handling the common builder methods.
type EngineBuilder struct {
	eb *types.SpecConfig
}

// NewFirecrackerEngineBuilder function initializes a new FirecrackerEngineBuilder instance.
// It sets the engine type to EngineFirecracker.String() and kernel image path as per the input argument.
func NewFirecrackerEngineBuilder(rootFileSystem string) *EngineBuilder {
	eb := types.NewSpecConfig(types.ExecutorTypeFirecracker.String())
	eb.WithParam(EngineKeyRootFileSystem, rootFileSystem)
	return &EngineBuilder{eb: eb}
}

// WithRootFileSystem is a builder method that sets the Firecracker engine root file system.
// It returns the FirecrackerEngineBuilder for further chaining of builder methods.
func (b *EngineBuilder) WithRootFileSystem(e string) *EngineBuilder {
	b.eb.WithParam(EngineKeyRootFileSystem, e)
	return b
}

// WithKernelImage is a builder method that sets the Firecracker engine kernel image.
// It returns the FirecrackerEngineBuilder for further chaining of builder methods.
func (b *EngineBuilder) WithKernelImage(e string) *EngineBuilder {
	b.eb.WithParam(EngineKeyKernelImage, e)
	return b
}

// WithInitrd is a builder method that sets the Firecracker init ram disk.
// It returns the FirecrackerEngineBuilder for further chaining of builder methods.
func (b *EngineBuilder) WithInitrd(e string) *EngineBuilder {
	b.eb.WithParam(EngineKeyInitrd, e)
	return b
}

// WithKernelArgs is a builder method that sets the Firecracker engine kernel arguments.
// It returns the FirecrackerEngineBuilder for further chaining of builder methods.
func (b *EngineBuilder) WithKernelArgs(e string) *EngineBuilder {
	b.eb.WithParam(EngineKeyKernelArgs, e)
	return b
}

// WithMMDSMessage is a builder method that sets the Firecracker engine MMDS message.
// It returns the FirecrackerEngineBuilder for further chaining of builder methods.
func (b *EngineBuilder) WithMMDSMessage(e string) *EngineBuilder {
	b.eb.WithParam(EngineKeyMMDSMessage, e)
	return b
}

// Build method constructs the final SpecConfig object by calling the embedded EngineBuilder's Build method.
func (b *EngineBuilder) Build() *types.SpecConfig {
	return b.eb
}
