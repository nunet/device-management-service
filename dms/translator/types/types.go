// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	jobtypes "gitlab.com/nunet/device-management-service/dms/jobs/types"
)

// Translation represents the result of a translation, including the configuration
// and any warnings about unsupported features.
type Translation struct {
	Config   *jobtypes.EnsembleConfig
	Warnings []string
}

// Translator defines the interface for converting a source configuration file
// into a NuNet EnsembleConfig.
type Translator interface {
	Translate(input []byte) (*Translation, error)
}
