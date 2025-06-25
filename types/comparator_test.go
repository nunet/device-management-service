// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package types

import (
	"testing"
)

func TestNumericComparator(t *testing.T) {
	t.Parallel()

	t.Run("int checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name     string
			left     int
			right    int
			expected Comparison
		}{
			{
				name:     "left is equal to right",
				left:     1,
				right:    1,
				expected: Equal,
			},
			{
				name:     "left is less than right",
				left:     1,
				right:    2,
				expected: Worse,
			},
			{
				name:     "left is greater than right",
				left:     2,
				right:    1,
				expected: Better,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				actual := NumericComparator(tt.left, tt.right)
				if actual != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, actual)
				}
			})
		}
	})

	t.Run("float64 checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name     string
			left     float64
			right    float64
			expected Comparison
		}{
			{
				name:     "left is equal to right",
				left:     1.0,
				right:    1.0,
				expected: Equal,
			},
			{
				name:     "left is less than right",
				left:     1.0,
				right:    2.0,
				expected: Worse,
			},
			{
				name:     "left is greater than right",
				left:     2.0,
				right:    1.0,
				expected: Better,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				actual := NumericComparator(tt.left, tt.right)
				if actual != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, actual)
				}
			})
		}
	})

	t.Run("uint64 checks", func(t *testing.T) {
		t.Parallel()
		tests := []struct {
			name     string
			left     uint64
			right    uint64
			expected Comparison
		}{
			{
				name:     "left is equal to right",
				left:     10,
				right:    10,
				expected: Equal,
			},
			{
				name:     "left is less than right",
				left:     5,
				right:    10,
				expected: Worse,
			},
			{
				name:     "left is greater than right",
				left:     15,
				right:    10,
				expected: Better,
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				t.Parallel()
				actual := NumericComparator(tt.left, tt.right)
				if actual != tt.expected {
					t.Errorf("expected %s, got %s", tt.expected, actual)
				}
			})
		}
	})
}

func TestLiteralComparator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		left     string
		right    string
		expected Comparison
	}{
		{
			name:     "left is equal to right",
			left:     "test",
			right:    "test",
			expected: Equal,
		},
		{
			name:     "left is not equal to right",
			left:     "test",
			right:    "test2",
			expected: None,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := LiteralComparator(tt.left, tt.right)
			if actual != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, actual)
			}
		})
	}
}

// TestStruct is used for testing ComplexCompare
type TestStruct struct {
	IntField    int
	StringField string
	FloatField  float64
	UintField   uint64
}

// Compare implements the Comparable interface for TestStruct
func (ts *TestStruct) Compare(other TestStruct) (Comparison, error) {
	intComp := NumericComparator(ts.IntField, other.IntField)
	strComp := LiteralComparator(ts.StringField, other.StringField)
	floatComp := NumericComparator(ts.FloatField, other.FloatField)
	uintComp := NumericComparator(ts.UintField, other.UintField)

	// Simple logic to determine overall comparison
	if intComp == Equal && strComp == Equal && floatComp == Equal && uintComp == Equal {
		return Equal, nil
	}
	return None, nil
}

// GetIntComparison returns the comparison for the IntField
func (ts *TestStruct) GetIntComparison(other TestStruct) Comparison {
	return NumericComparator(ts.IntField, other.IntField)
}

// GetStringComparison returns the comparison for the StringField
func (ts *TestStruct) GetStringComparison(other TestStruct) Comparison {
	return LiteralComparator(ts.StringField, other.StringField)
}

// GetFloatComparison returns the comparison for the FloatField
func (ts *TestStruct) GetFloatComparison(other TestStruct) Comparison {
	return NumericComparator(ts.FloatField, other.FloatField)
}

// GetUintComparison returns the comparison for the UintField
func (ts *TestStruct) GetUintComparison(other TestStruct) Comparison {
	return NumericComparator(ts.UintField, other.UintField)
}

