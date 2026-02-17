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
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"time"

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/utils"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/validate"
	"gitlab.com/nunet/device-management-service/types"
	cutils "gitlab.com/nunet/device-management-service/utils/convert"
	vutils "gitlab.com/nunet/device-management-service/utils/validate"
)

const (
	defaultAllocationFailureStrategy = "stay_down"
	defaultNodeFailureStrategy       = "stay_down"
)

var (
	validEscalationStrategies        = [...]string{"redeploy", "teardown"}
	validAllocationFailureStrategies = [...]string{"stay_down", "one_for_one", "one_for_all", "rest_for_one"}
	validNodeFailureStrategies       = [...]string{"stay_down", "restart", "redeploy"}
)

// NewEnsembleV1Validator creates a new validator for the NuNet configuration.
func NewEnsembleV1Validator() validate.Validator {
	return validate.NewValidator(
		map[tree.Path]validate.ValidatorFunc{
			"V1":                           ValidateSpec,
			"V1.subnet":                    ValidateSubnet,
			"V1.allocations.*":             ValidateAllocation,
			"V1.edges.[]":                  ValidateEdgeConstraints,
			"V1.nodes.*":                   ValidateNode,
			"V1.supervisor":                ValidateSupervisor,
			"V1.supervisor.children.[]":    ValidateSupervisor,
			"V1.allocations.*.resources":   ValidateResources,
			"V1.allocations.*.execution":   ValidateExecution,
			"V1.allocations.*.healthcheck": ValidateHealthCheck,
			"V1.allocations.*.volume":      ValidateVolume,
			"V1.contracts.*":               ValidateContract,
		},
	)
}

// ValidateSpec checks the root configuration for consistency.
func ValidateSpec(_ *map[string]any, data any, _ tree.Path) error {
	spec, dataOk := data.(map[string]any)
	if !dataOk {
		return fmt.Errorf("invalid spec configuration: %v", data)
	}

	// validate escalation strategy if present
	if es, ok := spec["escalation_strategy"].(string); ok {
		if !slices.Contains(validEscalationStrategies[:], es) {
			return fmt.Errorf("invalid escalation_strategy %q: must be one of %q", es, validEscalationStrategies)
		}
	}

	// Check if the allocations map is present and not empty
	allocs, ok := spec["allocations"].(map[string]any)
	if !ok || len(allocs) == 0 {
		return fmt.Errorf("at least one allocation must be defined")
	}

	allocationNames := make(map[string]string)
	dnsNames := make(map[string]string)
	for allocName, allocConfigRaw := range allocs {
		// All allocation names must be fully qualified domain names
		if !vutils.IsDNSNameValid(allocName) {
			return fmt.Errorf("invalid allocation name, must be a valid hostname: %s", allocName)
		}

		// Check for duplicate allocation names (case-insensitive)
		lowerName := strings.ToLower(allocName)
		if originalName, exists := allocationNames[lowerName]; exists {
			return fmt.Errorf("duplicate allocation names found: '%s' and '%s'", originalName, allocName)
		}
		allocationNames[lowerName] = allocName

		// Check for duplicate dns_name values
		allocConfig, ok := allocConfigRaw.(map[string]any)
		if !ok {
			continue
		}
		dnsName, ok := allocConfig["dns_name"].(string)
		if !ok || dnsName == "" {
			continue // skip if dns_name is not set or not a string
		}
		if existingAlloc, exists := dnsNames[dnsName]; exists {
			return fmt.Errorf("duplicate dns_name found: '%s' used in allocations '%s' and '%s'", dnsName, existingAlloc, allocName)
		}
		dnsNames[dnsName] = allocName
	}

	// check for cyclic dependencies
	graph := utils.CreateAdjencyList(allocs, tree.NewPath("depends_on"))
	if utils.DetectCycles(graph) {
		return fmt.Errorf("cyclic dependencies detected in allocations")
	}

	// Check if nodes are defined when edge_constraints are present
	if edges, ok := spec["edges"].([]any); ok && len(edges) > 0 {
		if spec["nodes"] == nil {
			return fmt.Errorf("nodes must be defined when edge_constraints are present")
		}
	}

	// Check that no allocation is present in multiple nodes and that dependencies are in the same node
	if nodes, ok := spec["nodes"].(map[string]any); ok && len(nodes) > 0 {
		// Create a map to track which node each allocation belongs to
		allocToNode := make(map[string]string)

		// Build the allocation-to-node map
		for nodeName, nodeConfig := range nodes {
			nodeMap, ok := nodeConfig.(map[string]any)
			if !ok {
				continue
			}

			nodeAllocs, ok := nodeMap["allocations"].([]any)
			if !ok {
				continue
			}

			for _, alloc := range nodeAllocs {
				allocName, ok := alloc.(string)
				if !ok {
					continue
				}

				// Check if this allocation is already assigned to another node
				if existingNode, exists := allocToNode[allocName]; exists {
					return fmt.Errorf("allocation '%s' is assigned to multiple nodes ('%s' and '%s'): an allocation can only be assigned to one node", allocName, existingNode, nodeName)
				}

				// Record this allocation's node
				allocToNode[allocName] = nodeName

				// Check dependencies immediately
				allocConfig, ok := allocs[allocName].(map[string]any)
				if !ok {
					continue // Should never happen
				}

				dependencies, ok := allocConfig["depends_on"].([]any)
				if !ok {
					continue
				}

				for _, dep := range dependencies {
					depName, ok := dep.(string)
					if !ok {
						continue
					}

					// Check if the dependency is in the same node
					if !slices.Contains(nodeAllocs, dep) {
						return fmt.Errorf("allocation '%s' depends on '%s', but '%s' is not in the same node: dependent allocations must be in the same node", allocName, depName, depName)
					}
				}
			}
		}
	}

	return nil
}

