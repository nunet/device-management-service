// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package ensemblev1

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

// TODO: avoid asserting error string messages.
// Only assert error msgs if using sentinel errors
func TestValidateSpec(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		spec     any
		wantErr  bool
		errorMsg string
	}{
		{
			// Basic structural validation - only requires allocations to be present with at least one entry
			name: "valid minimal spec",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{
						"type": "service",
					},
				},
			},
			wantErr: false,
		},
		{
			// Optional fields (nodes, edges) can be empty arrays
			name: "valid spec with empty nodes and edges",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{
						"type": "service",
					},
				},
				"nodes": []any{},
				"edges": []any{},
			},
			wantErr: false,
		},
		{
			// Spec must be a valid map type for structural access
			name:     "nil spec",
			spec:     nil,
			wantErr:  true,
			errorMsg: "invalid spec configuration",
		},
		{
			// Required field 'allocations' must exist
			name:     "empty spec",
			spec:     map[string]any{},
			wantErr:  true,
			errorMsg: "at least one allocation must be defined",
		},
		{
			// Allocations must contain at least one entry
			name: "empty allocations",
			spec: map[string]any{
				"allocations": map[string]any{},
			},
			wantErr:  true,
			errorMsg: "at least one allocation must be defined",
		},
		{
			// Duplicate allocation names (case-insensitive) should be rejected
			name: "duplicate allocation names",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{
						"type": "service",
					},
					"Alloc1": map[string]any{
						"type": "service",
					},
				},
			},
			wantErr:  true,
			errorMsg: "duplicate allocation names found",
		},
		{
			// Duplicate dns_name should be rejected
			name: "duplicate dns_name",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{
						"type":     "service",
						"dns_name": "shared.example.com",
					},
					"alloc2": map[string]any{
						"type":     "service",
						"dns_name": "shared.example.com",
					},
				},
			},
			wantErr:  true,
			errorMsg: "duplicate dns_name found",
		},
		{
			// Unique dns_name values should pass
			name: "unique dns_name",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{
						"type":     "service",
						"dns_name": "a.example.com",
					},
					"alloc2": map[string]any{
						"type":     "service",
						"dns_name": "b.example.com",
					},
					"alloc3": map[string]any{
						"type": "service",
					}, // missing dns_name is fine
				},
			},
			wantErr: false,
		},
		{
			// When edges exist, nodes field must be present (empty nodes map is valid)
			// Actual node references will be validated at a later stage
			name: "edges with empty nodes map",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{
						"type": "service",
					},
				},
				"nodes": map[string]any{},
				"edges": []any{
					map[string]any{},
				},
			},
			wantErr: false,
		},
		{
			// When edges exist, nodes field must be present
			// This is a structural requirement, not a reference validation
			name: "edges without nodes",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{},
				},
				"edges": []any{
					map[string]any{},
				},
			},
			wantErr:  true,
			errorMsg: "nodes must be defined when edge_constraints are present",
		},
		{
			// escalation_strategy must be one of the supported values
			name: "unsupported escalation strategy",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{},
				},
				"escalation_strategy": "invalid",
			},
			wantErr:  true,
			errorMsg: "invalid escalation_strategy",
		},
		{
			// valid escalation strategy
			name: "valid escalation strategy",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{},
				},
				"escalation_strategy": "redeploy",
			},
			wantErr: false,
		},
		{
			// cyclic dependency detected
			name: "cyclic dependency detected",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{
						"type":       "service",
						"depends_on": []any{"alloc2", "alloc3"},
					},
					"alloc2": map[string]any{
						"type":       "service",
						"depends_on": []any{"alloc3"},
					},
					"alloc3": map[string]any{
						"type":       "service",
						"depends_on": []any{"alloc1"},
					},
				},
			},
			wantErr:  true,
			errorMsg: "cyclic dependencies detected",
		},
		{
			// allocation assigned to multiple nodes
			name: "allocation assigned to multiple nodes",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{
						"type": "service",
					},
					"alloc2": map[string]any{
						"type": "service",
					},
				},
				"nodes": map[string]any{
					"node1": map[string]any{
						"allocations": []any{"alloc1"},
					},
					"node2": map[string]any{
						"allocations": []any{"alloc1", "alloc2"},
					},
				},
			},
			wantErr:  true,
			errorMsg: "allocation 'alloc1' is assigned to multiple nodes",
		},
		{
			// dependent allocations in different nodes
			name: "dependent allocations in different nodes",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{
						"type":       "service",
						"depends_on": []any{"alloc2"},
					},
					"alloc2": map[string]any{
						"type": "service",
					},
				},
				"nodes": map[string]any{
					"node1": map[string]any{
						"allocations": []any{"alloc1"},
					},
					"node2": map[string]any{
						"allocations": []any{"alloc2"},
					},
				},
			},
			wantErr:  true,
			errorMsg: "allocation 'alloc1' depends on 'alloc2', but 'alloc2' is not in the same node",
		},
		{
			// valid configuration with dependencies in the same node
			name: "valid configuration with dependencies in the same node",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{
						"type":       "service",
						"depends_on": []any{"alloc2"},
					},
					"alloc2": map[string]any{
						"type": "service",
					},
				},
				"nodes": map[string]any{
					"node1": map[string]any{
						"allocations": []any{"alloc1", "alloc2"},
					},
				},
			},
			wantErr: false,
		},
		{
			// valid configuration with dependency not in any node
			name: "valid configuration with dependency not in any node",
			spec: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{
						"type":       "service",
						"depends_on": []any{"alloc2"},
					},
					"alloc2": map[string]any{
						"type": "service",
					},
				},
				"nodes": map[string]any{
					"node1": map[string]any{
						"allocations": []any{"alloc1"},
					},
				},
			},
			wantErr:  true,
			errorMsg: "allocation 'alloc1' depends on 'alloc2', but 'alloc2' is not in the same node",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSpec(nil, tt.spec, "")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateContract(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		contract any
		wantErr  bool
		errorMsg string
	}{
		{
			name: "valid contract",
			contract: map[string]any{
				"did": "did:example:1",
			},
			wantErr: false,
		},
		{
			name:     "nil contract",
			contract: nil,
			wantErr:  true,
			errorMsg: "invalid contract configuration:",
		},
		{
			name:     "invalid contract type",
			contract: "not a map",
			wantErr:  true,
			errorMsg: "invalid contract configuration: not a map",
		},
		{
			name: "invalid did type",
			contract: map[string]any{
				"did": 123,
			},
			wantErr:  true,
			errorMsg: "contract 'did' must be a string",
		},
		{
			name: "invalid did format",
			contract: map[string]any{
				"did": "invalid-did",
			},
			wantErr:  true,
			errorMsg: "invalid did format",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateContract(nil, tt.contract, "")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateAllocation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		root     map[string]any
		alloc    any
		path     tree.Path
		wantErr  bool
		errorMsg string
	}{
		{
			// Basic structural validation - requires execution, executor, and resources
			// Only execution with a type and executor are required to be non empty at this level
			// Detailed validation checks for each are done in their own validators
			name: "valid minimal docker allocation",
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
			},
			wantErr: false,
		},
		{
			// Allocation must be a valid map type for structural access
			name:     "nil allocation",
			alloc:    nil,
			wantErr:  true,
			errorMsg: "invalid allocation configuration",
		},
		{
			// Required field 'execution' must exist
			name: "missing execution",
			alloc: map[string]any{
				"type":             "task",
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
			},
			wantErr:  true,
			errorMsg: "allocation must have an execution",
		},
		{
			// Required field 'executor' must exist
			name: "missing executor",
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
			},
			wantErr:  true,
			errorMsg: "allocation must have an executor",
		},
		{
			// Required field 'type' must exist
			name: "missing executor",
			alloc: map[string]any{
				"executor": "docker",
				"execution": map[string]any{
					"type": "docker",
				},
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
			},
			wantErr:  true,
			errorMsg: "allocation must have a type",
		},
		{
			// Required field 'resources' must exist
			name: "missing resources",
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"failure_recovery": defaultAllocationFailureStrategy,
			},
			wantErr:  true,
			errorMsg: "allocation must have resources",
		},
		{
			// Execution must have a required field 'type'
			name: "missing execution type",
			alloc: map[string]any{
				"type":             "task",
				"execution":        map[string]any{},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
			},
			wantErr:  true,
			errorMsg: "execution must have a type",
		},
		{
			// Execution type must match executor
			name: "mismatched execution type and executor",
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "firecracker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
			},
			wantErr:  true,
			errorMsg: "must match execution type",
		},
		{
			// If DNS is provided, it must be valid
			name: "invalid DNS name",
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"dns_name":         "invalid..dns",
			},
			wantErr:  true,
			errorMsg: "invalid dns_name",
		},
		{
			// Valid DNS should pass
			name: "valid DNS name",
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"dns_name":         "valid-dns.com",
			},
			wantErr: false,
		},
		{
			// Empty keys reference should pass without validation
			name: "empty keys reference",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"keys":             []any{},
			},
			wantErr: false,
		},
		{
			// wrong data for key
			name: "wrong data for key",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"keys": []any{
					"not a map",
					"another not a map",
				},
			},
			wantErr:  true,
			errorMsg: "must be a map",
		},
		{
			// key type not specified
			name: "key type not specified",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"keys": []any{
					map[string]any{
						"file": "keyfile",
						"dest": "keydest",
					},
				},
			},
			wantErr:  true,
			errorMsg: "must have a type",
		},
		{
			// unknown key type
			name: "unknown key type",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"keys": []any{
					map[string]any{
						"type": "invalid-type",
						"file": "keyfile",
						"dest": "keydest",
					},
				},
			},
			wantErr:  true,
			errorMsg: "has invalid type",
		},
		{
			// empty key file path
			name: "empty key file path",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"keys": []any{
					map[string]any{
						"type": "ssh",
						"file": "",
						"dest": "keydest",
					},
				},
			},
			wantErr:  true,
			errorMsg: "is empty",
		},
		{
			// no key dest when type isn't ssh
			name: "no key dest and no execution user specified",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"keys": []any{
					map[string]any{
						"type": "gpg",
						"file": "/key/file/path",
						"dest": "",
					},
				},
			},
			wantErr:  true,
			errorMsg: "missing a destination",
		},
		{
			// key destination specified
			name: "not ssh key but key destination specified",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"keys": []any{
					map[string]any{
						"type": "gpg",
						"file": "/key/file/path",
						"dest": "/destination/path",
					},
				},
			},
			wantErr: false,
		},
		{
			// provision scripts reference not defined in spec
			name: "provision scripts reference not defined",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"provision":        []any{"valid-script"},
			},
			wantErr:  true,
			errorMsg: "scripts must be defined",
		},
		{
			// Provision scripts reference not found in spec
			name: "provision scripts reference not found",
			root: map[string]any{
				"scripts": map[string]any{
					"valid-script": "script",
				},
			},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"provision":        []any{"non-existent-script"},
			},
			wantErr:  true,
			errorMsg: "script 'non-existent-script' not found",
		},
		{
			// Empty provision reference should pass without validation
			name: "empty provision reference",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"provision":        []any{},
			},
			wantErr: false,
		},
		{
			// Valid provision reference
			name: "valid provision reference",
			root: map[string]any{
				"scripts": map[string]any{
					"valid-script": map[string]any{},
				},
			},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"provision":        "valid-script",
			},
			wantErr: false,
		},
		{
			// failure recovery not defined
			name: "failure recovery not defined",
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":  "docker",
				"resources": map[string]any{},
				"provision": "valid-script",
			},
			wantErr:  true,
			errorMsg: "failure_recovery must be specified",
		},
		{
			// failure recovery must be one of the supported options
			name: "unsupported failure recovery",
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": "invalid",
			},
			wantErr:  true,
			errorMsg: "invalid failure_recovery",
		},
		{
			// valid failure recovery
			name: "valid failure recovery",
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
			},
			wantErr: false,
		},
		{
			// depends on must be a list of allocations
			name: "depends on must be a list of allocations",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"depends_on":       "alloc1",
			},
			wantErr:  true,
			errorMsg: "depends_on must be a list",
		},
		{
			// depends on must be a valid reference
			name: "depends on must be a valid reference",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"depends_on":       []any{"invalid"},
			},
			wantErr:  true,
			errorMsg: "not found",
		},
		{
			// depends on can not reference itself
			name: "depends on can not reference itself",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"depends_on":       []any{"alloc1"},
			},
			path:     tree.NewPath("V1", "allocations", "alloc1"),
			wantErr:  true,
			errorMsg: "depends_on must not refer to itself",
		},
		{
			// valid depends on
			name: "valid depends on",
			root: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{},
				},
			},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"depends_on":       []any{"alloc1"},
			},
			path:    tree.NewPath("V1", "allocations", "alloc2"),
			wantErr: false,
		},
		{
			// failure recovery not defined
			name: "failure recovery not defined",
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":  "docker",
				"resources": map[string]any{},
			},
			wantErr:  true,
			errorMsg: "failure_recovery must be specified",
		},
		{
			// failure recovery must be one of the supported options
			name: "unsupported failure recovery",
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": "invalid",
			},
			wantErr:  true,
			errorMsg: "invalid failure_recovery",
		},
		{
			// valid failure recovery
			name: "valid failure recovery",
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
			},
			wantErr: false,
		},
		{
			// depends on must be a list of allocations
			name: "depends on must be a list of allocations",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"depends_on":       "alloc1",
			},
			wantErr:  true,
			errorMsg: "depends_on must be a list",
		},
		{
			// depends on must be a valid reference
			name: "depends on must be a valid reference",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"depends_on":       []any{"invalid"},
			},
			wantErr:  true,
			errorMsg: "not found",
		},
		{
			// depends on can not reference itself
			name: "depends on can not reference itself",
			root: map[string]any{},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"depends_on":       []any{"alloc1"},
			},
			path:     tree.NewPath("V1", "allocations", "alloc1"),
			wantErr:  true,
			errorMsg: "depends_on must not refer to itself",
		},
		{
			// valid depends on
			name: "valid depends on",
			root: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{},
				},
			},
			alloc: map[string]any{
				"type": "task",
				"execution": map[string]any{
					"type": "docker",
				},
				"executor":         "docker",
				"resources":        map[string]any{},
				"failure_recovery": defaultAllocationFailureStrategy,
				"depends_on":       []any{"alloc1"},
			},
			path:    tree.NewPath("V1", "allocations", "alloc2"),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.root != nil && tt.root["V1"] == nil {
				tt.root["V1"] = tt.root
			}
			err := ValidateAllocation(&tt.root, tt.alloc, tt.path)
			if tt.wantErr {
				assert.Error(t, err, "expected error: %q", tt.errorMsg)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateResources(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		data     any
		wantErr  bool
		errorMsg string
	}{
		{
			// Only required fields are provided: cpu cores, ram size, and disk size
			// This represents the minimum valid configuration for resources
			name: "valid minimal resources",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
			},
			wantErr: false,
		},
		{
			name: "valid resources with optional fields",
			data: map[string]any{
				"cpu": map[string]any{
					"cores":        2,
					"vendor":       "intel",
					"architecture": "x86_64",
					"clock_speed":  2.4,
				},
				"ram": map[string]any{
					"size":        4096,
					"clock_speed": 3200,
					"type":        "ddr4",
				},
				"disk": map[string]any{
					"size":       10000,
					"type":       "ssd",
					"read_speed": 500,
				},
				"gpus": []any{
					map[string]any{
						"memory": 8192,
						"vendor": "nvidia",
						"model":  "rtx3080",
					},
				},
			},
			wantErr: false,
		},
		{
			// Resources must be a map type for structural validation			name:    "invalid resources type",
			data:     "invalid",
			wantErr:  true,
			errorMsg: "invalid resources configuration",
		},
		{
			// CPU configuration is required and must be a map
			name: "missing cpu configuration",
			data: map[string]any{
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
			},
			wantErr:  true,
			errorMsg: "resources must have cpu configuration",
		},
		{
			// CPU cores is a required field within CPU configuration
			name: "missing cpu cores",
			data: map[string]any{
				"cpu": map[string]any{},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
			},
			wantErr:  true,
			errorMsg: "cpu must have cores value",
		},
		{
			// CPU cores must be a positive number
			name: "negative cpu cores",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": -2,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
			},
			wantErr:  true,
			errorMsg: "cpu cores must be positive",
		},
		{
			// TODO: This behavior might be reconsidered
			// CPU cores can be decimal but will be rounded up
			name: "decimal cpu cores should round up",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2.7,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
			},
			wantErr: false,
		},
		{
			// CPU arch is optional but if specified must not be empty
			name: "empty cpu architecture",
			data: map[string]any{
				"cpu": map[string]any{
					"cores":        2,
					"architecture": "",
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
			},
			wantErr:  true,
			errorMsg: "cpu architecture cannot be empty if specified",
		},
		{
			// CPU frequency is optional but if specified must be a positive number
			name: "negative cpu frequency",
			data: map[string]any{
				"cpu": map[string]any{
					"cores":       2,
					"clock_speed": -2.4,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
			},
			wantErr:  true,
			errorMsg: "cpu clock_speed must be positive",
		},
		{
			// RAM configuration is required and must be a map
			name: "missing ram configuration",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"disk": map[string]any{
					"size": 10000,
				},
			},
			wantErr:  true,
			errorMsg: "resources must have ram configuration",
		},
		{
			// RAM size is required within RAM configuration
			name: "missing ram size",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{},
				"disk": map[string]any{
					"size": 10000,
				},
			},
			wantErr:  true,
			errorMsg: "ram must have size value",
		},
		{
			// RAM size must be a positive number
			name: "negative ram size",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size": -4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
			},
			wantErr:  true,
			errorMsg: "ram size must be positive",
		},
		{
			// RAM speed is optional but if specified must be a positive number
			name: "negative ram speed",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size":        4096,
					"clock_speed": -3200,
				},
				"disk": map[string]any{
					"size": 10000,
				},
			},
			wantErr:  true,
			errorMsg: "ram clock_speed must be positive",
		},
		{
			// Disk configuration is required and must be a map
			name: "missing disk configuration",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size": 4096,
				},
			},
			wantErr:  true,
			errorMsg: "resources must have disk configuration",
		},
		{
			// Disk size is required within disk configuration
			name: "missing disk size",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{},
			},
			wantErr:  true,
			errorMsg: "disk must have size value",
		},
		{
			// Disk size must be a positive number
			name: "negative disk size",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": -10000,
				},
			},
			wantErr:  true,
			errorMsg: "disk size must be positive",
		},
		{
			// Disk type is optional but if specified must not be empty
			name: "empty disk type",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
					"type": "",
				},
			},
			wantErr:  true,
			errorMsg: "disk type cannot be empty if specified",
		},
		{
			// GPUs must be an array if specified
			name: "invalid gpus type",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
				"gpus": "invalid",
			},
			wantErr:  true,
			errorMsg: "gpus must be an array",
		},
		{
			// Each GPU in the array must be a map
			name: "invalid gpu configuration",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
				"gpus": []any{"invalid"},
			},
			wantErr:  true,
			errorMsg: "gpu at index 0 is not a valid configuration",
		},
		{
			// GPU memory if specified must be positive
			name: "negative gpu memory",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
				"gpus": []any{
					map[string]any{
						"memory": -8192,
					},
				},
			},
			wantErr:  true,
			errorMsg: "gpu memory at index 0 must be positive",
		},
		{
			// GPU vendor if specified must not be empty
			name: "empty gpu vendor",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
				"gpus": []any{
					map[string]any{
						"memory": 8192,
						"vendor": "",
					},
				},
			},
			wantErr:  true,
			errorMsg: "gpu vendor cannot be empty if specified at index 0",
		},
		{
			// GPU model if specified must not be empty
			name: "empty gpu model",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
				"gpus": []any{
					map[string]any{
						"memory": 8192,
						"model":  "",
					},
				},
			},
			wantErr:  true,
			errorMsg: "gpu model cannot be empty if specified at index 0",
		},
		{
			// Resources should accept different numeric types (string, float, int)
			// All valid numeric values should be properly converted and validated
			name: "valid resources with different numeric types",
			data: map[string]any{
				"cpu": map[string]any{
					"cores":       "2",
					"clock_speed": 2.4,
				},
				"ram": map[string]any{
					"size":        "4096",
					"clock_speed": 3200.0,
				},
				"disk": map[string]any{
					"size": "0.1e4",
				},
			},
			wantErr: false,
		},
		// Invalid numeric types should fail validation
		{
			name: "invalid numeric type",
			data: map[string]any{
				"cpu": map[string]any{
					"cores":       "1",
					"clock_speed": "2.4",
				},
				"ram": map[string]any{
					"size":        "4096",
					"clock_speed": "3200",
				},
				"disk": map[string]any{
					"size": "invalid",
				},
			},
			wantErr:  true,
			errorMsg: "disk size must be a valid number",
		},
		{
			// Multiple GPUs can be specified with different valid configurations
			// All GPUs should pass validation independently
			name: "multiple gpus with mixed configurations",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
				"gpus": []any{
					map[string]any{
						"memory": 8192,
						"vendor": "nvidia",
						"model":  "rtx3080",
					},
					map[string]any{
						"memory": 16384,
						"vendor": "amd",
						"model":  "rx6900xt",
					},
				},
			},
			wantErr: false,
		},
		{
			// When multiple GPUs are specified, validation should fail if any GPU is invalid
			// First GPU is valid, second has negative memory which should fail
			name: "mixed valid and invalid gpu configurations",
			data: map[string]any{
				"cpu": map[string]any{
					"cores": 2,
				},
				"ram": map[string]any{
					"size": 4096,
				},
				"disk": map[string]any{
					"size": 10000,
				},
				"gpus": []any{
					map[string]any{
						"memory": 8192,
						"vendor": "nvidia",
						"model":  "rtx3080",
					},
					map[string]any{
						"memory": -16384, // Negative memory should fail
						"vendor": "amd",
						"model":  "rx6900xt",
					},
				},
			},
			wantErr:  true,
			errorMsg: "gpu memory at index 1 must be positive",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateResources(nil, tt.data, "")
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateNode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		node        map[string]any
		root        map[string]any
		path        tree.Path
		expectError bool
		errorMsg    string
	}{
		{
			// Valid node with all required fields
			name: "valid node",
			node: map[string]any{
				"allocations":      []any{"alloc1", "alloc2"},
				"failure_recovery": defaultAllocationFailureStrategy,
				"ports": []any{
					map[string]any{
						"public":     8080,
						"private":    80,
						"allocation": "alloc1",
					},
				},
				"location": map[string]any{
					"accept": []any{
						map[string]any{
							"continent": "NA",
							"country":   "US",
							"city":      "San Francisco",
						},
					},
				},
				"peer": "peer1",
			},
			root: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{
						"type": "service",
					},
					"alloc2": map[string]any{
						"type": "service",
					},
				},
			},
			path:        tree.NewPath("nodes", "node1"),
			expectError: false,
		},
		{
			// Missing public port when private port is specified
			name: "missing public port",
			node: map[string]any{
				"failure_recovery": defaultNodeFailureStrategy,
				"ports": []any{
					map[string]any{
						"private":    8080,
						"allocation": "alloc1",
					},
				},
			},
			root:        map[string]any{},
			path:        tree.NewPath("nodes", "node1"),
			expectError: true,
			errorMsg:    "public port must be specified when private port is defined",
		},
		{
			// Invalid port number
			name: "invalid port number",
			node: map[string]any{
				"failure_recovery": defaultNodeFailureStrategy,
				"ports": []any{
					map[string]any{
						"public":     80,
						"private":    -1,
						"allocation": "alloc1",
					},
				},
			},
			root:        map[string]any{},
			path:        tree.NewPath("nodes", "node1"),
			expectError: true,
			errorMsg:    "port must have a valid private port number",
		},
		{
			// Invalid location
			name: "invalid location",
			node: map[string]any{
				"failure_recovery": defaultNodeFailureStrategy,
				"location": map[string]any{
					"accept": []any{
						map[string]any{
							"city": "San Francisco",
							// missing required continent
						},
					},
				},
			},
			root:        map[string]any{},
			path:        tree.NewPath("nodes", "node1"),
			expectError: true,
			errorMsg:    "invalid location constraints: invalid accept location at index 0: continent is required and cannot be empty",
		},
		{
			// Allocation not found
			name: "allocation not found",
			node: map[string]any{
				"allocations":      []any{"nonexistent"},
				"failure_recovery": defaultNodeFailureStrategy,
			},
			root: map[string]any{
				"allocations": map[string]any{},
			},
			path:        tree.NewPath("nodes", "node1"),
			expectError: true,
			errorMsg:    "referenced allocation 'nonexistent' not found",
		},
		{
			// Invalid allocation reference type
			name: "invalid allocation reference type",
			node: map[string]any{
				"allocations":      []any{123},
				"failure_recovery": defaultNodeFailureStrategy,
			},
			root: map[string]any{
				"allocations": map[string]any{},
			},
			path:        tree.NewPath("nodes", "node1"),
			expectError: true,
			errorMsg:    "invalid allocation reference: 123",
		},
		{
			// redundancy should be a number
			name: "invalid redundancy type",
			node: map[string]any{
				"redundancy":       "invalid",
				"failure_recovery": defaultNodeFailureStrategy,
			},
			expectError: true,
			errorMsg:    "redundancy must be a number",
		},
		{
			// redundancy should be positive
			name: "redundancy should be a positive number",
			node: map[string]any{
				"redundancy":       -1,
				"failure_recovery": defaultNodeFailureStrategy,
			},
			expectError: true,
			errorMsg:    "redundancy must be a positive number",
		},
		{
			name: "redundancy should be greater than 0",
			node: map[string]any{
				"redundancy":       -1,
				"failure_recovery": defaultNodeFailureStrategy,
			},
			expectError: true,
			errorMsg:    "redundancy must be a positive number",
		},
		{
			// valid redundancy
			name: "valid redundancy",
			node: map[string]any{
				"redundancy":       1,
				"failure_recovery": defaultNodeFailureStrategy,
			},
			expectError: false,
		},
		{
			// failure recovery not defined
			name:        "failure recovery not defined",
			node:        map[string]any{},
			expectError: true,
			errorMsg:    "failure_recovery must be specified",
		},
		{
			// failure recovery should be one of the supported options
			name: "unsupported failure recovery",
			node: map[string]any{
				"failure_recovery": "invalid",
			},
			expectError: true,
			errorMsg:    "invalid failure_recovery",
		},
		{
			// valid failure_recovery
			name: "valid failure_recovery",
			node: map[string]any{
				"failure_recovery": defaultNodeFailureStrategy,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.root != nil && tt.root["V1"] == nil {
				tt.root["V1"] = tt.root
			}
			err := ValidateNode(&tt.root, tt.node, tt.path)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateExecution(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		execution   map[string]any
		expectError bool
		errorMsg    string
	}{
		{
			// Valid docker execution with all optional fields
			name: "valid docker execution",
			execution: map[string]any{
				"type": "docker",
				"params": map[string]any{
					"image": "ubuntu:latest",
					"environment": []any{
						"KEY1=value1",
						"KEY2=value2",
						"_TEST=value3",
					},
					"working_directory": "/app",
					"entrypoint": []any{
						"sh",
						"-c",
					},
					"cmd": []any{
						"echo",
						"hello",
					},
				},
			},
			expectError: false,
		},
		{
			// Missing type
			name: "missing type",
			execution: map[string]any{
				"params": map[string]any{
					"image": "ubuntu:latest",
				},
			},
			expectError: true,
			errorMsg:    "execution must have a type",
		},
		{
			// Docker missing image
			name: "docker missing image",
			execution: map[string]any{
				"type":   "docker",
				"params": map[string]any{},
			},
			expectError: true,
			errorMsg:    "docker execution must have an image",
		},
		{
			// Invalid docker image format
			name: "invalid docker image format",
			execution: map[string]any{
				"type": "docker",
				"params": map[string]any{
					"image": "INVALID/Image/Format:tag!",
				},
			},
			expectError: true,
			errorMsg:    "invalid docker image format: INVALID/Image/Format:tag!",
		},
		{
			// Docker invalid environment variable format
			name: "docker invalid environment variable format",
			execution: map[string]any{
				"type": "docker",
				"params": map[string]any{
					"image": "ubuntu:latest",
					"environment": []any{
						"invalid_format",
					},
				},
			},
			expectError: true,
			errorMsg:    "docker environment variable at index 0 must be in KEY=VALUE format",
		},
		{
			// Docker invalid environment variable key
			name: "docker invalid environment variable key",
			execution: map[string]any{
				"type": "docker",
				"params": map[string]any{
					"image": "ubuntu:latest",
					"environment": []any{
						"123KEY=value",
					},
				},
			},
			expectError: true,
			errorMsg:    "invalid environment variable key format at index 0: 123KEY",
		},
		{
			// Valid firecracker execution with all fields
			name: "valid firecracker execution",
			execution: map[string]any{
				"type": "firecracker",
				"params": map[string]any{
					"kernel_image":     "/path/to/kernel",
					"root_file_system": "/path/to/rootfs",
					"kernel_args":      "console=ttyS0",
					"initrd":           "/path/to/initrd",
				},
			},
			expectError: false,
		},
		{
			// Firecracker missing kernel image
			name: "firecracker missing kernel image",
			execution: map[string]any{
				"type": "firecracker",
				"params": map[string]any{
					"root_file_system": "/path/to/rootfs",
				},
			},
			expectError: true,
			errorMsg:    "firecracker execution must have a kernel_image",
		},
		{
			// Firecracker missing root file system
			name: "firecracker missing root file system",
			execution: map[string]any{
				"type": "firecracker",
				"params": map[string]any{
					"kernel_image": "/path/to/kernel",
				},
			},
			expectError: true,
			errorMsg:    "firecracker execution must have a root_file_system",
		},
		{
			// Firecracker empty kernel args
			name: "firecracker empty kernel args",
			execution: map[string]any{
				"type": "firecracker",
				"params": map[string]any{
					"kernel_image":     "/path/to/kernel",
					"root_file_system": "/path/to/rootfs",
					"kernel_args":      "",
				},
			},
			expectError: true,
			errorMsg:    "firecracker kernel_args cannot be empty if specified",
		},
		{
			// Firecracker empty initrd
			name: "firecracker empty initrd",
			execution: map[string]any{
				"type": "firecracker",
				"params": map[string]any{
					"kernel_image":     "/path/to/kernel",
					"root_file_system": "/path/to/rootfs",
					"initrd":           "",
				},
			},
			expectError: true,
			errorMsg:    "firecracker initrd cannot be empty if specified",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateExecution(nil, tt.execution, "")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSupervisor(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		supervisor  map[string]any
		root        map[string]any
		expectError bool
		errorMsg    string
	}{
		{
			// Valid supervisor with all required fields
			name: "valid supervisor",
			supervisor: map[string]any{
				"strategy":    "OneForOne",
				"allocations": []any{"alloc1", "alloc2"},
				"children": []any{
					map[string]any{
						"strategy":    "AllForOne",
						"allocations": []any{"alloc3"},
					},
				},
			},
			root: map[string]any{
				"allocations": map[string]any{
					"alloc1": map[string]any{
						"type": "service",
					},
					"alloc2": map[string]any{
						"type": "service",
					},
					"alloc3": map[string]any{
						"type": "service",
					},
				},
			},
			expectError: false,
		},
		{
			// Invalid strategy
			name: "invalid strategy",
			supervisor: map[string]any{
				"strategy":    "InvalidStrategy",
				"allocations": []any{"alloc1"},
			},
			expectError: true,
			errorMsg:    "invalid supervisor strategy: InvalidStrategy",
		},
		{
			// Missing allocation
			name: "missing allocation",
			supervisor: map[string]any{
				"strategy":    "OneForOne",
				"allocations": []any{"nonexistent"},
			},
			root: map[string]any{
				"allocations": map[string]any{},
			},
			expectError: true,
			errorMsg:    "referenced allocation 'nonexistent' not found",
		},
		{
			// Invalid child type
			name: "invalid child type",
			supervisor: map[string]any{
				"strategy": "OneForOne",
				"children": []any{
					"not a map",
				},
			},
			expectError: true,
			errorMsg:    "invalid child supervisor at index 0: must be a map",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.root != nil && tt.root["V1"] == nil {
				tt.root["V1"] = tt.root
			}
			err := ValidateSupervisor(&tt.root, tt.supervisor, "V1.supervisor")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLocation(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		location    any
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid continent only",
			location: map[string]any{
				"continent": "NA",
			},
			expectError: false,
		},
		{
			name: "valid full location",
			location: map[string]any{
				"continent": "EU",
				"country":   "United Kingdom",
				"city":      "London",
				"asn":       uint(12345),
				"isp":       "Example ISP",
			},
			expectError: false,
		},
		{
			name:        "invalid not a map",
			location:    nil,
			expectError: true,
			errorMsg:    "location must be a map",
		},
		{
			name: "missing continent",
			location: map[string]any{
				"country": "United Kingdom",
			},
			expectError: true,
			errorMsg:    "continent is required and cannot be empty",
		},
		{
			name: "empty continent",
			location: map[string]any{
				"continent": "",
			},
			expectError: true,
			errorMsg:    "continent is required and cannot be empty",
		},
		{
			name: "empty country",
			location: map[string]any{
				"continent": "NA",
				"country":   "",
			},
			expectError: true,
			errorMsg:    "country cannot be empty if specified",
		},
		{
			name: "city without country",
			location: map[string]any{
				"continent": "EU",
				"city":      "London",
			},
			expectError: true,
			errorMsg:    "country must be specified when city is provided",
		},
		{
			name: "empty city",
			location: map[string]any{
				"continent": "EU",
				"country":   "United Kingdom",
				"city":      "",
			},
			expectError: true,
			errorMsg:    "city cannot be empty if specified",
		},
		{
			name: "empty isp",
			location: map[string]any{
				"continent": "EU",
				"isp":       "",
			},
			expectError: true,
			errorMsg:    "ISP cannot be empty if specified",
		},
		{
			name: "zero asn",
			location: map[string]any{
				"continent": "EU",
				"asn":       uint(0),
			},
			expectError: true,
			errorMsg:    "ASN must be positive if specified",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateLocation(tt.location, 0)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateLocationConstraints(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		location    map[string]any
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid accept only",
			location: map[string]any{
				"accept": []any{
					map[string]any{
						"continent": "NA",
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid reject only",
			location: map[string]any{
				"reject": []any{
					map[string]any{
						"continent": "EU",
					},
				},
			},
			expectError: false,
		},
		{
			name: "valid both accept and reject",
			location: map[string]any{
				"accept": []any{
					map[string]any{
						"continent": "SA",
					},
				},
				"reject": []any{
					map[string]any{
						"continent": "NA",
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid accept location",
			location: map[string]any{
				"accept": []any{
					map[string]any{}, // missing continent
				},
			},
			expectError: true,
			errorMsg:    "invalid accept location at index 0: continent is required and cannot be empty",
		},
		{
			name: "invalid reject location",
			location: map[string]any{
				"reject": []any{
					map[string]any{}, // missing continent
				},
			},
			expectError: true,
			errorMsg:    "invalid reject location at index 0: continent is required and cannot be empty",
		},
		{
			name: "invalid accept location type",
			location: map[string]any{
				"accept": []any{
					"not a map",
				},
			},
			expectError: true,
			errorMsg:    "invalid accept location at index 0: location must be a map",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validateLocationConstraints(tt.location)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateEdgeConstraints(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name           string
		edgeConstraint any
		root           map[string]any
		expectError    bool
		errorMsg       string
	}{
		{
			name: "valid edge constraint",
			edgeConstraint: map[string]any{
				"S": "node1",
				"T": "node2",
			},
			root: map[string]any{
				"nodes": map[string]any{
					"node1": map[string]any{},
					"node2": map[string]any{},
				},
			},
			expectError: false,
		},
		{
			name: "valid edge constraint with RTT and BW",
			edgeConstraint: map[string]any{
				"S":   "node1",
				"T":   "node2",
				"RTT": uint(100),
				"BW":  uint(1000),
			},
			root: map[string]any{
				"nodes": map[string]any{
					"node1": map[string]any{},
					"node2": map[string]any{},
				},
			},
			expectError: false,
		},
		{
			name:           "invalid not a map",
			edgeConstraint: nil,
			root:           map[string]any{},
			expectError:    true,
			errorMsg:       "invalid edge constraints configuration: <nil>",
		},
		{
			name: "missing source node",
			edgeConstraint: map[string]any{
				"T": "node2",
			},
			root: map[string]any{
				"nodes": map[string]any{
					"node1": map[string]any{},
					"node2": map[string]any{},
				},
			},
			expectError: true,
			errorMsg:    "invalid edge constraints configuration: edges should be a pair of named nodes",
		},
		{
			name: "missing target node",
			edgeConstraint: map[string]any{
				"S": "node1",
			},
			root: map[string]any{
				"nodes": map[string]any{
					"node1": map[string]any{},
					"node2": map[string]any{},
				},
			},
			expectError: true,
			errorMsg:    "invalid edge constraints configuration: edges should be a pair of named nodes",
		},
		{
			name: "empty source node",
			edgeConstraint: map[string]any{
				"S": "",
				"T": "node2",
			},
			root: map[string]any{
				"nodes": map[string]any{
					"node1": map[string]any{},
					"node2": map[string]any{},
				},
			},
			expectError: true,
			errorMsg:    "invalid edge constraints configuration: edges should be a pair of named nodes",
		},
		{
			name: "empty target node",
			edgeConstraint: map[string]any{
				"S": "node1",
				"T": "",
			},
			root: map[string]any{
				"nodes": map[string]any{
					"node1": map[string]any{},
					"node2": map[string]any{},
				},
			},
			expectError: true,
			errorMsg:    "invalid edge constraints configuration: edges should be a pair of named nodes",
		},
		{
			name: "same source and target",
			edgeConstraint: map[string]any{
				"S": "node1",
				"T": "node1",
			},
			root: map[string]any{
				"nodes": map[string]any{
					"node1": map[string]any{},
					"node2": map[string]any{},
				},
			},
			expectError: true,
			errorMsg:    "edge constraint source and target must be different nodes",
		},
		{
			name: "missing nodes in root",
			edgeConstraint: map[string]any{
				"S": "node1",
				"T": "node2",
			},
			root:        map[string]any{},
			expectError: true,
			errorMsg:    "invalid edge constraints configuration: nodes must be defined",
		},
		{
			name: "invalid nodes type in root",
			edgeConstraint: map[string]any{
				"S": "node1",
				"T": "node2",
			},
			root: map[string]any{
				"nodes": []any{}, // should be a map
			},
			expectError: true,
			errorMsg:    "invalid edge constraints configuration: nodes must be a map",
		},
		{
			name: "source node not found",
			edgeConstraint: map[string]any{
				"S": "node1",
				"T": "node2",
			},
			root: map[string]any{
				"nodes": map[string]any{
					"node2": map[string]any{},
				},
			},
			expectError: true,
			errorMsg:    "invalid edge constraints configuration: node 'node1' not found",
		},
		{
			name: "target node not found",
			edgeConstraint: map[string]any{
				"S": "node1",
				"T": "node2",
			},
			root: map[string]any{
				"nodes": map[string]any{
					"node1": map[string]any{},
				},
			},
			expectError: true,
			errorMsg:    "invalid edge constraints configuration: node 'node2' not found",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.root != nil && tt.root["V1"] == nil {
				tt.root["V1"] = tt.root
			}
			err := ValidateEdgeConstraints(&tt.root, tt.edgeConstraint, "V1.edges.[]")
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewEnsembleV1Validator(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name        string
		config      map[string]any
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid complete configuration",
			config: map[string]any{
				"V1": map[string]any{
					"escalation_strategy": "redeploy",
					"allocations": map[string]any{
						"alloc1": map[string]any{
							"type":     "task",
							"executor": "docker",
							"execution": map[string]any{
								"type": "docker",
								"params": map[string]any{
									"image": "ubuntu:latest",
									"env": map[string]string{
										"KEY1": "value1",
									},
								},
							},
							"resources": map[string]any{
								"cpu": map[string]any{
									"cores": 2,
								},
								"ram": map[string]any{
									"size": 4096,
								},
								"disk": map[string]any{
									"size": 10000,
								},
							},
							"failure_recovery": "one_for_one",
						},
						"alloc2": map[string]any{
							"type":     "service",
							"executor": "firecracker",
							"execution": map[string]any{
								"type": "firecracker",
								"params": map[string]any{
									"kernel_image":     "vmlinux",
									"kernel_args":      "console=ttyS0",
									"root_file_system": "rootfs.ext4",
								},
							},
							"resources": map[string]any{
								"cpu": map[string]any{
									"cores": 2,
								},
								"ram": map[string]any{
									"size": 4096,
								},
								"disk": map[string]any{
									"size": 10000,
								},
							},
							"failure_recovery": "one_for_all",
							"depends_on":       []any{"alloc1"},
						},
						"alloc3": map[string]any{
							"type":     "service",
							"executor": "docker",
							"execution": map[string]any{
								"type": "docker",
								"params": map[string]any{
									"image": "nginx:latest",
								},
							},
							"resources": map[string]any{
								"cpu": map[string]any{
									"cores": 1,
								},
								"ram": map[string]any{
									"size": 2048,
								},
								"disk": map[string]any{
									"size": 5000,
								},
							},
							"failure_recovery": "one_for_one",
						},
					},
					"nodes": map[string]any{
						"node1": map[string]any{
							"allocations": []any{"alloc1", "alloc2"},
							"location": map[string]any{
								"accept": []any{
									map[string]any{
										"continent": "EU",
										"country":   "United Kingdom",
										"city":      "London",
									},
								},
								"reject": []any{
									map[string]any{
										"continent": "NA",
									},
								},
							},
							"ports": []any{
								map[string]any{
									"private":    8080,
									"public":     8080,
									"allocation": "alloc1",
								},
							},
							"redundancy":       2,
							"failure_recovery": "restart",
						},
						"node2": map[string]any{
							"allocations": []any{"alloc3"},
							"location": map[string]any{
								"accept": []any{
									map[string]any{
										"continent": "NA",
									},
								},
							},
							"failure_recovery": "restart",
						},
					},
					"edges": []any{
						map[string]any{
							"S":   "node1",
							"T":   "node2",
							"RTT": uint(100),
							"BW":  uint(1000),
						},
					},
					"supervisor": map[string]any{
						"strategy": "OneForOne",
						"allocations": []any{
							"alloc1",
							"alloc2",
							"alloc3",
						},
						"children": []any{
							map[string]any{
								"strategy": "AllForOne",
								"allocations": []any{
									"alloc2",
								},
							},
							map[string]any{
								"strategy": "OneForOne",
								"allocations": []any{
									"alloc3",
								},
							},
						},
					},
				},
			},
			expectError: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			validator := NewEnsembleV1Validator()
			err := validator.Validate(&tt.config)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateHealthCheck(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		data     any
		wantErr  bool
		errorMsg string
	}{
		{
			// Empty health_check reference should pass without validation
			name:     "empty health_check reference",
			data:     map[string]any{},
			wantErr:  true,
			errorMsg: "healthcheck cannot be empty if specified",
		},
		{
			// Invalid health check without exec
			name: "invalid health check wihtout exec",
			data: map[string]any{
				"type": "command",
				"response": map[string]any{
					"type":  "string",
					"value": "hello",
				},
				"interval": "10s",
			},
			wantErr:  true,
			errorMsg: "command type healthcheck must have exec",
		},
		{
			// Valid health check config
			name: "valid health check reference",
			data: map[string]any{
				"type": "command",
				"exec": []string{"echo", "hello"},
				"response": map[string]any{
					"type":  "string",
					"value": "hello",
				},
				"interval": "10s",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateHealthCheck(nil, tt.data, "")
			if tt.wantErr {
				assert.Error(t, err, "expected error: %q", tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSubnet(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		data     any
		wantErr  bool
		errorMsg string
	}{
		{
			// happy case
			name: "subnet.join set to true",
			data: map[string]any{
				"join": true,
			},
			wantErr: false,
		},
		{
			// wrong data type to subnet.join
			name: "subnet.join set to string",
			data: map[string]any{
				"join": "a string value",
			},
			wantErr:  true,
			errorMsg: "subnet.join expects boolean value",
		},
		{
			// subnet must be a map type to include subnet configs
			data:     "",
			wantErr:  true,
			errorMsg: "invalid subnet config",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateSubnet(nil, tt.data, "")
			if tt.wantErr {
				assert.Error(t, err, "expected error: %q", tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateVolume(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		data     any
		wantErr  bool
		errorMsg string
	}{
		{
			name:     "invalid data for volume",
			data:     string("invalid volume data"),
			wantErr:  true,
			errorMsg: "invalid volume configuration",
		},
		{
			name:     "empty volume reference",
			data:     []any{},
			wantErr:  true,
			errorMsg: "volume cannot be empty if specified",
		},
		{
			name: "invalid volume type not a map",
			data: []any{
				12,
			},
			wantErr:  true,
			errorMsg: "invalid volume entry",
		},
		{
			name: "invalid volume type",
			data: []any{
				map[string]any{
					"type": "invalidtype",
				},
			},
			wantErr:  true,
			errorMsg: "unsupported volume type: invalidtype",
		},
		{
			name: "invalid only volume type",
			data: []any{
				map[string]any{
					"type": "local",
				},
			},
			wantErr:  true,
			errorMsg: "local type volume must define a path",
		},
		{
			name: "invalid glusterfs type not defining servers",
			data: []any{
				map[string]any{
					"type": "glusterfs",
				},
			},
			wantErr:  true,
			errorMsg: "glusterfs type volume must define one or more servers",
		},
		{
			name: "invalid glusterfs type defining empty servers",
			data: []any{
				map[string]any{
					"type":    "glusterfs",
					"servers": []string{},
				},
			},
			wantErr:  true,
			errorMsg: "glusterfs type volume must define at least one server",
		},
		{
			name: "invalid glusterfs type defining one invalid server",
			data: []any{
				map[string]any{
					"type":    "glusterfs",
					"servers": []string{"server1", "", "server3"},
				},
			},
			wantErr:  true,
			errorMsg: "glusterfs volume server at index 1 must be a non-empty string",
		},
		{
			name: "invalid glusterfs spec missing mount_destination",
			data: []any{
				map[string]any{
					"type":    "glusterfs",
					"servers": []string{"server1", "server2", "server3"},
				},
			},
			wantErr:  true,
			errorMsg: "volume must define a mount_destination",
		},
		{
			name: "invalid local spec not defining a src",
			data: []any{
				map[string]any{
					"type":              "local",
					"mount_destination": "/mnt/local",
				},
			},
			wantErr:  true,
			errorMsg: "local type volume must define a path",
		},
		{
			name: "invalid local spec with invalid relative path as src",
			data: []any{
				map[string]any{
					"type":              "local",
					"src":               "relative/path/volume1",
					"mount_destination": "/mnt/local",
				},
			},
			wantErr:  true,
			errorMsg: "local volume src must be an absolute path",
		},
		{
			name: "invalid local spec with invalid characters for src",
			data: []any{
				map[string]any{
					"type":              "local",
					"src":               "abcd$%^&volume1",
					"mount_destination": "/mnt/local",
				},
			},
			wantErr:  true,
			errorMsg: "local volume src contains invalid characters",
		},
		{
			name: "invalid - no type specified",
			data: []any{
				map[string]any{
					"src":               "/data/volume1",
					"mount_destination": "/mnt/local",
				},
			},
			wantErr:  true,
			errorMsg: "volume must have a type",
		},
		{
			name: "valid local spec",
			data: []any{
				map[string]any{
					"type":              "local",
					"src":               "/data/volume1",
					"mount_destination": "/mnt/local",
				},
			},
			wantErr: false,
		},
		{
			name: "valid local named volume spec",
			data: []any{
				map[string]any{
					"type":              "local",
					"src":               "the_named-volume_123",
					"mount_destination": "/mnt/local",
				},
			},
			wantErr: false,
		},
		{
			name: "valid glusterfs spec",
			data: []any{
				map[string]any{
					"type":              "glusterfs",
					"servers":           []string{"server1", "server2", "server3"},
					"mount_destination": "/mnt/glusterfs",
				},
			},
			wantErr: false,
		},
		{
			name: "valid glusterfs spec",
			data: []any{
				map[string]any{
					"type":               "glusterfs",
					"servers":            []string{"server1", "server2", "server3"},
					"mount_destination":  "/mnt/glusterfs",
					"client_private_key": "/path/to/private.key",
					"client_pem":         "/path/to/client.pem",
					"client_ca":          "/path/to/ca.cert",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateVolume(nil, tt.data, "")
			if tt.wantErr {
				assert.Error(t, err, "expected error: %q", tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
