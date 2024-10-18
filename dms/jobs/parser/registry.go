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
