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

	"gitlab.com/nunet/device-management-service/dms/jobs/parser/transform"
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/tree"
)

func NewEnsemblev1Encoder() transform.Transformer {
	return transform.NewTransformer(
		[]map[tree.Path]transform.TransformerFunc{
			{
				"": FormatSpec,
			},
			{
				"allocations.*.execution":         transform.FlattenSpecConfigTransformer("execution"),
				"allocations.*.volumes.[].remote": transform.FlattenSpecConfigTransformer("volume remote"),
			},
			{
				"allocations.*.resources.cpu.clock_speed": transform.ToSIFormatWithUnit("cpu clock_speed", "Hz"),
				"allocations.*.resources.ram.clock_speed": transform.ToSIFormatWithUnit("ram clock_speed", "Hz"),
				"allocations.*.resources.ram.size":        transform.ToBytesFormat("ram size"),
				"allocations.*.resources.disk.size":       transform.ToBytesFormat("disk size"),
				"allocations.*.resources.gpu.[].vram":     transform.ToBytesFormat("gpu vram"),
			},
			{
				"allocations.*.volumes": transform.NamedSliceToMapTransformer("volumes"),
			},
		},
	)
}

func FormatSpec(_ *map[string]interface{}, data any, _ tree.Path) (any, error) {
	spec, ok := data.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid spec configuration: %v", data)
	}

	v1, ok := spec["v1"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid spec configuration: %v", data)
	}
	v1["version"] = "v1"

	return v1, nil
}
