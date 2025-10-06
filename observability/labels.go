// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package observability

import (
	"reflect"
	"runtime/debug"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogLabel is the type used for routing decisions based on label names.
type LogLabel string

// These constants define the labels we can attach to log entries.
// TODO desc for each label
const (
	LabelDefault LogLabel = "default"
	// TODO merge with LabelContract?
	LabelAccounting LogLabel = "accounting"
	LabelMetric     LogLabel = "metric"
	LabelDeployment LogLabel = "deployment"
	LabelAllocation LogLabel = "allocation"
	LabelNode       LogLabel = "node"
	LabelContract   LogLabel = "contract"
	// TODO unused
	LabelUser LogLabel = "user"
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

// labelInjectionCore ensures "labels" is set and sets "es_skip"/"es_index".
type labelInjectionCore struct {
	next         zapcore.Core
	levelEnabler zapcore.LevelEnabler
}

// GetLabelRoutingConfig inspects the provided labels and returns whether logs
// should be skipped for ES (skipES) and which ES index to route them to (esIndex).
func GetLabelRoutingConfig(labels []string) (skipES bool, esIndex string) {
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

func newLabelInjectionCore(next zapcore.Core, enabler zapcore.LevelEnabler) zapcore.Core {
	return &labelInjectionCore{
		next:         next,
		levelEnabler: enabler,
	}
}

func (l *labelInjectionCore) Enabled(level zapcore.Level) bool {
	return l.levelEnabler.Enabled(level)
}

func (l *labelInjectionCore) With(fields []zapcore.Field) zapcore.Core {
	return &labelInjectionCore{
		next:         l.next.With(fields),
		levelEnabler: l.levelEnabler,
	}
}

func (l *labelInjectionCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if l.Enabled(ent.Level) {
		return ce.AddCore(ent, l)
	}
	return ce
}

// Write merges or sets a single "labels" field, then determines "es_skip" and "es_index".
func (l *labelInjectionCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	mergedLabels := make([]string, 0)
	uniqueLabels := make(map[string]bool)

	// We'll store all non-"labels" fields in a new slice and handle "labels" separately.
	finalFields := make([]zapcore.Field, 0, len(fields))

	for _, f := range fields {
		if f.Key == "labels" {
			labels := extractLabels([]zapcore.Field{f})
			for _, lbl := range labels {
				if lbl == "" {
					continue
				}
				if !uniqueLabels[lbl] {
					uniqueLabels[lbl] = true
					mergedLabels = append(mergedLabels, lbl)
				}
			}
		} else {
			finalFields = append(finalFields, f)
		}
	}

	// Default to ["default"] if no labels found.
	if len(mergedLabels) == 0 {
		mergedLabels = []string{string(LabelDefault)}
	}
	finalFields = append(finalFields, zap.Strings("labels", mergedLabels))

	// Use the routing logic to decide skipES, overrideIndex.
	skipES, overrideIndex := GetLabelRoutingConfig(mergedLabels)
	if skipES {
		finalFields = append(finalFields, zap.Bool("es_skip", true))
	}
	if overrideIndex != "" {
		finalFields = append(finalFields, zap.String("es_index", overrideIndex))
	}

	finalFields = l.gatherFields(ent, fields, finalFields)
	finalFields = append(finalFields, zap.String("transaction.id", rootTransaction.TraceContext().Trace.String()))

	return l.next.Write(ent, finalFields)
}

func (l *labelInjectionCore) gatherFields(
	ent zapcore.Entry, fields []zapcore.Field, finalFields []zapcore.Field,
) []zapcore.Field {
	// catch WARN and ERROR and create spans
	switch {
	case ent.Level == zapcore.WarnLevel || ent.Level == zapcore.ErrorLevel:
		// extract error
		var errMsg string
		for _, f := range fields {
			if f.Key != "error" {
				continue
			}
			if f.String != "" {
				errMsg = " " + f.String
			} else if err, ok := f.Interface.(error); ok {
				errMsg = " " + err.Error()
			}
		}
		errSnippet := errMsg

		// optionally attach an unstructured log msg
		var logMsg string
		if fields == nil {
			logMsg = ent.Message[0:min(len(ent.Message), 300)]
		}

		if len(errSnippet) == 0 {
			errSnippet = " " + logMsg
		}
		if len(errSnippet) > 30 {
			errSnippet = errSnippet[:30] + "..."
		}

		// keep in sync with the call path
		skipFrames := 13
		// get a stack trace
		sTrace := strings.Split(string(debug.Stack()), "\n")[skipFrames:]
		if len(sTrace) > 0 && sTrace[len(sTrace)-1] == "" {
			sTrace = sTrace[:len(sTrace)-1]
		}

		// create span
		end := StartSpan(strings.ToUpper(ent.Level.String())+errSnippet,
			"error", errMsg,
			"logMsg", logMsg,
			// format stack for Kibana
			"stack_trace", strings.Join(sTrace, "\n ------ "),
		)
		var spanID string
		if len(activeSpans) > 0 {
			spanID = activeSpans[len(activeSpans)-1].TraceContext().Span.String()
		} else {
			spanID = "no-active-span"
		}
		end()
		// bind this log msg to this err span, add the stack trace
		// sTraceStr, _ := json.Marshal(sTrace)
		for i, v := range sTrace {
			sTrace[i] = strings.Replace(v, "\t", "    ", 1)
		}
		finalFields = append(finalFields,
			zap.String("span.id", spanID),
			zap.Dict("stack_trace", zap.Strings("_", sTrace)))

	case len(activeSpans) > 0:
		// bind non-err logs to traces
		latestSpan := activeSpans[len(activeSpans)-1]
		finalFields = append(finalFields, zap.String("span.id", latestSpan.TraceContext().Span.String()))

	case latestSpanID != "":
		finalFields = append(finalFields, zap.String("span.id", latestSpanID))
	}
	return finalFields
}

func (l *labelInjectionCore) Sync() error {
	return l.next.Sync()
}

// extractLabels looks for a field with key == "labels" and returns a string slice.
func extractLabels(fields []zapcore.Field) []string {
	for _, f := range fields {
		if f.Key != "labels" {
			continue
		}

		// CASE 1: Directly handle a single string field
		if f.Type == zapcore.StringType {
			return []string{f.String}
		}

		// CASE 2: Switch on f.Interface's concrete type
		switch val := f.Interface.(type) {
		case []string:
			return val

		case string:
			// e.g. if we only have one label
			return []string{val}

		case []interface{}:
			// e.g. if the library passes an array of interfaces
			var s []string
			for _, i := range val {
				if str, ok := i.(string); ok {
					s = append(s, str)
				}
			}
			return s
		}

		// CASE 3: If the field is using reflection (zapcore.ReflectType)
		if f.Type == zapcore.ReflectType {
			rv := reflect.ValueOf(f.Interface)
			if rv.Kind() == reflect.Slice {
				var s []string
				for i := 0; i < rv.Len(); i++ {
					elem := rv.Index(i).Interface()
					if str, ok := elem.(string); ok {
						s = append(s, str)
					}
				}
				return s
			}
		}
	}

	// If we found no "labels" field or couldn't parse it, return nil
	return nil
}
