package hardware

import "testing"

func TestGetMachineResources(t *testing.T) {
	t.Parallel()

	_, err := GetMachineResources()
	if err != nil {
		t.Errorf("failed to get machine resources: %s", err)
	}
}
