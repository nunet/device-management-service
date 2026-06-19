// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package orchestrator

import (
	jtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

// Test constants
const (
	// Node names
	node1 = "node1"
	node2 = "node2"
	node3 = "node3"
	node4 = "node4"

	// Allocation names
	alloc1  = "alloc1"
	alloc2  = "alloc2"
	alloc3  = "alloc3"
	alloc4  = "alloc4"
	missing = "missing"

	// Key names
	key1   = "key1"
	value1 = "value1"

	// Script names
	script1 = "script1"
)

// Test data
var (
	// Script content
	scriptContent = []byte("echo hello")

	// Test scripts map
	testScripts = map[string][]byte{
		script1: scriptContent,
	}

	// Test keys map
	testKeys = map[string]string{
		key1: value1,
	}

	// Allocation configs
	Alloc1Cfg = jtypes.AllocationConfig{
		Type:     jtypes.AllocationTypeService,
		Executor: jtypes.ExecutorDocker,
		DNSName:  "service1",
	}

	Alloc2Cfg = jtypes.AllocationConfig{
		Type:     jtypes.AllocationTypeTask,
		Executor: jtypes.ExecutorDocker,
		DNSName:  "task1",
	}

	Alloc3Cfg = jtypes.AllocationConfig{
		Type:     jtypes.AllocationTypeService,
		Executor: jtypes.ExecutorContainerd,
		DNSName:  "service2",
	}

	Alloc4Cfg = jtypes.AllocationConfig{
		Type:     jtypes.AllocationTypeTask,
		Executor: jtypes.ExecutorContainerd,
		DNSName:  "task2",
	}

	// Node configs
	Node1Cfg = jtypes.NodeConfig{
		Allocations: []string{alloc1},
		Location: jtypes.LocationConstraints{
			Accept: []jtypes.Location{
				{Country: "US", City: "New York"},
			},
		},
	}

	Node2Cfg = jtypes.NodeConfig{
		Allocations: []string{alloc2},
		Location: jtypes.LocationConstraints{
			Accept: []jtypes.Location{
				{Country: "DE", City: "Berlin"},
			},
		},
	}

	Node3Cfg = jtypes.NodeConfig{
		Allocations: []string{alloc3, alloc4},
		Location: jtypes.LocationConstraints{
			Accept: []jtypes.Location{
				{Country: "JP", City: "Tokyo"},
			},
		},
	}

	Node4Cfg = jtypes.NodeConfig{
		Allocations: []string{},
		Location: jtypes.LocationConstraints{
			Accept: []jtypes.Location{
				{Country: "AU", City: "Sydney"},
			},
		},
	}
)