// ValidateAllocation checks the allocation configuration.
func ValidateAllocation(root *map[string]any, data any, path tree.Path) error {
	allocation, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid allocation configuration: %v", data)
	}

	// Check if the allocation has an execution
	if allocation["execution"] == nil {
		return fmt.Errorf("allocation must have an execution")
	}

	// Check if the allocation has resources
	if allocation["resources"] == nil {
		return fmt.Errorf("allocation must have resources")
	}

	// Validate executor type matches execution type
	executor, ok := allocation["executor"].(string)
	if !ok || executor == "" {
		return fmt.Errorf("allocation must have an executor")
	}

	execution, ok := allocation["execution"].(map[string]any)
	if !ok {
		return fmt.Errorf("invalid execution configuration")
	}

	execType, ok := execution["type"].(string)
	if !ok || execType == "" {
		return fmt.Errorf("execution must have a type")
	}

	if executor != execType {
		return fmt.Errorf("allocation executor type (%s) must match execution type (%s)", executor, execType)
	}

	allocType, ok := allocation["type"].(string)
	if !ok || allocType == "" {
		return fmt.Errorf("allocation must have a type (service or task)")
	}

	// Validate DNS name if present
	if dnsName, ok := allocation["dns_name"].(string); ok {
		if dnsName == "" {
			return fmt.Errorf("dns_name cannot be empty if specified")
		}
		// Add basic DNS name format validation
		if !vutils.IsDNSNameValid(dnsName) {
			return fmt.Errorf("invalid dns_name format: %s", dnsName)
		}
	}

	// validate failure_recovery
	if failureRecovery, ok := allocation["failure_recovery"].(string); ok {
		if !slices.Contains(validAllocationFailureStrategies[:], failureRecovery) {
			return fmt.Errorf("invalid failure_recovery %q: must be one of %q", failureRecovery, validAllocationFailureStrategies)
		}
	} else {
		return fmt.Errorf("failure_recovery must be specified for allocation")
	}

	// validate depends_on if present
	if dependsOn, ok := allocation["depends_on"]; ok {
		dependsOn, ok := dependsOn.([]any)
		if !ok {
			return fmt.Errorf("depends_on must be a list of allocation names")
		}
		var allocs map[string]any
		if cfg, err := utils.GetConfigAtPath(*root, "V1.allocations"); err == nil {
			allocs, _ = cfg.(map[string]any)
		}
		for _, v := range dependsOn {
			dep, ok := v.(string)
			if !ok {
				return fmt.Errorf("depends_on must be a list of allocation names")
			}
			if dep != "" && path.Last() == dep {
				return fmt.Errorf("depends_on must not refer to itself")
			}
			if alloc, exists := allocs[dep]; !exists || alloc == nil {
				return fmt.Errorf("depends_on allocation '%s' not found", dependsOn)
			}
		}
	}

	// Validate keys if specified
	if keys, ok := allocation["keys"].([]any); ok && len(keys) > 0 {
		for i, keyObj := range keys {
			keyMap, ok := keyObj.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid allocation key spec at index %d: must be a map", i)
			}

			keyType, ok := keyMap["type"].(string)
			if !ok || keyType == "" {
				return fmt.Errorf("allocation key spec at index %d must have a type", i)
			}

			if !types.KeySSH.Equal(keyType) && !types.KeyGPG.Equal(keyType) {
				return fmt.Errorf("key at index %d has invalid type: %s (must be 'ssh' or 'gpg')", i, keyType)
			}

			keyFile, ok := keyMap["file"].(string)
			if !ok || keyFile == "" {
				return fmt.Errorf("allocation key at index %d is empty", i)
			}

			// destination not required for ssh keys. However 'user' in execution will be
			// used if defined. When a user isn't defined, we default to root.
			keyDest, ok := keyMap["dest"].(string)
			if (!ok || keyDest == "") && !types.KeySSH.Equal(keyType) {
				return fmt.Errorf("allocation key at index %d is missing a destination", i)
			}
		}
	}

	// Validate provision scripts if specified
	if provision, ok := allocation["provision"].([]any); ok && len(provision) > 0 {
		rootScripts, err := utils.GetConfigAtPath(*root, tree.NewPath("V1.scripts"))
		if err != nil || rootScripts == nil {
			return fmt.Errorf("scripts must be defined when provision is defined")
		}
		rootScriptsMap, ok := rootScripts.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid scripts configuration")
		}
		for _, script := range provision {
			scriptStr, ok := script.(string)
			if !ok {
				return fmt.Errorf("invalid script reference: %v", script)
			}
			if _, exists := rootScriptsMap[scriptStr]; !exists {
				return fmt.Errorf("referenced script '%s' not found", scriptStr)
			}
		}
	}

	return nil
}

