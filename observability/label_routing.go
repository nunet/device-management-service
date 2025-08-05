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
