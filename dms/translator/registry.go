// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package translator

import (
	"sync"

	"gitlab.com/nunet/device-management-service/dms/translator/types"
)

type Registry struct {
	translators map[SpecType]types.Translator
	mu          sync.RWMutex
}

func (r *Registry) RegisterTranslator(specType SpecType, t types.Translator) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.translators[specType] = t
}

func (r *Registry) GetTranslator(specType SpecType) (types.Translator, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, exists := r.translators[specType]
	return p, exists
}