// ValidateResources validates the resource configuration
func ValidateResources(_ *map[string]any, data any, _ tree.Path) error {
	resources, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid resources configuration: %v", data)
	}

	// Validate CPU (required)
	cpu, ok := resources["cpu"].(map[string]any)
	if !ok {
		return fmt.Errorf("resources must have cpu configuration")
	}

	// Handle cores as any number type and convert to positive integer
	cores, ok := cpu["cores"]
	if !ok {
		return fmt.Errorf("cpu must have cores value")
	}

	coresFloat, err := cutils.ToPositiveFloat64(cores, "cpu cores")
	if err != nil {
		return err
	}
	cpu["cores"] = coresFloat

	// Optional CPU fields
	if arch, ok := cpu["architecture"].(string); ok && arch == "" {
		return fmt.Errorf("cpu architecture cannot be empty if specified")
	}

	if freq, ok := cpu["clock_speed"]; ok {
		if _, err := cutils.ToPositiveFloat64(freq, "cpu clock_speed"); err != nil {
			return err
		}
	}

	// Validate RAM (required)
	ram, ok := resources["ram"].(map[string]any)
	if !ok {
		return fmt.Errorf("resources must have ram configuration")
	}

	// Validate RAM size
	size, ok := ram["size"]
	if !ok {
		return fmt.Errorf("ram must have size value")
	}

	sizeFloat, err := cutils.ToPositiveFloat64(size, "ram size")
	if err != nil {
		return err
	}
	ram["size"] = sizeFloat

	// Optional RAM speed
	if speed, ok := ram["clock_speed"]; ok {
		speedFloat, err := cutils.ToPositiveFloat64(speed, "ram clock_speed")
		if err != nil {
			return err
		}
		ram["clock_speed"] = uint64(speedFloat)
	}

	// Validate disk (required)
	disk, ok := resources["disk"].(map[string]any)
	if !ok {
		return fmt.Errorf("resources must have disk configuration")
	}

	// Validate disk size
	diskSize, ok := disk["size"]
	if !ok {
		return fmt.Errorf("disk must have size value")
	}

	diskSizeFloat, err := cutils.ToPositiveFloat64(diskSize, "disk size")
	if err != nil {
		return err
	}
	disk["size"] = diskSizeFloat

	// Optional disk type
	if diskType, ok := disk["type"].(string); ok && diskType == "" {
		return fmt.Errorf("disk type cannot be empty if specified")
	}

	// Optional GPUs
	if gpusRaw, ok := resources["gpus"]; ok {
		gpus, ok := gpusRaw.([]any)
		if !ok {
			return fmt.Errorf("gpus must be an array")
		}

		for i, gpuRaw := range gpus {
			gpu, ok := gpuRaw.(map[string]any)
			if !ok {
				return fmt.Errorf("gpu at index %d is not a valid configuration", i)
			}

			// Validate GPU memory if specified
			if memory, ok := gpu["memory"]; ok {
				memoryFloat, err := cutils.ToPositiveFloat64(memory, fmt.Sprintf("gpu memory at index %d", i))
				if err != nil {
					return err
				}
				gpu["memory"] = memoryFloat
			}

			// Validate vendor if specified
			if vendor, ok := gpu["vendor"].(string); ok && vendor == "" {
				return fmt.Errorf("gpu vendor cannot be empty if specified at index %d", i)
			}

			// Validate model if specified
			if model, ok := gpu["model"].(string); ok && model == "" {
				return fmt.Errorf("gpu model cannot be empty if specified at index %d", i)
			}
		}
	}

	return nil
}

