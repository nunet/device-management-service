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
