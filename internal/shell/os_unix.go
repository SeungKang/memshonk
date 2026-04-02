//go:build !windows

package shell

import (
	"golang.org/x/sys/unix"
)

func hasPermissionToDir(path string) error {
	err := unix.Access(path, unix.X_OK)
	if err != nil {
		return err
	}

	return nil
}