// ValidateNode checks the node configuration.
func ValidateNode(root *map[string]any, data any, _ tree.Path) error {
	node, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid node configuration: %v", data)
	}

	// Validate allocation references
	if allocations, ok := node["allocations"].([]any); ok {
		rootAllocs, err := utils.GetConfigAtPath(*root, tree.NewPath("V1.allocations"))
		if err != nil || rootAllocs == nil {
			return fmt.Errorf("allocations must be defined when node is defined")
		}
		rootAllocsMap, ok := rootAllocs.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid allocations configuration")
		}
		for _, alloc := range allocations {
			allocStr, ok := alloc.(string)
			if !ok {
				return fmt.Errorf("invalid allocation reference: %v", alloc)
			}
			if _, exists := rootAllocsMap[allocStr]; !exists {
				return fmt.Errorf("referenced allocation '%s' not found", allocStr)
			}
		}
	}

	// validate redundancy if present
	if redundancy, ok := node["redundancy"]; ok {
		// check if redundancy is a positive integer
		redundancyInt, ok := redundancy.(int)
		if !ok {
			return fmt.Errorf("redundancy must be a number")
		}
		if redundancyInt < 0 {
			return fmt.Errorf("redundancy must be a positive number")
		}
	}

	// validate failure recovery
	if failureRecovery, ok := node["failure_recovery"].(string); ok {
		if !slices.Contains(validNodeFailureStrategies[:], failureRecovery) {
			return fmt.Errorf("invalid failure_recovery %q: must be one of %q", failureRecovery, validNodeFailureStrategies)
		}
	} else {
		return fmt.Errorf("failure_recovery must be specified for node")
	}

	// Validate ports if specified
	if ports, ok := node["ports"].([]any); ok {
		for _, port := range ports {
			portMap, ok := port.(map[string]any)
			if !ok {
				return fmt.Errorf("invalid port configuration: %v", port)
			}

			// Validate private port
			if private, ok := portMap["private"].(int); !ok || private <= 0 {
				return fmt.Errorf("port must have a valid private port number")
			}

			// Require public port when private port is specified
			if _, ok := portMap["public"].(int); !ok {
				return fmt.Errorf("public port must be specified when private port is defined")
			}

			// Validate public port
			if public, ok := portMap["public"].(int); ok && (public != 0 && public < 1025 || public > 65535) {
				return fmt.Errorf("port must have a valid public port number between 1025 and 65535 if specified")
			}

			// Validate allocation reference
			if alloc, ok := portMap["allocation"].(string); ok {
				rootAllocs, err := utils.GetConfigAtPath(*root, tree.NewPath("V1.allocations"))
				if err != nil || rootAllocs == nil {
					return fmt.Errorf("allocations must be defined when referenced in port")
				}
				rootAllocsMap, ok := rootAllocs.(map[string]any)
				if !ok {
					return fmt.Errorf("invalid allocations configuration")
				}
				if _, exists := rootAllocsMap[alloc]; !exists {
					return fmt.Errorf("referenced allocation '%s' not found in port configuration", alloc)
				}
			}
		}
	}

	// Validate location constraints if specified
	if location, ok := node["location"].(map[string]any); ok {
		if err := validateLocationConstraints(location); err != nil {
			return fmt.Errorf("invalid location constraints: %v", err)
		}
	}

	// Validate peer if specified
	if peer, ok := node["peer"].(string); ok && peer == "" {
		return fmt.Errorf("peer cannot be empty if specified")
	}

	return nil
}

