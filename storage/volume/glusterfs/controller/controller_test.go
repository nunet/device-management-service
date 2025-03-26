package controller

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/lib/sys"
)

const pemData = `-----BEGIN CERTIFICATE-----
MIIDBzCCAe+gAwIBAgIUUv6ChJPjUGuZgha3FVuBAwVsJ2swDQYJKoZIhvcNAQEL
BQAwEzERMA8GA1UEAwwIY2xpZW50MDEwHhcNMjUwMzIxMTE0NzAzWhcNMjUwNDIw
MTE0NzAzWjATMREwDwYDVQQDDAhjbGllbnQwMTCCASIwDQYJKoZIhvcNAQEBBQAD
ggEPADCCAQoCggEBAMrKH6rORCn+keE3xWRigo5emNR3dgy4sAUppagVRAeAlr24
GHzoE88/CThmAB/+jo9BTa5q9KYFB7XzjIFmgWKbCRK2nYYlIHBq0G1CPMVuMb3S
19TMTgksCfXlBS2SNhBTOJMbBpQPmOQO0zFWqI1G5Fsnlnmz09nN01Y6JEW86OMt
vUkjsbLrkFY9fZFmOSsGmG71UH+oFz4I0axy26uToG3ofQTHdmyK9NGnWHo9Gevk
FKCLO39IK+XHB57Q9eHgH4rqtCHdwUkwNb1if6sYl8Zpq2e8WfPmfsYvJ/QJTj3e
posDMCOh5nYsaM+a6+Wm1CyYz6osWt/NsGAPduUCAwEAAaNTMFEwHQYDVR0OBBYE
FC4Ec9NLWbt7UMvRZqxQE+jNaPVrMB8GA1UdIwQYMBaAFC4Ec9NLWbt7UMvRZqxQ
E+jNaPVrMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEBACsUhg8e
nLlg6VjsPusiSFQVkvfgkaOFnHDlXy1srNcLkjAgDCfWN/UWUC16Gajo6R/86nKq
UlVkYOvjWCnbXTljTeCK/S9UJp/HjzyqyQa6RB8g5mh9BKVNUkiuqABS8X9UuxVP
Fjsc8HDShZOG9e4V12T2R8lAFZkVKt0IAye2D1wY/Zu5iCvIwjeGPstkX5b6Sshg
jhXHPS0IVfFENiF5P3HUzUa+lj5ekINjp18EjCFuG9JDuue97DgK9ibvaokbMsY9
dGRATsA89JBR8SKXX4iW3XX+UpV1TpPZQBpdU2sBV6+SWGP1VBF4DgWhq/IinpN6
1l2b8kBso2JR/Jg=
-----END CERTIFICATE-----`

type stubCommander struct {
	output []byte
	err    error
}

func (s *stubCommander) CombinedOutput() ([]byte, error) {
	return s.output, s.err
}

func fakeExecCommand(name string, args ...string) sys.Commander {
	joinedArgs := strings.Join(args, " ")
	if name == "gluster" {
		if strings.Contains(joinedArgs, "fail_create") {
			return &stubCommander{
				output: []byte("simulated create failure"),
				err:    errors.New("create volume error"),
			}
		}
		if strings.Contains(joinedArgs, "fail_start") {
			return &stubCommander{
				output: []byte("simulated start failure"),
				err:    errors.New("start volume error"),
			}
		}
		if strings.Contains(joinedArgs, "fail_stop") {
			return &stubCommander{
				output: []byte("simulated stop failure"),
				err:    errors.New("stop volume error"),
			}
		}
		if strings.Contains(joinedArgs, "fail_delete") {
			return &stubCommander{
				output: []byte("simulated delete failure"),
				err:    errors.New("delete volume error"),
			}
		}
		return &stubCommander{
			output: []byte("successful"),
			err:    nil,
		}
	}

	return &stubCommander{
		output: []byte("default output"),
		err:    nil,
	}
}

