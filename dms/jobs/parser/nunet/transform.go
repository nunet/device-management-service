package nunet

import (
	"fmt"
	"strconv"
	"strings"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/transform"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

// NewNuNetValidator creates a new validator for the NuNet configuration.
func NewNuNetTransformer() transform.Transformer {
	return transform.NewTransformer(
		[]map[tree.Path]transform.TransformerFunc{
			{
				"jobs":             TransformJobs,
				"jobs.**.children": TransformJobs,
				"jobs.**.volumes":  TransformVolumes,
				"jobs.**.networks": TransformNetworks,
			},
			{
				"jobs.**.volumes.[]":        TransformVolume,
				"jobs.**.networks.[]":       TransformNetwork,
				"jobs.**.libraries.[]":      TransformLibrary,
			},
			{
				"jobs.**.execution":         TransformExecution,
				"jobs.**.volumes.[].remote": TransformVolumeRemote,
			},
		},
	)
}

// TransformJobs transforms the jobs map to a slice and assigns the keys to the "name" field.
func TransformJobs(root *map[string]interface{}, data any, path tree.Path) (any, error) {
	if data == nil {
		return nil, nil
	}
	jobs, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid jobs configuration: %v", data)
	}
	return transform.MapToSlice(jobs)
}

// TransformVolumes transforms the volumes map to a slice and assigns the keys to the "name" field.
func TransformVolumes(root *map[string]interface{}, data any, path tree.Path) (any, error) {
	if data == nil {
		return nil, nil
	}
	volumes, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid volumes configuration: %v", data)
	}
	return transform.MapToSlice(volumes)
}

// TransformNetworks transforms the networks map to a slice and assigns the keys to the "name" field.
func TransformNetworks(root *map[string]interface{}, data any, path tree.Path) (any, error) {
	if data == nil {
		return nil, nil
	}
	networks, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid networks configuration: %v", data)
	}
	return transform.MapToSlice(networks)
}

// TransformExecution transforms the engine configuration from flat map to SpecConfig format.
func TransformExecution(root *map[string]interface{}, data any, path tree.Path) (any, error) {
	if data == nil {
		return nil, nil
	}
	engine, ok := data.(map[string]any)
	result := map[string]any{}
	if !ok {
		return nil, fmt.Errorf("invalid engine configuration: %v", data)
	}
	params := map[string]any{}
	for k, v := range engine {
		if k != "type" {
			params[k] = v
		}
	}
	result["type"] = engine["type"]
	result["params"] = params

	return result, nil
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

	// Collect all potential parent paths where the volume could be defined.
	parentPaths := []tree.Path{}
	pathParts := path.Parts()
	for i, part := range pathParts {
		if part == "children" {
			parentPaths = append(parentPaths, tree.NewPath(pathParts[:i]...))
		}
	}

	// Merge the volume configuration with the parent configurations.
	for _, parent := range parentPaths {
		// Check if the volume exists in the parent
		c, err := transform.GetConfigAtPath(*root, parent.Next("volumes"))
		if err != nil {
			fmt.Println("error: ", err)
			continue
		}

		volumes, _ := transform.ToAnySlice(c)
		for _, v := range volumes {
			if volume, ok := v.(map[string]any); ok && volume["name"] == config["name"] {
				// Merge the configurations
				for k, v := range volume {
					config[k] = v
				}
			}
		}
	}
	return config, nil
}

func TransformVolumeRemote(root *map[string]interface{}, data any, path tree.Path) (any, error) {
	if data == nil {
		return nil, nil
	}
	config, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid volume configuration: %v", data)
	}
	remoteConfig := map[string]any{}
	remoteConfig["type"] = config["type"]
	if params, ok := config["params"]; ok {
		remoteConfig["params"] = params.(map[string]any)
		return remoteConfig, nil
	}
	params := map[string]any{}
	for k, v := range config {
		if k != "type" {
			params[k] = v
		}
	}
	remoteConfig["params"] = params
	return remoteConfig, nil
}


// TransformNetwork transforms the network configuration
func TransformNetwork(root *map[string]interface{}, data any, path tree.Path) (any, error) {
	if data == nil {
		return nil, nil
	}
	config, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid network configuration: %v", data)
	}
	ports, _ := transform.ToAnySlice(config["ports"])
	portMap := []map[string]any{}
	for _, port := range ports {
		protocol, host, container := "tcp", 0, 0;
		switch v := port.(type) {
		case string:
			parts := strings.Split(v, ":")
			if len(parts) <= 2 {
				host, _ = strconv.Atoi(parts[0])
				container, _ = strconv.Atoi(parts[len(parts)-1])
			} else if len(parts) == 3 {
				protocol = parts[0]
				host, _ = strconv.Atoi(parts[1])
				container, _ = strconv.Atoi(parts[len(parts)-1])
			}
		case int:
			host = v
			container = v
		case map[string]any:
			switch h := v["host_port"].(type){
			case int:
				host = h
			case string:
				host, _ = strconv.Atoi(h)
			}

			switch c := v["container_port"].(type){
			case int:
				container = c
			case string:
				container, _ = strconv.Atoi(c)
			}

			if p, ok := v["protocol"].(string); ok {
				protocol = p
			}
		}
		portMap = append(portMap, map[string]any{
			"protocol":      protocol,
			"host_port":      host,
			"container_port": container,
		})
	}

	config["port_map"] = portMap
	delete(config, "ports")

	return config, nil
}

// TransformLibrary tansforms the library configuration to a map.
// The library configuration can be a string in the format "name:version" or a map.
func TransformLibrary(root *map[string]interface{}, data any, path tree.Path) (any, error) {
	if data == nil {
		return nil, nil
	}
	switch v := data.(type) {
	case string:
		parts := strings.Split(v, ":")
		if len(parts) == 1 {
			parts = append(parts, "")
		}
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid library configuration: %v", data)
		}
		return map[string]any{
			"name":    parts[0],
			"version": parts[1],
		}, nil
	case map[string]any:
		return v, nil
	default:
		return nil, fmt.Errorf("invalid library configuration: %v", data)
	}
}
