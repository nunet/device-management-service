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

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// labelInjectionCore ensures "labels" is set and sets "es_skip"/"es_index".
type labelInjectionCore struct {
	next         zapcore.Core
	levelEnabler zapcore.LevelEnabler
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
	skipES, overrideIndex := GetLableRoutingConfig(mergedLabels)
	if skipES {
		finalFields = append(finalFields, zap.Bool("es_skip", true))
	}
	if overrideIndex != "" {
		finalFields = append(finalFields, zap.String("es_index", overrideIndex))
	}

	return l.next.Write(ent, finalFields)
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