// ValidateExecution checks the execution configuration.
func ValidateExecution(_ *map[string]any, data any, _ tree.Path) error {
	execution, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid execution configuration: %v", data)
	}

	// Check execution type
	execType, ok := execution["type"].(string)
	if !ok || execType == "" {
		return fmt.Errorf("execution must have a type")
	}

	// Get params
	params, ok := execution["params"].(map[string]any)
	if !ok {
		return fmt.Errorf("execution must have params")
	}

	// Validate based on execution type
	switch execType {
	case "docker":
		if err := validateDockerExecution(params); err != nil {
			return err
		}
	case "firecracker":
		if err := validateFirecrackerExecution(params); err != nil {
			return err
		}
	case "null":
		// No specific validation for null executor
		return nil
	default:
		return fmt.Errorf("unsupported execution type: %s", execType)
	}

	return nil
}

// validateDockerExecution validates docker-specific execution configuration
func validateDockerExecution(execution map[string]any) error {
	// Validate image (required)
	image, ok := execution["image"].(string)
	if !ok || image == "" {
		return fmt.Errorf("docker execution must have an image")
	}

	// Validate image format with a single regex
	if !vutils.IsDockerImageValid(image) {
		return fmt.Errorf("invalid docker image format: %s", image)
	}

	// Validate entrypoint if present
	if entrypoint, ok := execution["entrypoint"].([]any); ok {
		for i, entry := range entrypoint {
			if _, ok := entry.(string); !ok {
				return fmt.Errorf("docker entrypoint at index %d must be a string", i)
			}
		}
	}

	// Validate command if present
	if cmd, ok := execution["cmd"].([]any); ok {
		for i, c := range cmd {
			if _, ok := c.(string); !ok {
				return fmt.Errorf("docker command at index %d must be a string", i)
			}
		}
	}

	// Validate environment variables if present
	if envs, ok := execution["environment"]; ok {
		v := reflect.ValueOf(envs)
		if v.Kind() != reflect.Slice {
			return fmt.Errorf("docker environment must be a slice")
		}

		for i := range v.Len() {
			envStr, ok := v.Index(i).Interface().(string)
			if !ok {
				return fmt.Errorf("docker environment variable at index %d must be a string", i)
			}
			if envStr == "" {
				return fmt.Errorf("docker environment variable at index %d cannot be empty", i)
			}
			// Verify format is KEY=VALUE
			if !strings.Contains(envStr, "=") {
				return fmt.Errorf("docker environment variable at index %d must be in KEY=VALUE format", i)
			}
			parts := strings.SplitN(envStr, "=", 2)
			if parts[0] == "" {
				return fmt.Errorf("docker environment variable key at index %d cannot be empty", i)
			}
			// Validate environment variable key format
			if !vutils.IsEnvVarKeyValid(parts[0]) {
				return fmt.Errorf("invalid environment variable key format at index %d: %s", i, parts[0])
			}
		}
	}

	// Validate working directory if present
	if workDir, ok := execution["working_directory"].(string); ok && workDir == "" {
		return fmt.Errorf("docker working directory cannot be empty if specified")
	}

	if restartPolicy, ok := execution["restart_policy"].(string); ok {
		if restartPolicy == "" {
			return fmt.Errorf("docker restart_policy cannot be empty if specified")
		}

		// validate restart policy is one of the allowed values
		switch restartPolicy {
		case "no", "on-failure", "always", "unless-stopped":
			// valid policies
		default:
			return fmt.Errorf("invalid docker restart_policy: %s", restartPolicy)
		}
	}

	return nil
}

