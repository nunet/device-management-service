// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

// observability\label_routing.go

package observability

// LogLabel is the type used for routing decisions based on label names.
type LogLabel string

// These constants define the labels we can attach to log entries.
const (
	LabelDefault    LogLabel = "default"
	LabelAccounting LogLabel = "accounting"
	LabelMetric     LogLabel = "metric"
	LabelDeployment LogLabel = "deployment"
	LabelAllocation LogLabel = "allocation"
	LabelNode       LogLabel = "node"
	LabelContract   LogLabel = "contract"
	LabelUser       LogLabel = "user"
)

// LabelRoutingConfig defines optional routing rules per label.
type LabelRoutingConfig struct {
	// If SkipES is true, logs with this label will not be sent to Elasticsearch.
	SkipES bool

	// If ESIndex is non-empty, logs with this label will be routed to that ES index
	// instead of the default/logs index.
	ESIndex string
}

// labelRoutingMap is our in-memory map from label → routing configuration.
var labelRoutingMap = map[LogLabel]LabelRoutingConfig{
	LabelContract: {
		SkipES:  false,
		ESIndex: "contract-index",
	},
	LabelAccounting: {
		SkipES:  false,
		ESIndex: "accounting-index",
	},
	LabelMetric: {
		SkipES:  false,
		ESIndex: "metric-index",
	},
	LabelDeployment: {
		SkipES:  false,
		ESIndex: "deployment-index",
	},
	LabelAllocation: {
		SkipES:  false,
		ESIndex: "allocation-index",
	},
	LabelNode: {
		SkipES:  false,
		ESIndex: "node-index",
	},
	LabelUser: {
		SkipES:  false,
		ESIndex: "user-index",
	},
}

// GetRoutingConfig inspects the provided labels and returns whether logs
// should be skipped for ES (skipES) and which ES index to route them to (esIndex).
func GetLableRoutingConfig(labels []string) (skipES bool, esIndex string) {
	for _, lbl := range labels {
		cfg, exists := labelRoutingMap[LogLabel(lbl)]
		if !exists {
			continue
		}
		if cfg.SkipES {
			skipES = true
		}
		if cfg.ESIndex != "" {
			esIndex = cfg.ESIndex
		}
	}
	return
}
