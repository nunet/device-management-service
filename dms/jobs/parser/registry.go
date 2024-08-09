package parser

import (
	"sync"
)

type Registry[T any] interface {
	GetParser(specType SpecType) (Parser[T], bool)
	RegisterParser(specType SpecType, p Parser[T])
}

type RegistryImpl[T any] struct {
	parsers map[SpecType]Parser[T]
	mu      sync.RWMutex
}

func (r *RegistryImpl[T]) RegisterParser(specType SpecType, p Parser[T]) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.parsers[specType] = p
}

func (r *RegistryImpl[T]) GetParser(specType SpecType) (Parser[T], bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, exists := r.parsers[specType]
	return p, exists
}
