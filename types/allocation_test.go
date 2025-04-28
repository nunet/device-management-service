package types

import (
	"testing"
)

func TestAllocationNameFromID(t *testing.T) {
	id := "ensembleABC_allocation456"
	expected := "allocation456"

	result := AllocationNameFromID(id)
	if result != expected {
		t.Errorf("AllocationNameFromID(%q) = %q, want %q", id, result, expected)
	}
}

func TestEnsembleIDFromAllocationID(t *testing.T) {
	id := "ensemble123_allocation912"
	expected := "ensemble123"

	result := EnsembleIDFromAllocationID(id)
	if result != expected {
		t.Errorf("EnsembleIDFromAllocationID(%q) = %q, want %q", id, result, expected)
	}
}

func TestConstructAllocationID(t *testing.T) {
	ensembleID := "ensembleXYZ"
	allocName := "allocation456"
	expected := "ensembleXYZ_allocation456"

	result := ConstructAllocationID(ensembleID, allocName)
	if result != expected {
		t.Errorf("ConstructAllocationID(%q, %q) = %q, want %q", ensembleID, allocName, result, expected)
	}
}