func TestComplexCompare(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		left     any
		right    any
		expected ComplexComparison
	}{
		{
			name: "all fields equal",
			left: TestStruct{
				IntField:    10,
				StringField: "test",
				FloatField:  1.5,
				UintField:   100,
			},
			right: TestStruct{
				IntField:    10,
				StringField: "test",
				FloatField:  1.5,
				UintField:   100,
			},
			expected: ComplexComparison{
				"IntField":    Equal,
				"StringField": Equal,
				"FloatField":  Equal,
				"UintField":   Equal,
			},
		},
		{
			name: "mixed comparison results",
			left: TestStruct{
				IntField:    20,
				StringField: "test",
				FloatField:  1.0,
				UintField:   200,
			},
			right: TestStruct{
				IntField:    10,
				StringField: "test",
				FloatField:  2.0,
				UintField:   100,
			},
			expected: ComplexComparison{
				"IntField":    Better,
				"StringField": Equal,
				"FloatField":  Worse,
				"UintField":   Better,
			},
		},
		{
			name: "all fields different",
			left: TestStruct{
				IntField:    5,
				StringField: "left",
				FloatField:  3.0,
				UintField:   50,
			},
			right: TestStruct{
				IntField:    10,
				StringField: "right",
				FloatField:  2.0,
				UintField:   100,
			},
			expected: ComplexComparison{
				"IntField":    Worse,
				"StringField": None,
				"FloatField":  Better,
				"UintField":   Worse,
			},
		},
		{
			name: "nested structs",
			left: NestedStruct{
				Inner: TestStruct{
					IntField:    10,
					StringField: "test",
					FloatField:  1.5,
					UintField:   100,
				},
				Name: "left",
			},
			right: NestedStruct{
				Inner: TestStruct{
					IntField:    10,
					StringField: "test",
					FloatField:  1.5,
					UintField:   100,
				},
				Name: "right",
			},
			expected: ComplexComparison{
				"Inner": Equal,
				"Name":  None,
			},
		},
		{
			name: "uncomparable types",
			left: UncomparableStruct{
				Field1: complex(1, 2),
				Field2: []int{1, 2, 3},
				Field3: map[string]int{"a": 1},
			},
			right: UncomparableStruct{
				Field1: complex(3, 4),
				Field2: []int{4, 5, 6},
				Field3: map[string]int{"b": 2},
			},
			expected: ComplexComparison{
				"Field1": None,
				"Field2": None,
				"Field3": None,
			},
		},
		{
			name: "pointer types",
			left: &TestStruct{
				IntField:    10,
				StringField: "test",
				FloatField:  1.5,
				UintField:   100,
			},
			right: &TestStruct{
				IntField:    20,
				StringField: "different",
				FloatField:  2.5,
				UintField:   200,
			},
			expected: ComplexComparison{
				"IntField":    Worse,
				"StringField": None,
				"FloatField":  Worse,
				"UintField":   Worse,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			actual := ComplexCompare(tt.left, tt.right)

			if len(actual) != len(tt.expected) {
				t.Errorf("expected %d fields, got %d", len(tt.expected), len(actual))
			}

			for field, expectedComp := range tt.expected {
				actualComp, exists := actual[field]
				if !exists {
					t.Errorf("field %s missing from result", field)
					continue
				}
				if actualComp != expectedComp {
					t.Errorf("field %s: expected %s, got %s", field, expectedComp, actualComp)
				}
			}
		})
	}
}

// NestedStruct is used for testing nested struct comparison
type NestedStruct struct {
	Inner TestStruct
	Name  string
}

type UncomparableStruct struct {
	Field1 complex128     // Complex numbers aren't directly comparable
	Field2 []int          // Slices aren't directly comparable
	Field3 map[string]int // Maps aren't directly comparable
}

// Compare implements the Comparable interface for NestedStruct
func (ns *NestedStruct) Compare(other NestedStruct) (Comparison, error) {
	innerComp, _ := ns.Inner.Compare(other.Inner)
	nameComp := LiteralComparator(ns.Name, other.Name)

	if innerComp == Equal && nameComp == Equal {
		return Equal, nil
	}
	return None, nil
}
