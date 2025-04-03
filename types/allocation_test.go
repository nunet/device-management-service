package types

import (
	"testing"
)

func TestAllocationNameFromID(t *testing.T) {
	id := "ensemble123_allocation456"
	expected := "allocation456"

	result := AllocationNameFromID(id)
	if result != expected {
		t.Errorf("AllocationNameFromID(%q) = %q, want %q", id, result, expected)
	}
}
