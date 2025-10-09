// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package ensemblev1

import (
	"fmt"
	"maps"
	"reflect"
	"strings"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/transform"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/utils"
)

func NewEnsemblev1Decoder() transform.Transformer {
	return transform.NewTransformer(
		[]map[tree.Path]transform.TransformerFunc{
			// Transform key value pairs to slices with name as key
			{
				"allocations.*.volumes": transform.MapToNamedSliceTransformer("volume"),
				"volumes":               transform.MapToNamedSliceTransformer("volume"),
				"resources":             transform.MapToNamedSliceTransformer("resource"),
			},
			// Transform configs
			{
				"allocations.*.volumes.[]":            TransformVolume,
				"allocations.*.resources":             TransformResources,
				"scripts.*":                           TransformStringToBytes,
				"allocations.*.execution.environment": TransformEnvironment,
			},
			// Transform numeric values
			{
				"allocations.*.resources.cpu.clock_speed": transform.ParseWithDefaultUnit("cpu clock_speed", "GHz"),
				"allocations.*.resources.ram.clock_speed": transform.ParseWithDefaultUnit("ram clock_speed", "GHz"),
				"allocations.*.resources.ram.size":        transform.ParseBytesWithDefaultUnit("ram size", "GiB"),
				"allocations.*.resources.disk.size":       transform.ParseBytesWithDefaultUnit("disk size", "GiB"),
				"allocations.*.resources.gpu.[].vram":     transform.ParseBytesWithDefaultUnit("gpu vram", "GiB"),
				"allocations.*.healthcheck.interval":      transform.ParseDuration("healthcheck duration"),
			},
			{
				"allocations.*.execution":         transform.ToSpecConfigTransformer("execution"),
				"allocations.*.volumes.[].remote": transform.ToSpecConfigTransformer("remote volume"),
				"edge_constraints.[]":             TransformEdgeConstraint,
			},
			{
				"": TransformSpec,
			},
		},
	)
}

func TransformStringToBytes(_ *map[string]interface{}, data any, _ tree.Path) (any, error) {
	switch v := data.(type) {
	case []byte:
		return v, nil
	case string:
		return []byte(v), nil
	default:
		return nil, fmt.Errorf("invalid data type: %T,", data)
	}
}

// TransformSpec transforms the spec configuration and wraps it in a "V1" key.
func TransformSpec(_ *map[string]interface{}, data any, _ tree.Path) (any, error) {
	spec, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid spec configuration: %v", data)
	}

	// set default values for allocations
	if allocations, ok := spec["allocations"]; ok {
		for allocName, alloc := range allocations.(map[string]any) {
			if allocation, ok := alloc.(map[string]any); ok {
				// set dns_name of allocations to the allocation name if not set
				if allocation["dns_name"] == nil {
					allocation["dns_name"] = allocName
				}
				// set failure_recovery to "stay_down" if not set
				if allocation["failure_recovery"] == nil {
					allocation["failure_recovery"] = defaultAllocationFailureStrategy
				}
			}
		}
	}

	// set default values for nodes
	if nodes, ok := spec["nodes"]; ok {
		for _, node := range nodes.(map[string]any) {
			if nodeConfig, ok := node.(map[string]any); ok {
				// set failure_recovery to "stay_down" if not set
				if nodeConfig["failure_recovery"] == nil {
					nodeConfig["failure_recovery"] = defaultNodeFailureStrategy
				}
				// set redundancy to 0 if not set
				if nodeConfig["redundancy"] == nil {
					nodeConfig["redundancy"] = 0
				}
			}
		}
	}

	// move edge_constraints to edges
	if edgeConstraints, ok := spec["edge_constraints"]; ok {
		spec["edges"] = edgeConstraints
		delete(spec, "edge_constraints")
	}

	return map[string]any{"v1": spec}, nil
}

// TransformEdgeConstraint maps the edges parameter to Source and Target (S and T) properties.
func TransformEdgeConstraint(_ *map[string]interface{}, data any, _ tree.Path) (any, error) {
	edgeConstraints, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid edge constraints: %v", data)
	}

	// Map the edges parameter to Source and Target (S and T) properties
	if edges, ok := edgeConstraints["edges"]; ok {
		// Assert edges is a list of two strings
		edgesList, ok := edges.([]any)
		if !ok || len(edgesList) != 2 {
			return nil, fmt.Errorf("invalid edges parameter: %v", edges)
		}
		edgeConstraints["S"] = edgesList[0]
		edgeConstraints["T"] = edgesList[1]
		delete(edgeConstraints, "edges")
	}

	return edgeConstraints, nil
}

// TransformVloume transforms the volume configuration and handles inheritance.
// The volume configuration can be a string in the format "name:mountpoint" or a map.
// If the volume is defined in the parent volumes, the configurations are merged.
func TransformVolume(root *map[string]interface{}, data any, path tree.Path) (any, error) {
	var config map[string]any

	// If the data is a string, split it into name and mountpoint.
	switch v := data.(type) {
	case string:
		mapping := strings.Split(v, ":")
		if len(mapping) != 2 {
			return nil, fmt.Errorf("invalid volume configuration: %v", data)
		}
		config = map[string]any{
			"name":       mapping[0],
			"mountpoint": mapping[1],
		}
	case map[string]any:
		config = v
	default:
		return nil, fmt.Errorf("invalid volume configuration: %v", data)
	}

	if path.Matches("allocations.*.volumes.[]") {
		// Handle volume inheritance
		parent := tree.NewPath("")

		if c, err := utils.GetConfigAtPath(*root, parent.Next("volumes")); err == nil {
			for _, v := range c.([]any) {
				if volume, ok := v.(map[string]any); ok && volume["name"] == config["name"] {
					// Merge the configurations
					maps.Copy(volume, config)
					config = volume
				}
			}
		}
	}
	return config, nil
}

// TransformResources transforms the resources configuration and handles inheritance.
// The resources configuration can be a string reference "reference" or a map.
// If the resources is defined in the parent resources, the configurations are merged.
func TransformResources(root *map[string]interface{}, data any, path tree.Path) (any, error) {
	var config map[string]any

	// If the data is a string, transform it to a map with the name as the reference
	switch v := data.(type) {
	case string:
		config = map[string]any{
			"name": v,
		}
	case map[string]any:
		config = v
	default:
		return nil, fmt.Errorf("invalid resources configuration: %v", data)
	}

	if path.Matches("allocations.*.resources") {
		// Handle volume inheritance
		parent := tree.NewPath("")

		if c, err := utils.GetConfigAtPath(*root, parent.Next("resources")); err == nil {
			for _, v := range c.([]any) {
				if rcs, ok := v.(map[string]any); ok && rcs["name"] == config["name"] {
					// Merge the configurations
					maps.Copy(rcs, config)
					config = rcs
				}
			}
		}
	}

	return config, nil
}

func TransformEnvironment(_ *map[string]interface{}, data any, _ tree.Path) (any, error) {
	switch v := data.(type) {
	case map[string]any, map[string]string:
		envs := make([]string, 0)
		for k, v := range reflect.ValueOf(v).Seq2() {
			envs = append(envs, fmt.Sprintf("%s=%s", k, v))
		}
		return envs, nil
	case []string, []any:
		return v, nil
	default:
		return nil, fmt.Errorf("invalid environment configuration: %T", data)
	}
}
