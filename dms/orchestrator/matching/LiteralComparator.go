package matching

import (
	"gitlab.com/nunet/device-management-service/types"
)

func LiteralComparator(l, r interface{}, preference ...Preference) types.Comparison {
	// comparator for literal (basically string) types:
	// left represent machine capabilities;
	// right represent required capabilities;
	// which can be only equal or not equal...


	// validate input type
	_, lok := l.(string)
	_, rok := r.(string)
	if !lok || !rok {
		return types.Error
	}

	var result types.Comparison
	result = types.Error // error is the default value
	switch l.(type) {
	case string:
		if l == r {
			result = types.Equal
		} 
	}
	return result
}

