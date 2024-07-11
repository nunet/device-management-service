package repositories

import (
	"fmt"
	"reflect"
)

// UpdateField is a generic function that updates a field of a struct or a pointer to a struct.
// The function uses reflection to dynamically update the specified field of the input struct.
func UpdateField[T interface{}](input T, fieldName string, newValue interface{}) (T, error) {
	// Use reflection to get the struct's field
	val := reflect.ValueOf(input)
	if val.Kind() == reflect.Ptr {
		// If input is a pointer, get the underlying element
		val = val.Elem()
	} else {
		// If input is not a pointer, ensure it's addressable
		val = reflect.ValueOf(&input).Elem()
	}

	// Check if the input is a struct
	if val.Kind() != reflect.Struct {
		return input, fmt.Errorf("Not a struct: %T", input)
	}

	// Get the field by name
	field := val.FieldByName(fieldName)
	if !field.IsValid() {
		return input, fmt.Errorf("Field not found: %v", fieldName)
	}

	// Check if the field is settable
	if !field.CanSet() {
		return input, fmt.Errorf("Field not settable: %v", fieldName)
	}

	// Check if types are compatible
	if !reflect.TypeOf(newValue).ConvertibleTo(field.Type()) {
		return input, fmt.Errorf(
			"Incompatible conversion: %v -> %v; value: %v",
			field.Type(), reflect.TypeOf(newValue), newValue,
		)
	}

	// Convert the new value to the field type
	convertedValue := reflect.ValueOf(newValue).Convert(field.Type())

	// Set the new value to the field
	field.Set(convertedValue)

	return input, nil
}

// IsEmptyValue checks if value represents a zero-value struct (or pointer to a zero-value struct) using reflection.
// The function is useful for determining if a struct or its pointer is empty, i.e., all fields have their zero-values.
func IsEmptyValue(value interface{}) bool {
	// Check if the value is nil
	if value == nil {
		return true
	}

	// Use reflection to get the value's type and kind
	val := reflect.ValueOf(value)

	// If the value is a pointer, dereference it to get the underlying element
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	// Check if the value is zero (empty) based on its kind
	return val.IsZero()
}
