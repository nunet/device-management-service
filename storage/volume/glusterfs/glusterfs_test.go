// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

//go:build linux
// +build linux

package glusterfs

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gitlab.com/nunet/device-management-service/storage"
)

type stubCommander struct {
	output []byte
	err    error
}

const mountPath = "/mnt/glusterfs"

const volName = "myvolume"

func (f *stubCommander) CombinedOutput() ([]byte, error) {
	return f.output, f.err
}

func fakeExecCommand(name string, args ...string) Commander {
	if name == "mount" {
		joinedArgs := strings.Join(args, " ")
		if strings.Contains(joinedArgs, "fail_mount") {
			return &stubCommander{
				output: []byte("simulated mount failure"),
				err:    errors.New("mount command error"),
			}
		}
		return &stubCommander{
			output: []byte("mount successful"),
			err:    nil,
		}
	} else if name == "umount" {
		joinedArgs := strings.Join(args, " ")
		if strings.Contains(joinedArgs, "fail_umount") {
			return &stubCommander{
				output: []byte("simulated unmount failure"),
				err:    errors.New("umount command error"),
			}
		}
		return &stubCommander{
			output: []byte("umount successful"),
			err:    nil,
		}
	}
	return &stubCommander{
		output: []byte("default output"),
		err:    nil,
	}
}

func TestMountSuccess(t *testing.T) {
	vt := storage.NewVolumeTracker()
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	execCommand = fakeExecCommand
	servers := []string{"10.0.0.1", "10.0.0.2"}
	g, err := New(vt, servers, volName, "", "", "")
	require.NoError(t, err)

	options := map[string]string{
		"option1": "value1",
	}

	err = g.Mount(mountPath, options)
	require.NoError(t, err)
}

func TestMountFailure(t *testing.T) {
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()
	execCommand = fakeExecCommand
	vt := storage.NewVolumeTracker()

	servers := []string{"10.0.0.1"}
	g, err := New(vt, servers, volName, "", "", "")
	require.NoError(t, err)

	options := map[string]string{
		"fail_mount": "true",
	}

	err = g.Mount(mountPath, options)
	if err == nil {
		t.Error("expected Mount() to fail, but it succeeded")
	} else {
		expectedSubstring := "mount command error"
		if !strings.Contains(err.Error(), expectedSubstring) {
			t.Errorf("expected error to contain %q, got %q", expectedSubstring, err.Error())
		}
	}
}

func TestUnmountSuccess(t *testing.T) {
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()
	execCommand = fakeExecCommand
	vt := storage.NewVolumeTracker()

	servers := []string{"10.0.0.1"}
	g, err := New(vt, servers, volName, "", "", "")
	require.NoError(t, err)
	err = g.Unmount(mountPath)
	require.EqualError(t, err, "/mnt/glusterfs is not mounted")

	err = g.Mount(mountPath, map[string]string{})
	require.NoError(t, err)
	err = g.Unmount(mountPath)
	require.NoError(t, err)
}

func TestUnmountFailure(t *testing.T) {
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()
	execCommand = fakeExecCommand
	vt := storage.NewVolumeTracker()
	servers := []string{"10.0.0.1"}
	g, err := New(vt, servers, volName, "", "", "")
	require.NoError(t, err)

	targetPath := "/mnt/fail_umount"
	err = g.Mount(targetPath, nil)
	require.NoError(t, err)

	err = g.Unmount(targetPath)
	if err == nil {
		t.Error("expected Unmount() to fail, but it succeeded")
	} else {
		expectedSubstring := "umount command error"
		if !strings.Contains(err.Error(), expectedSubstring) {
			t.Errorf("expected error to contain %q, got %q", expectedSubstring, err.Error())
		}
	}
}

func TestNewValidation(t *testing.T) {
	vt := storage.NewVolumeTracker()
	_, err := New(vt, []string{}, "volume", "", "", "")
	if err == nil {
		t.Error("expected error when no servers provided, got nil")
	}

	_, err = New(vt, []string{"10.0.0.1"}, "", "", "", "")
	if err == nil {
		t.Error("expected error when no volume provided, got nil")
	}
}

func TestMountCommandArgs(t *testing.T) {
	origExecCommand := execCommand
	defer func() { execCommand = origExecCommand }()

	var capturedName string
	var capturedArgs []string
	execCommand = func(name string, args ...string) Commander {
		capturedName = name
		capturedArgs = args
		return &stubCommander{
			output: []byte("ok"),
			err:    nil,
		}
	}

	servers := []string{"10.0.0.1", "10.0.0.2"}
	volume := "myvolume"
	sslCert := "/path/to/cert.pem"
	sslKey := "/path/to/key.pem"
	sslCA := "/path/to/ca.pem"
	vt := storage.NewVolumeTracker()
	g, err := New(vt, servers, volume, sslCert, sslKey, sslCA)
	require.NoError(t, err)

	options := map[string]string{
		"opt1": "val1",
	}

	err = g.Mount(mountPath, options)
	require.NoError(t, err)

	if capturedName != "mount" {
		t.Errorf("expected command name 'mount', got %q", capturedName)
	}

	expectedSource := fmt.Sprintf("%s:%s", strings.Join(servers, ","), volume)
	expectedOpts := make([]string, 0)
	for k, v := range options {
		expectedOpts = append(expectedOpts, fmt.Sprintf("%s=%s", k, v))
	}

	expectedOpts = append(expectedOpts, fmt.Sprintf("ssl_cert=%s", sslCert))
	expectedOpts = append(expectedOpts, fmt.Sprintf("ssl_key=%s", sslKey))
	expectedOpts = append(expectedOpts, fmt.Sprintf("ssl_ca=%s", sslCA))

	// expect the following structure:
	// ["-t", "glusterfs", "-o", "<opts>", "<source>", "<targetPath>"]
	expectedArgs := []string{"-t", "glusterfs", "-o", strings.Join(expectedOpts, ","), expectedSource, mountPath}

	if !reflect.DeepEqual(capturedArgs, expectedArgs) {
		t.Errorf("expected args:\n%v\ngot:\n%v", expectedArgs, capturedArgs)
	}
}
