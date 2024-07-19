package validate

import (
	"reflect"
)

func ConvertNumericToFloat64(n any) (float64, bool) {
	switch n := n.(type) {
	case int, int8, int16, int32, int64:
		return float64(reflect.ValueOf(n).Int()), true
	case uint, uint8, uint16, uint32, uint64:
		return float64(reflect.ValueOf(n).Uint()), true
	case float32:
		return float64(n), true
	case float64:
		return n, true
	default:
		return 0, false
	}
}
