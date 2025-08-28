// Copyright 2024, Nunet
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and limitations under the License.

package sys

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/unix"
)

// Commander interface represents a command that can be executed.
type Commander interface {
	CombinedOutput() ([]byte, error)
}

// ExecFunc defines the function signature for command execution.
type ExecFunc func(name string, args ...string) Commander

// ExecCommand is the function used to run external commands.
// It defaults to exec.Command with added capabilities.
var ExecCommand ExecFunc = func(name string, args ...string) Commander {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		AmbientCaps: []uintptr{
			unix.CAP_SYS_ADMIN,
		},
	}
	return cmd
}
