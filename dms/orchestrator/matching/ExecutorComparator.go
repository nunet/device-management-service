package matching

import (
	"gitlab.com/nunet/device-management-service/types"
)

func ExecutorComparator(lraw, rraw interface{}, preference ...Preference) types.Comparison {
	// comparator for  Executor types
	// it is needed because executor type is defined as enum of ExecutorType's in types.execution.go
	// left represent machine capabilities
	// right represent required capabilities
	// it is not so complex as the type has only one field
	// therefore this method just passes it through...

	// validate input type
	_, lrawok := lraw.(types.Executor)
	_, rrawok := rraw.(types.Executor)
	if !lrawok || !rrawok {
		return types.Error
	}
	l := lraw.(types.Executor)
	r := rraw.(types.Executor)

	leftExecutorType := l.ExecutorType
	rightExecutorType := r.ExecutorType
	comparison := Compare(leftExecutorType, rightExecutorType)
	return comparison
}