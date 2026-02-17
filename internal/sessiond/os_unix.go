//go:build !windows

package sessiond

import (
	"syscall"
)

func DaemonSysProcAttr() *syscall.SysProcAttr {
	return &syscall.SysProcAttr{
		// Setting Setpgid to true ensures that the
		// child process does not receive signals sent
		// to the parent process. Those signals would
		// normally propagate to the child because the
		// parent and child processes default to being
		// in the same process group.
		//
		// This prevents the process from being killed
		// when the user exits their SSH session.
		Setpgid: true,
	}
}