// validateFirecrackerExecution validates firecracker-specific execution configuration
func validateFirecrackerExecution(execution map[string]any) error {
	// Validate kernel image (required)
	kernelImage, ok := execution["kernel_image"].(string)
	if !ok || kernelImage == "" {
		return fmt.Errorf("firecracker execution must have a kernel_image")
	}

	// Validate root file system (required)
	rootFS, ok := execution["root_file_system"].(string)
	if !ok || rootFS == "" {
		return fmt.Errorf("firecracker execution must have a root_file_system")
	}

	// Validate kernel args if present
	if kernelArgs, ok := execution["kernel_args"].(string); ok && kernelArgs == "" {
		return fmt.Errorf("firecracker kernel_args cannot be empty if specified")
	}

	// Validate initrd if present
	if initrd, ok := execution["initrd"].(string); ok && initrd == "" {
		return fmt.Errorf("firecracker initrd cannot be empty if specified")
	}

	return nil
}

// ValidateSupervisor checks the supervisor configuration.
func ValidateSupervisor(root *map[string]any, data any, _ tree.Path) error {
	supervisor, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid supervisor configuration: %v", data)
	}

	// Validate strategy if specified
	if strategy, ok := supervisor["strategy"].(string); ok {
		if strategy == "" {
			return fmt.Errorf("supervisor strategy cannot be empty if specified")
		}
		// Validate strategy is one of the allowed values
		switch strategy {
		case "OneForOne", "AllForOne", "RestForOne":
			// valid strategies
		default:
			return fmt.Errorf("invalid supervisor strategy: %s", strategy)
		}
	}

	// Validate allocations if specified
	if allocations, ok := supervisor["allocations"].([]any); ok {
		rootAllocs, err := utils.GetConfigAtPath(*root, tree.NewPath("V1.allocations"))
		if err != nil || rootAllocs == nil {
			return fmt.Errorf("allocations must be defined when supervisor is defined")
		}
		rootAllocsMap, ok := rootAllocs.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid allocations configuration")
		}
		for _, alloc := range allocations {
			allocStr, ok := alloc.(string)
			if !ok {
				return fmt.Errorf("invalid allocation reference: %v", alloc)
			}
			if _, exists := rootAllocsMap[allocStr]; !exists {
				return fmt.Errorf("referenced allocation '%s' not found", allocStr)
			}
		}
	}

	// Validate children if specified - only check type since path validation will handle the rest
	if children, ok := supervisor["children"].([]any); ok {
		for i, child := range children {
			if _, ok := child.(map[string]any); !ok {
				return fmt.Errorf("invalid child supervisor at index %d: must be a map", i)
			}
		}
	}

	return nil
}

