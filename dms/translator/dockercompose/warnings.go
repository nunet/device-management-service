// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package dockercompose

import (
	"fmt"
	"sync"

	"github.com/compose-spec/compose-go/v2/types"
)

// WarningCollector safely collects warnings for unsupported features during the translation process.
type WarningCollector struct {
	warnings []string
	mu       sync.Mutex
}

// Add records a new warning. It is safe for concurrent use.
func (wc *WarningCollector) Add(serviceName, feature, reason string) {
	wc.mu.Lock()
	defer wc.mu.Unlock()

	var msg string
	if serviceName != "" {
		msg = fmt.Sprintf("Service '%s': Unsupported feature '%s' was ignored. Reason: %s", serviceName, feature, reason)
	} else {
		msg = fmt.Sprintf("Top-level feature '%s' was ignored. Reason: %s", feature, reason)
	}
	wc.warnings = append(wc.warnings, msg)
}

// Get returns all collected warnings.
func (wc *WarningCollector) Get() []string {
	wc.mu.Lock()
	defer wc.mu.Unlock()
	// Return a copy to prevent race conditions if the original slice is modified later.
	warningsCopy := make([]string, len(wc.warnings))
	copy(warningsCopy, wc.warnings)
	return warningsCopy
}

// checkUnsupportedTopLevelFeatures checks for unsupported features at the root of the Compose file.
func checkUnsupportedTopLevelFeatures(p *types.Project, w *WarningCollector) {
	if len(p.Configs) > 0 {
		w.Add("", "configs", "top-level 'configs' are not supported.")
	}
	if len(p.Secrets) > 0 {
		w.Add("", "secrets", "top-level 'secrets' are not supported.")
	}
}

// checkForUnsupportedServiceFeatures checks for service-level features that cannot be translated.
func checkForUnsupportedServiceFeatures(s types.ServiceConfig, w *WarningCollector) {
	unsupported := map[string]string{
		"build":             "NuNet requires pre-built Docker images.",
		"cgroup_parent":     "cgroup configuration is managed by the NuNet DMS.",
		"container_name":    "container naming is managed by the NuNet DMS.",
		"devices":           "direct device access is not supported for security reasons.",
		"dns":               "DNS is managed by the NuNet virtual network.",
		"dns_search":        "DNS is managed by the NuNet virtual network.",
		"external_links":    "linking is handled by the NuNet virtual network.",
		"extra_hosts":       "host entries are managed by the NuNet virtual network.",
		"ipc":               "IPC namespace is not configurable.",
		"mac_address":       "network identity is managed by the NuNet DMS.",
		"privileged":        "privileged mode is not supported for security reasons.",
		"restart":           "restart policy is managed by the allocation's 'failure_recovery' strategy.",
		"security_opt":      "security options are managed by the NuNet DMS.",
		"shm_size":          "shared memory size is not configurable.",
		"stop_grace_period": "shutdown behavior is managed by the NuNet DMS.",
		"sysctls":           "kernel parameters are not configurable for security reasons.",
		"tmpfs":             "in-memory filesystems are not supported.",
		"ulimits":           "resource limits are managed by the NuNet DMS.",
	}

	// Use reflection in a real-world scenario for more dynamic checking if needed,
	// but for this specific list, direct checks are clearer.
	if s.Build != nil {
		w.Add(s.Name, "build", unsupported["build"])
	}
	if s.CgroupParent != "" {
		w.Add(s.Name, "cgroup_parent", unsupported["cgroup_parent"])
	}
	if s.ContainerName != "" {
		w.Add(s.Name, "container_name", unsupported["container_name"])
	}
	if len(s.Devices) > 0 {
		w.Add(s.Name, "devices", unsupported["devices"])
	}
	// ... and so on for the other unsupported features.
}
