package types

import "reflect"

// Comparable public Comparable interface to be enforced on types that can be compared
type Comparable[T any] interface {
	Compare(other T) (Comparison, error)
}

type Calculable[T any] interface {
	Add(other T) error
	Subtract(other T) error
}

type PreferenceString string

const (
	Hard PreferenceString = "Hard"
	Soft PreferenceString = "Soft"
)

type Preference struct {
	TypeName                  string
	Strength                  PreferenceString
	DefaultComparatorOverride Comparator
}

type Comparator func(l, r interface{}, preference ...Preference) Comparison

// ComplexCompare helper function to return a complex comparison of two complex types
// this uses reflection, could become a performance bottleneck
// with generics, we wouldn't need this function and could use the ComplexCompare method directly
func ComplexCompare(l, r interface{}) ComplexComparison {
	// Complex comparison is a comparison of two complex types
	// Which have nested fields that need to be considered together
	// before a final comparison for the whole complex type can be made
	// it is a helper function used in some type comparators

	complexComparison := make(map[string]Comparison)
	val1 := reflect.ValueOf(l)
	val2 := reflect.ValueOf(r)
	for i := 0; i < val1.NumField(); i++ {
		field1 := val1.Field(i)
		field2 := val2.Field(i)

		// we need a pointer to field1 to call the ComplexCompare method
		// if the field is not addressable we need to create a pointer to it
		// and then call the ComplexCompare method on it
		var field1Ptr reflect.Value
		if field1.CanAddr() {
			field1Ptr = field1.Addr()
		} else {
			ptr := reflect.New(field1.Type())
			ptr.Elem().Set(field1)
			field1Ptr = ptr
		}

		compareMethod := field1Ptr.MethodByName("Compare")
		if compareMethod.IsValid() &&
			compareMethod.Type().NumIn() == 1 &&
			compareMethod.Type().In(0) == field1.Type() {
			result := compareMethod.Call([]reflect.Value{field2})[0].Interface().(Comparison)
			complexComparison[val1.Type().Field(i).Name] = result
		} else {
			complexComparison[val1.Type().Field(i).Name] = None
		}
	}

	return complexComparison
}

func NumericComparator[T float64 | float32 | int | int32 | int64 | uint64](l, r T, _ ...Preference) Comparison {
	// comparator for numeric types:
	// left represents machine capabilities;
	// right represents required capabilities;

	switch {
	case l == r:
		return Equal

	case l < r:
		return Worse

	default:
		return Better
	}
}

func LiteralComparator[T ~string](l, r T, _ ...Preference) Comparison {
	// comparator for literal (string-like) types:
	// left represents machine capabilities;
	// right represents required capabilities;
	// which can only be equal or not equal.

	// ComplexCompare the string values
	if l == r {
		return Equal
	}

	return None
}
