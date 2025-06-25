//go:build linux && (amd64 || 386) && cgo

package amdsmi

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type AMDSMITestSuite struct {
	suite.Suite

	sockets []SocketHandle
	procs   []ProcessorHandle
	gpu     ProcessorHandle
}

func (s *AMDSMITestSuite) SetupSuite() {
	initSt, err := Init()
	if err != nil || initSt.Code != StatusSuccess {
		s.T().Skipf(
			"SKIP: Machine possibly without an AMD gpu: unable to initialize AMD SMI (status=%v err=%v)",
			initSt, err)
	}

	sockets, soStatus := GetSocketHandles()
	s.Require().Equal(StatusSuccess, soStatus.Code, "GetSocketHandles must succeed")
	s.Require().NotEmpty(sockets, "At least one socket should exist")
	s.sockets = sockets

	procs, prStatus := GetProcessorHandles(sockets[0])
	s.Require().Equal(StatusSuccess, prStatus.Code, "GetProcessorHandles must succeed")
	s.Require().NotEmpty(procs, "Socket must have at least one processor")
	s.procs = procs

	// find the first GPU on that socket
	for _, p := range procs {
		pt, ptStatus := GetProcessorType(p)
		s.Assert().Equal(StatusSuccess, ptStatus.Code, "GetProcessorType must succeed")
		if pt == ProcessorTypeAMDGPU {
			s.gpu = p
			break
		}
	}
	if s.gpu.handle == nil {
		s.T().Skip("SKIP: no AMD GPU found")
	}
}

// TearDownSuite runs once after all tests
func (s *AMDSMITestSuite) TearDownSuite() {
	st := Shutdown()
	s.Assert().Equal(StatusSuccess, st.Code, "Shutdown should succeed")
}

// TestGetSocketName checks that we get a non‐empty name back
func (s *AMDSMITestSuite) TestGetSocketName() {
	name, st := GetSocketName(s.sockets[0], 64)
	s.Require().Equal(StatusSuccess, st.Code)
	s.Require().NotEmpty(name, "Socket name should not be empty")
}

// TestProcessorType ensures we can read back the processor type
func (s *AMDSMITestSuite) TestProcessorType() {
	pt, st := GetProcessorType(s.procs[0])
	s.Require().Equal(StatusSuccess, st.Code)
	// valid enum: UNKNOWN, AMD_GPU, AMD_CPU, etc.
	s.Assert().NotEqual(ProcessorTypeNonAMDGPU, pt) // simple sanity check
}

// TestGPUBoardInfo ensures board info struct is populated
func (s *AMDSMITestSuite) TestGPUBoardInfo() {
	bi, st := GetGPUBoardInfo(s.gpu)
	s.Require().Equal(StatusSuccess, st.Code)
	s.Require().NotEmpty(bi.ModelNumber, "ModelNumber should be set")
	s.Require().NotEmpty(bi.ProductName, "ProductName should be set")
	s.Require().NotEmpty(bi.ProductSerial, "ProductSerial should be set")
	s.Require().NotEmpty(bi.ManufacturerName, "ManufacturerName should be set")
}

// TestGPUID checks the GPU ID call
func (s *AMDSMITestSuite) TestGPUID() {
	id, st := GetGPUID(s.gpu)
	s.Require().Equal(StatusSuccess, st.Code)
	s.Assert().Greater(id, uint32(0), "GPU ID should be positive")
}

// TestGPUUUID checks that we get a valid UUID string
func (s *AMDSMITestSuite) TestGPUUUID() {
	uuid, st := GetGPUUUID(s.gpu)
	s.Require().Equal(StatusSuccess, st.Code)
	s.Assert().GreaterOrEqual(len(uuid), int(GPUUUIDSize)-1, "UUID must have expected length")
}

// TestGPUVRAM checks VRAM usage fields
func (s *AMDSMITestSuite) TestGPUVRAM() {
	vr, st := GetGPUVRAM(s.gpu)
	s.Require().Equal(StatusSuccess, st.Code)
	s.Assert().Greater(vr.Total, uint32(0), "VRAM.Total must be > 0")
	s.Assert().LessOrEqual(vr.Used, vr.Total, "VRAM.Used must not exceed Total")
}

func TestAMDSMITestSuite(t *testing.T) {
	suite.Run(t, new(AMDSMITestSuite))
}
