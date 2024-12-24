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
	"strings"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/transform"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/utils"
	"gitlab.com/nunet/device-management-service/utils/convert"
)

func NewEnsemblev1Transformer() transform.Transformer {
	return transform.NewTransformer(
		[]map[tree.Path]transform.TransformerFunc{
			{
				"allocations.*.volumes": transform.MapToNamedSliceTransformer("volume"),
				"volumes":               transform.MapToNamedSliceTransformer("volume"),
				"resources":             transform.MapToNamedSliceTransformer("resource"),
			},
			{
				"allocations.*.volumes.[]": TransformVolume,
				"allocations.*.resources":  TransformResources,
				"scripts.*":                TransformStringToBytes,
			},
			{
				"allocations.*.execution":         transform.SpecConfigTransformer("execution"),
				"allocations.*.volumes.[].remote": transform.SpecConfigTransformer("remote volume"),
				"edge_constraints.[]":             TransformEdgeConstraint,
			},
			{
				"": TransformSpec,
			},
		},
	)
}

func TransformStringToBytes(_ *map[string]interface{}, data any, _ tree.Path) (any, error) {
	str, ok := data.(string)
	if !ok {
		return nil, fmt.Errorf("invalid string data: %v", data)
	}

	data = []byte(str)

	return data, nil
}

// TransformSpec transforms the spec configuration and wraps it in a "V1" key.
func TransformSpec(_ *map[string]interface{}, data any, _ tree.Path) (any, error) {
	spec, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid spec configuration: %v", data)
	}

	// set dns_name of allocations to the allocation name if not set
	if allocations, ok := spec["allocations"]; ok {
		for allocName, alloc := range allocations.(map[string]any) {
			allocation, ok := alloc.(map[string]any)
			if ok {
				if allocation["dns_name"] == nil {
					allocation["dns_name"] = allocName
				}
			}
		}
	}

	// move edge_constraints to edges
	if edgeConstraints, ok := spec["edge_constraints"]; ok {
		spec["edges"] = edgeConstraints
		delete(spec, "edge_constraints")
	}

	return map[string]any{"V1": spec}, nil
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

		c, err := utils.GetConfigAtPath(*root, parent.Next("volumes"))
		if err != nil {
			return config, nil
		}

		volumes, _ := utils.ToAnySlice(c)
		for _, v := range volumes {
			if volume, ok := v.(map[string]any); ok && volume["name"] == config["name"] {
				// Merge the configurations
				for k, v := range config {
					volume[k] = v
				}
				config = volume
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

		c, err := utils.GetConfigAtPath(*root, parent.Next("resources"))
		if err != nil {
			return config, nil
		}

		resources, _ := utils.ToAnySlice(c)
		for _, v := range resources {
			if rcs, ok := v.(map[string]any); ok && rcs["name"] == config["name"] {
				// Merge the configurations
				for k, v := range config {
					rcs[k] = v
				}
				config = rcs
			}
		}
	}

	// Convert resource values to their respective units:
	// - CPU and RAM clock_speed: defaults to GHz, accepts SI units (Hz, MHz, GHz)
	// - RAM and disk size: defaults to GiB, accepts binary (KiB, MiB, GiB) and decimal (KB, MB, GB) units
	// - GPU VRAM: defaults to GiB, accepts binary and decimal units
	if cpu, ok := config["cpu"].(map[string]any); ok {
		if speed, ok := cpu["clock_speed"]; ok {
			val, err := convert.ParseSIWithDefaultUnit(speed, "GHz")
			if err != nil {
				return nil, fmt.Errorf("invalid cpu clock_speed: %v", err)
			}
			cpu["clock_speed"] = val
		}
	}

	if ram, ok := config["ram"].(map[string]any); ok {
		if speed, ok := ram["clock_speed"]; ok {
			val, err := convert.ParseSIWithDefaultUnit(speed, "GHz")
			if err != nil {
				return nil, fmt.Errorf("invalid ram clock_speed: %v", err)
			}
			ram["clock_speed"] = val
		}
		if size, ok := ram["size"]; ok {
			val, err := convert.ParseBytesWithDefaultUnit(size, "GiB")
			if err != nil {
				return nil, fmt.Errorf("invalid ram size: %v", err)
			}
			ram["size"] = val
		}
	}

	if disk, ok := config["disk"].(map[string]any); ok {
		if size, ok := disk["size"]; ok {
			val, err := convert.ParseBytesWithDefaultUnit(size, "GiB")
			if err != nil {
				return nil, fmt.Errorf("invalid disk size: %v", err)
			}
			disk["size"] = val
		}
	}

	if gpus, ok := config["gpu"].([]any); ok {
		for i, g := range gpus {
			if gpu, ok := g.(map[string]any); ok {
				if vram, ok := gpu["vram"]; ok {
					val, err := convert.ParseBytesWithDefaultUnit(vram, "GiB")
					if err != nil {
						return nil, fmt.Errorf("invalid gpu[%d] vram: %v", i, err)
					}
					gpu["vram"] = val
				}
			}
		}
	}

	return config, nil
}