func TestCreateVolumeSuccess(t *testing.T) {
	origExecCommand := sys.ExecCommand
	defer func() { sys.ExecCommand = origExecCommand }()
	sys.ExecCommand = fakeExecCommand

	bricks := []string{"host1:/brick1", "host2:/brick2"}

	tmpPath := filepath.Join(getCurrentFileDirectory(), "testdata")

	gc, _ := NewGlusterController(tmpPath, bricks, 2)
	_, err := gc.CreateVolume("myvolume", pemData)
	require.NoError(t, err)
}

func TestCreateVolumeFailure(t *testing.T) {
	origExecCommand := sys.ExecCommand
	defer func() { sys.ExecCommand = origExecCommand }()
	sys.ExecCommand = fakeExecCommand

	tmpPath := filepath.Join(getCurrentFileDirectory(), "testdata")

	gc, _ := NewGlusterController(tmpPath, []string{}, 0)
	_, err := gc.CreateVolume("fail_create", pemData)
	require.Error(t, err)
	require.Contains(t, err.Error(), "create volume error")
}

func TestStartVolumeSuccess(t *testing.T) {
	origExecCommand := sys.ExecCommand
	defer func() { sys.ExecCommand = origExecCommand }()
	sys.ExecCommand = fakeExecCommand

	bricks := []string{"host1:/brick1", "host2:/brick2"}

	tmpPath := filepath.Join(getCurrentFileDirectory(), "testdata")

	gc, _ := NewGlusterController(tmpPath, bricks, 0)
	err := gc.StartVolume("myvolume")
	require.NoError(t, err)
}

func TestStartVolumeFailure(t *testing.T) {
	origExecCommand := sys.ExecCommand
	defer func() { sys.ExecCommand = origExecCommand }()
	sys.ExecCommand = fakeExecCommand

	tmpPath := filepath.Join(getCurrentFileDirectory(), "testdata")

	gc, _ := NewGlusterController(tmpPath, []string{}, 0)
	err := gc.StartVolume("fail_start")
	require.Error(t, err)
	require.Contains(t, err.Error(), "start volume error")
}

func TestDeleteVolumeSuccess(t *testing.T) {
	origExecCommand := sys.ExecCommand
	defer func() { sys.ExecCommand = origExecCommand }()
	sys.ExecCommand = fakeExecCommand

	tmpPath := filepath.Join(getCurrentFileDirectory(), "testdata")

	gc, _ := NewGlusterController(tmpPath, []string{}, 0)
	err := gc.DeleteVolume("myvolume")
	require.NoError(t, err)
}

func TestDeleteVolumeStopFailure(t *testing.T) {
	origExecCommand := sys.ExecCommand
	defer func() { sys.ExecCommand = origExecCommand }()
	sys.ExecCommand = fakeExecCommand

	tmpPath := filepath.Join(getCurrentFileDirectory(), "testdata")

	gc, _ := NewGlusterController(tmpPath, []string{}, 0)
	err := gc.DeleteVolume("fail_stop")
	require.Error(t, err)
	require.Contains(t, err.Error(), "stop volume error")
}

func TestDeleteVolumeDeleteFailure(t *testing.T) {
	origExecCommand := sys.ExecCommand
	defer func() { sys.ExecCommand = origExecCommand }()
	sys.ExecCommand = fakeExecCommand

	tmpPath := filepath.Join(getCurrentFileDirectory(), "testdata")

	gc, _ := NewGlusterController(tmpPath, []string{}, 0)
	err := gc.DeleteVolume("fail_delete")
	require.Error(t, err)
	require.Contains(t, err.Error(), "delete volume error")
}

func TestValidatePEM(t *testing.T) {
	err := validatePEM([]byte(pemData))
	require.NoError(t, err)
}

func getCurrentFileDirectory() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		panic("Unable to get current file info")
	}
	return filepath.Dir(file)
}
