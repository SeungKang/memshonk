package sessiond

import (
	"syscall"
)

func DaemonSysProcAttr() *syscall.SysProcAttr {
	return nil
}
