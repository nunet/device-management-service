// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux

package jobs

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v2"

	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/executor/firecracker"
	"gitlab.com/nunet/device-management-service/types"
)

func TestGenerateEnsemble(t *testing.T) {
	ens := jobtypes.EnsembleConfigV1{
		Allocations: make(map[string]jobtypes.AllocationConfig),
		Nodes:       make(map[string]NodeConfig),
		Edges:       []EdgeConstraint{},
		Supervisor:  jobtypes.SupervisorConfig{},
		Keys:        make(map[string]string),
		Scripts:     make(map[string][]byte),
	}

	ens.EscalationStrategy = jobtypes.EscalationStrategyRedeploy

	specdock := docker.NewDockerEngineBuilder("image1").WithWorkingDirectory("/").WithCmd("withCMD").WithEntrypoint("WithEntrypoint").WithEnvironment("env1").Build()

	ens.Allocations["alloc1"] = jobtypes.AllocationConfig{
		Executor: jobtypes.ExecutorDocker,
		Resources: types.Resources{
			CPU:  types.CPU{ClockSpeed: 2, Cores: 2, Threads: 2, Architecture: ""},
			RAM:  types.RAM{Size: 4, ClockSpeed: 3, Type: ""},
			Disk: types.Disk{Size: 20},
			GPUs: types.GPUs{
				{
					Vendor: "",
					Model:  "",
					VRAM:   12,
				},
			},
		},
		Execution:       *specdock,
		DNSName:         "mydocker",
		FailureRecovery: jobtypes.AllocationFailureRecoveryOneForAll,
	}

	firecrackerspec := firecracker.NewFirecrackerEngineBuilder("/").WithInitrd("WithInitrd").WithKernelImage("WithInitrd").WithRootFileSystem("/").Build()
	ens.Allocations["alloc2"] = jobtypes.AllocationConfig{
		Executor: jobtypes.ExecutorFirecracker,
		Resources: types.Resources{
			CPU:  types.CPU{ClockSpeed: 2, Cores: 2, Threads: 2, Architecture: ""},
			RAM:  types.RAM{Size: 4, ClockSpeed: 3, Type: ""},
			Disk: types.Disk{Size: 20},
			GPUs: make(types.GPUs, 0),
		},
		Execution:       *firecrackerspec,
		DNSName:         "myfirecracker",
		FailureRecovery: jobtypes.AllocationFailureRecoveryOneForAll,
	}

	peerID := "peeridhere"

	ens.Nodes["node1"] = NodeConfig{
		Allocations:     []string{"alloc1"},
		Redundancy:      1,
		FailureRecovery: jobtypes.NodeFailureRecoveryStayDown,
		Ports: []jobtypes.PortConfig{{
			Public:     1,
			Private:    1,
			Allocation: "alloc1",
		}},
		Location: LocationConstraints{
			Accept: []Location{
				{
					Continent: "Europe",
					ASN:       123,
				},
			},
		},
		Peer: peerID,
	}

	ens.Nodes["node2"] = NodeConfig{
		Allocations:     []string{"alloc2"},
		Redundancy:      2,
		FailureRecovery: jobtypes.NodeFailureRecoveryStayDown,
		Ports: []jobtypes.PortConfig{{
			Public:     1,
			Private:    1,
			Allocation: "alloc2",
		}},
		Location: LocationConstraints{},
		Peer:     peerID,
	}

	ens.Edges = []EdgeConstraint{{S: "node1", T: "node2", RTT: 2000, BW: 102410}}

	ens.Supervisor = jobtypes.SupervisorConfig{
		Strategy:    jobtypes.StrategyAllForOne,
		Allocations: []string{"alloc1"},
		Children:    []jobtypes.SupervisorConfig{{}},
	}

	yamlData, err := yaml.Marshal(&ens)
	if err != nil {
		t.Fatal(err)
	}

	t.Log(string(yamlData))

	fmt.Println(string(yamlData))
}
