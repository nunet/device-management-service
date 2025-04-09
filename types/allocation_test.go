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

func TestEnsembleIDFromAllocationID(t *testing.T) {
	id := "ensemble123_allocation456"
	expected := "ensemble123"

	result := EnsembleIDFromAllocationID(id)
	if result != expected {
		t.Errorf("EnsembleIDFromAllocationID(%q) = %q, want %q", id, result, expected)
	}
}
