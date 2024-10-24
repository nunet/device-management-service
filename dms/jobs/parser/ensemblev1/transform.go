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

		c, err := transform.GetConfigAtPath(*root, parent.Next("volumes"))
		if err != nil {
			return config, nil
		}

		volumes, _ := transform.ToAnySlice(c)
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

		c, err := transform.GetConfigAtPath(*root, parent.Next("resources"))
		if err != nil {
			return config, nil
		}

		resources, _ := transform.ToAnySlice(c)
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
	return config, nil
}
