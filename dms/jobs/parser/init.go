// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package parser

import (
	"gitlab.com/nunet/device-management-service/dms/jobs/parser/ensemblev1"
)

var registry *Registry

func init() {
	registry = &Registry{
		parsers: make(map[SpecType]Parser),
	}

	// Register Nunet parser.
	ensembleV1Parser := NewBasicParser(
		ensemblev1.NewEnsemblev1Transformer(),
		ensemblev1.NewEnsembleV1Validator(),
	)
	registry.RegisterParser(SpecTypeEnsembleV1, ensembleV1Parser)
}
