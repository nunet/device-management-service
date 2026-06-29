// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package containerd

import (
	"encoding/json"
	"fmt"
	"strings"

	containernetns "github.com/containerd/containerd/v2/pkg/netns"
	"github.com/containernetworking/cni/libcni"
	"gitlab.com/nunet/device-management-service/types"
	"gitlab.com/nunet/device-management-service/utils/validate"
)

const (
	EngineKeyImage            = "image"
	EngineKeyEntrypoint       = "entrypoint"
	EngineKeyCmd              = "cmd"
	EngineKeyEnvironment      = "environment"
	EngineKeyWorkingDirectory = "working_directory"
	EngineKeyRuntime          = "runtime"
)

const (
	DefaultRuntime = "io.containerd.runc.v2"
	KataRuntime    = "io.containerd.kata.v2"
)

const (
	DefaultCNIIfName = "eth0"
)

type EngineSpec struct {
	Image            string   `json:"image,omitempty" yaml:"image,omitempty"`
	Entrypoint       []string `json:"entrypoint,omitempty" yaml:"entrypoint,omitempty"`
	Cmd              []string `json:"cmd,omitempty" yaml:"cmd,omitempty"`
	Environment      []string `json:"environment,omitempty" yaml:"environment,omitempty"`
	WorkingDirectory string   `json:"working_directory,omitempty" yaml:"working_directory,omitempty"`
	Runtime          string   `json:"runtime,omitempty" yaml:"runtime,omitempty"`
}

type portMapping struct {
	HostPort      int    `json:"hostPort"`
	ContainerPort int    `json:"containerPort"`
	Protocol      string `json:"protocol"`
	HostIP        string `json:"hostIP,omitempty"`
}

type networkSetup struct {
	containerID string
	netNS       *containernetns.NetNS
	netConfList *libcni.NetworkConfigList
	runtimeConf *libcni.RuntimeConf
	netInfo     types.ExecutorNetInfo
}

func (c EngineSpec) Validate() error {
	if validate.IsBlank(c.Image) {
		return fmt.Errorf("invalid containerd engine params: image cannot be empty")
	}
	return nil
}

func (c EngineSpec) RuntimeName() string {
	runtimeName := strings.TrimSpace(strings.ToLower(c.Runtime))
	switch runtimeName {
	case "", "runc", DefaultRuntime:
		return DefaultRuntime
	case "kata", KataRuntime:
		return KataRuntime
	default:
		return c.Runtime
	}
}

func DecodeSpec(spec *types.SpecConfig) (EngineSpec, error) {
	if spec == nil {
		return EngineSpec{}, fmt.Errorf("invalid containerd engine spec: spec cannot be nil")
	}

	if !spec.IsType(string(types.ExecutorTypeContainerd)) {
		return EngineSpec{}, fmt.Errorf(
			"invalid containerd engine type. expected %s, but received: %s",
			types.ExecutorTypeContainerd,
			spec.Type,
		)
	}

	if spec.Params == nil {
		return EngineSpec{}, fmt.Errorf("invalid containerd engine params: params cannot be nil")
	}

	paramBytes, err := json.Marshal(spec.Params)
	if err != nil {
		return EngineSpec{}, fmt.Errorf("failed to encode containerd engine params: %w", err)
	}

	var containerdSpec *EngineSpec
	if err := json.Unmarshal(paramBytes, &containerdSpec); err != nil {
		return EngineSpec{}, fmt.Errorf("failed to decode containerd engine params: %w", err)
	}

	if containerdSpec.Runtime == "" {
		containerdSpec.Runtime = DefaultRuntime
	}

	return *containerdSpec, containerdSpec.Validate()
}

type EngineBuilder struct {
	eb *types.SpecConfig
}

func NewContainerdEngineBuilder(image string) *EngineBuilder {
	eb := types.NewSpecConfig(string(types.ExecutorTypeContainerd))
	eb.WithParam(EngineKeyImage, image)
	eb.WithParam(EngineKeyRuntime, DefaultRuntime)
	return &EngineBuilder{eb: eb}
}

func (b *EngineBuilder) WithEntrypoint(e ...string) *EngineBuilder {
	b.eb.WithParam(EngineKeyEntrypoint, e)
	return b
}

func (b *EngineBuilder) WithCmd(c ...string) *EngineBuilder {
	b.eb.WithParam(EngineKeyCmd, c)
	return b
}

func (b *EngineBuilder) WithEnvironment(e ...string) *EngineBuilder {
	b.eb.WithParam(EngineKeyEnvironment, e)
	return b
}

func (b *EngineBuilder) WithWorkingDirectory(w string) *EngineBuilder {
	b.eb.WithParam(EngineKeyWorkingDirectory, w)
	return b
}

func (b *EngineBuilder) WithRuntime(runtime string) *EngineBuilder {
	b.eb.WithParam(EngineKeyRuntime, runtime)
	return b
}

func (b *EngineBuilder) Build() *types.SpecConfig {
	return b.eb
}
