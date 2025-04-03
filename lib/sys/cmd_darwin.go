package sys

import (
	"os/exec"
	"syscall"
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
	cmd.SysProcAttr = &syscall.SysProcAttr{}
	return cmd
}
