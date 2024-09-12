package types

import (
	"reflect"
)

// ExecutorType is the type of the executor
type ExecutorType string

const (
	ExecutorTypeDocker      ExecutorType = "docker"
	ExecutorTypeFirecracker ExecutorType = "firecracker"
	ExecutorTypeWasm        ExecutorType = "wasm"

	ExecutionStatusCodeSuccess = 0
)

// implementing Comparable interface
var _ Comparable[ExecutorType] = (*ExecutorType)(nil)

// Compare compares two ExecutorType objects
func (e ExecutorType) Compare(other ExecutorType) Comparison {
	return LiteralComparator(string(e), string(other))
}

// String returns the string representation of the ExecutorType
func (e ExecutorType) String() string {
	return string(e)
}

// Executor is the executor type
type Executor struct {
	ExecutorType ExecutorType `json:"executor_type"`
}

// implementing Comparable interface
var _ Comparable[Executor] = (*Executor)(nil)

// Compare compares two Executor objects
func (e *Executor) Compare(other Executor) Comparison {
	// comparator for  Executor types
	// it is needed because executor type is defined as enum of ExecutorType's in types.execution.go
	// left represent machine capabilities
	// right represent required capabilities
	// it is not so complex as the type has only one field
	// therefore this method just passes it through...

	return e.ExecutorType.Compare(other.ExecutorType)
}

// Equal checks if two Executor objects are equal
func (e *Executor) Equal(executor Executor) bool {
	return e.ExecutorType == executor.ExecutorType
}

// Executors is a list of Executor objects
type Executors []Executor

// implementing Comparable and Calculable interface
var (
	_ Comparable[Executors] = (*Executors)(nil)
	_ Calculable[Executors] = (*Executors)(nil)
)

// Add adds the Executor object to another Executor object
func (e *Executors) Add(other Executors) error {
	// append to Executors slice
	*e = append(*e, other...)
	return nil
}

// Subtract subtracts the Executor object from another Executor object
func (e *Executors) Subtract(other Executors) error {
	if len(other) == 0 {
		return nil
	}

	toRemove := make(map[ExecutorType]struct{})
	for _, ex := range other {
		toRemove[ex.ExecutorType] = struct{}{}
	}

	result := (*e)[:0]
	for _, ex := range *e {
		if _, found := toRemove[ex.ExecutorType]; !found {
			result = append(result, ex)
		}
	}

	*e = result[:len(result):len(result)]
	return nil
}

// Contains checks if an Executor object is in the list of Executors
func (e *Executors) Contains(executor Executor) bool {
	executors := *e
	for _, ex := range executors {
		if ex.Equal(executor) {
			return true
		}
	}
	return false
}

// Compare compares two Executors objects
func (e *Executors) Compare(other Executors) Comparison {
	if reflect.DeepEqual(*e, other) {
		return Equal
	}

	// comparator for Executors types:
	// left represent machine capabilities;
	// right represent required capabilities;
	lSlice := make([]interface{}, 0, len(*e))
	rSlice := make([]interface{}, 0, len(other))

	for _, ex := range *e {
		lSlice = append(lSlice, ex)
	}
	for _, ex := range other {
		rSlice = append(rSlice, ex)
	}

	if !IsSameShallowType(lSlice, rSlice) {
		return Error
	}

	switch {
	case reflect.DeepEqual(lSlice, rSlice):
		// if available capabilities are
		// equal to required capabilities
		// then the result of comparison is 'Equal'
		return Equal

	case IsStrictlyContained(lSlice, rSlice):
		// if machine capabilities contain all the required capabilities
		// then the result of comparison is 'Better'
		return Better

	case IsStrictlyContained(rSlice, lSlice):
		// if required capabilities contain all the machine capabilities
		// then the result of comparison is 'Worse'
		// ("available Capabilities are worse than required")')
		// (note that Equal case is already handled above)
		return Worse
	default:
		return Error
	}
}
