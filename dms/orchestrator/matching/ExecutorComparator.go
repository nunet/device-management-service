package matching

import (
	"gitlab.com/nunet/device-management-service/models"
)

func ExecutorComparator(lraw, rraw interface{}, preference ...Preference) models.Comparison {
	// comparator for  Executor types
	// it is needed because executor type is defined as enum of ExecutorType's in models/execution.go
	// left represent machine capabilities
	// right represent required capabilities
	// it is not so complex as the type has only one field
	// therefore this method just passes it through...

	// validate input type
	_, lrawok := lraw.(models.Executor)
	_, rrawok := rraw.(models.Executor)
	if !lrawok || !rrawok {
		return models.Error
	}
	l := lraw.(models.Executor)
	r := rraw.(models.Executor)

	leftExecutorType := l.ExecutorType
	rightExecutorType := r.ExecutorType
	comparison := Compare(leftExecutorType, rightExecutorType)
	return comparison
}