// validateLocationConstraints validates the location constraints configuration
func validateLocationConstraints(location map[string]any) error {
	// Validate accept locations if present
	if accept, ok := location["accept"].([]any); ok {
		for i, loc := range accept {
			if err := validateLocation(loc, i); err != nil {
				return fmt.Errorf("invalid accept location at index %d: %v", i, err)
			}
		}
	}

	// Validate reject locations if present
	if reject, ok := location["reject"].([]any); ok {
		for i, loc := range reject {
			if err := validateLocation(loc, i); err != nil {
				return fmt.Errorf("invalid reject location at index %d: %v", i, err)
			}
		}
	}

	return nil
}

// validateLocation validates a single location configuration
func validateLocation(data any, _ int) error {
	location, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("location must be a map")
	}

	// continent required if specifying location
	continent, ok := location["continent"].(string)
	if !ok || continent == "" {
		return fmt.Errorf("continent is required when specifying a location")
	}

	// Country is optional
	if country, ok := location["country"].(string); ok && country == "" {
		return fmt.Errorf("country cannot be empty if specified")
	}

	// City is optional but requires country if specified
	if city, ok := location["city"].(string); ok {
		if city == "" {
			return fmt.Errorf("city cannot be empty if specified")
		}
		country, hasCountry := location["country"].(string)
		if !hasCountry || country == "" {
			return fmt.Errorf("country must be specified when city is provided")
		}
	}

	// ASN is optional
	if asn, ok := location["asn"].(uint); ok {
		if asn <= 0 {
			return fmt.Errorf("ASN must be positive if specified")
		}
	}

	// ISP is optional
	if isp, ok := location["isp"].(string); ok && isp == "" {
		return fmt.Errorf("ISP cannot be empty if specified")
	}

	return nil
}

// ValidateEdgeConstraints checks the edge constraints configuration.
func ValidateEdgeConstraints(root *map[string]any, data any, _ tree.Path) error {
	edgeConstraints, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid edge constraints configuration: %v", data)
	}

	// Check if "S" and "T" keys exist and are non-empty
	s, sOk := edgeConstraints["S"].(string)
	t, tOk := edgeConstraints["T"].(string)
	if !sOk || s == "" || !tOk || t == "" {
		return fmt.Errorf("invalid edge constraints configuration: edges should be a pair of named nodes")
	}

	// Validate that S and T are different nodes
	if s == t {
		return fmt.Errorf("edge constraint source and target must be different nodes")
	}

	// Check if "nodes" key exists
	nodesConfig, err := utils.GetConfigAtPath(*root, tree.NewPath("V1.nodes"))
	if err != nil || nodesConfig == nil {
		return fmt.Errorf("invalid edge constraints configuration: nodes must be defined")
	}
	nodes, nodesOk := nodesConfig.(map[string]any)
	if !nodesOk {
		return fmt.Errorf("invalid edge constraints configuration: nodes must be a map")
	}

	// Check if S and T are present in the "nodes" map
	if _, ok := nodes[s]; !ok {
		return fmt.Errorf("invalid edge constraints configuration: node '%s' not found", s)
	}

	if _, ok := nodes[t]; !ok {
		return fmt.Errorf("invalid edge constraints configuration: node '%s' not found", t)
	}

	// Validate RTT and BW if present
	if rtt, ok := edgeConstraints["RTT"].(uint); ok {
		edgeConstraints["RTT"] = rtt
	}

	if bw, ok := edgeConstraints["BW"].(uint); ok {
		edgeConstraints["BW"] = bw
	}

	return nil
}

// ValidateHealthCheck checks the healthcheck configuration.
func ValidateHealthCheck(_ *map[string]any, data any, _ tree.Path) error {
	healthcheck, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid healthcheck configuration: %v", data)
	}

	if len(healthcheck) == 0 {
		return fmt.Errorf("healthcheck cannot be empty if specified")
	}

	// Check healthcheck type
	hcType, ok := healthcheck["type"]
	if !ok || hcType == "" {
		return fmt.Errorf("healthcheck must have a type")
	}

	// Must have exec or endpoint
	switch hcType {
	case "http":
		if _, ok := healthcheck["endpoint"]; !ok {
			return fmt.Errorf("http type healthcheck must have an endpoint")
		}
	case "command":
		if _, ok := healthcheck["exec"]; !ok {
			return fmt.Errorf("command type healthcheck must have exec")
		}
	default:
		return fmt.Errorf("unsupported healthcheck type: %s", hcType)
	}

	if interval, ok := healthcheck["interval"].(time.Duration); ok {
		if interval == 0 {
			return fmt.Errorf("healthcheck interval must be greater than 0")
		}
	}

	return nil
}

