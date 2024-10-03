package gpu

import (
	"testing"
)

// TODO: we need to mock the gpu details so that the library can pick it up
// https://gitlab.com/nunet/device-management-service/-/issues/534
func TestGetAMDGPUInfo(t *testing.T) {
	t.Skipf("Skipping test as it requires a real AMD GPU")
}

func TestGetIntelGPUInfo(t *testing.T) {
	t.Skipf("Skipping test as it requires a real Intel GPU")
}

func TestGetNvidiaGPUInfo(t *testing.T) {
	t.Skipf("Skipping test as it requires a real Nvidia GPU")
}

func TestGetGPUs(t *testing.T) {
	t.Skipf("Skipping test as it requires a real GPU")
}
