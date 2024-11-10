// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package parser

import (
	"sync"
)

type Registry struct {
	parsers map[SpecType]Parser
	mu      sync.RWMutex
}

func (r *Registry) RegisterParser(specType SpecType, p Parser) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parsers[specType] = p
}

func (r *Registry) GetParser(specType SpecType) (Parser, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, exists := r.parsers[specType]
	return p, exists
}
