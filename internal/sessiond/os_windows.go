package sessiond

import (
	"syscall"
)

func DaemonSysProcAttr() *syscall.SysProcAttr {
	// CREATE_NEW_PROCESS_GROUP prevents console signals (e.g. Ctrl+C)
	// sent to the client from propagating to the daemon.
	//
	// CREATE_NEW_CONSOLE detaches the daemon from the parent's console
	// session. This is necessary on Windows 11 where Windows Terminal
	// uses job objects and without a new console, the daemon gets killed
	// when the client exits. Creating a new console dissociates the
	// daemon from that job object.
	//
	// HideWindow ensures the daemon's console window is not visible
	// across different terminal emulators and Windows versions.
	return &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP | 0x00000010, // CREATE_NEW_CONSOLE
		HideWindow:    true,
	}
}