// ValidateSubnet validates the subnet config
func ValidateSubnet(_ *map[string]any, data any, _ tree.Path) error {
	subnet, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid subnet config")
	}

	if len(subnet) == 0 {
		return fmt.Errorf("subnet can not be empty if specified")
	}

	if _, ok := subnet["join"].(bool); !ok {
		return fmt.Errorf("subnet.join expects boolean value")
	}

	return nil
}

// ValidateContract validates the contract configuration.
// It checks that the contract has a valid DID and host format.
// Payment details are validated at contract creation time, not here.
func ValidateContract(_ *map[string]any, data any, _ tree.Path) error {
	contract, ok := data.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid contract configuration: %v", data)
	}

	// Validate DID
	did, ok := contract["did"].(string)
	if !ok {
		return fmt.Errorf("contract 'did' must be a string")
	}
	if !strings.HasPrefix(did, "did:") {
		return fmt.Errorf("invalid did format")
	}

	// Validate host (if present)
	if host, ok := contract["host"].(string); ok {
		if !strings.HasPrefix(host, "did:") {
			return fmt.Errorf("invalid host did format")
		}
	}

	// Payment details are validated at contract creation time, not here
	return nil
}

// ValidateVolume validates the allocation's volume config
func ValidateVolume(_ *map[string]any, data any, _ tree.Path) error {
	volumes, ok := data.([]any)
	if !ok {
		return fmt.Errorf("invalid volume configuration: %v", data)
	}

	if len(volumes) == 0 {
		return fmt.Errorf("volume cannot be empty if specified")
	}

	for _, vol := range volumes {
		volume, ok := vol.(map[string]any)
		if !ok {
			return fmt.Errorf("invalid type: %T - expecting map[string]", vol)
		}

		// Check volume type
		volType, ok := volume["type"]
		if !ok || volType == "" {
			return fmt.Errorf("volume must have a type")
		}

		// types for now are local or glusterfs
		switch volType {
		case "glusterfs":
			servers, ok := volume["servers"].([]string)
			if !ok {
				return fmt.Errorf("glusterfs type volume must define one or more servers")
			}
			if len(servers) == 0 {
				return fmt.Errorf("glusterfs type volume must define at least one server")
			}
			for i, server := range servers {
				serverStr := server
				if serverStr == "" {
					return fmt.Errorf("glusterfs volume server at index %d must be a non-empty string", i)
				}
			}
		case "local":
			volumeSrc, ok := volume["src"].(string)
			if !ok {
				return fmt.Errorf("local type volume must define source destination as 'src'")
			}

			if strings.Contains(volumeSrc, "/") {
				// validate as a path
				if !filepath.IsAbs(volumeSrc) {
					return fmt.Errorf("local type volume source must be an absolute path")
				}
			} else {
				// validate as a named volume (alphanumeric, dashes, underscores and dot only)
				// ignoring linter because only in the case of multiple named volumes will the regex run multiple times
				matched, err := regexp.MatchString(`^[a-zA-Z0-9][a-zA-Z0-9_.-]*$`, volumeSrc) //nolint:staticcheck
				if err != nil {
					return fmt.Errorf("error validating local volume src: %w", err)
				}
				if !matched {
					return fmt.Errorf("local volume src contains invalid characters")
				}
			}
		default:
			return fmt.Errorf("unsupported volume type: %s", volType)
		}

		if _, ok := volume["mount_destination"]; !ok {
			return fmt.Errorf("volume must define a mount destination")
		}
	}
	return nil
}
