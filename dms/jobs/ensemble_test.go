package jobs

import (
	"fmt"
	"testing"

	"gitlab.com/nunet/device-management-service/executor/docker"
	"gitlab.com/nunet/device-management-service/executor/firecracker"
	"gitlab.com/nunet/device-management-service/types"
	"gopkg.in/yaml.v2"
)

func TestGenerateEnsemble(t *testing.T) {
	ens := EnsembleConfigV1{
		Allocations: make(map[string]AllocationConfig),
		Nodes:       make(map[string]NodeConfig),
		Edges:       []EdgeConstraint{},
		Supervisor:  SupervisorConfig{},
		Keys:        make(map[string]string),
		Scripts:     make(map[string][]byte),
	}

	specdock := docker.NewDockerEngineBuilder("image1").WithWorkingDirectory("/").WithCmd("withCMD").WithEntrypoint("WithEntrypoint").WithEnvironment("env1").Build()

	ens.Allocations["alloc1"] = AllocationConfig{
		Executor: ExecutorDocker,
		Resources: types.Resources{
			CPU:  types.CPU{ClockSpeed: 2, Cores: 2, Threads: 2, Architecture: ""},
			RAM:  types.RAM{Size: 1024, ClockSpeed: 3, Type: ""},
			Disk: types.Disk{Size: 20000},
			GPUs: types.GPUs{
				{
					Vendor: "",
					Model:  "",
					VRAM:   12,
				},
			},
		},
		Execution: *specdock,
		DNSName:   "mydocker",
	}

	firecrackerspec := firecracker.NewFirecrackerEngineBuilder("/").WithInitrd("WithInitrd").WithKernelImage("WithInitrd").WithRootFileSystem("/").Build()
	ens.Allocations["alloc2"] = AllocationConfig{
		Executor: ExecutorFirecracker,
		Resources: types.Resources{
			CPU:  types.CPU{ClockSpeed: 2, Cores: 2, Threads: 2, Architecture: ""},
			RAM:  types.RAM{Size: 1024, ClockSpeed: 3, Type: ""},
			Disk: types.Disk{Size: 20000},
			GPUs: make(types.GPUs, 0),
		},
		Execution: *firecrackerspec,

		DNSName: "myfirecracker",
	}

	peerID := "peeridhere"

	ens.Nodes["node1"] = NodeConfig{
		Allocations: []string{"alloc1"},
		Ports: []PortConfig{{
			Public:     1,
			Private:    1,
			Allocation: "alloc1",
		}},
		Location: LocationConstraints{
			Accept: []Location{
				{
					Region: "Europe",
					ASN:    123,
				},
			},
		},
		Peer: peerID,
	}

	ens.Nodes["node2"] = NodeConfig{
		Allocations: []string{"alloc2"},
		Ports: []PortConfig{{
			Public:     1,
			Private:    1,
			Allocation: "alloc2",
		}},
		Location: LocationConstraints{},
		Peer:     peerID,
	}

	ens.Edges = []EdgeConstraint{{S: "node1", T: "node2", RTT: 2000, BW: 102410}}

	ens.Supervisor = SupervisorConfig{
		Strategy:    StrategyAllForOne,
		Allocations: []string{"alloc1"},
		Children:    []SupervisorConfig{{}},
	}

	yamlData, err := yaml.Marshal(&ens)
	if err != nil {
		fmt.Printf("Error encoding YAML: %v\n", err)
		return
	}

	t.Log(string(yamlData))

	fmt.Println(string(yamlData))
}
