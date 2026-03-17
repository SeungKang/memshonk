//go:build linux

package progctl

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/SeungKang/memshonk/internal/linuxmaps"
	"github.com/SeungKang/memshonk/internal/memory"

	"golang.org/x/sys/unix"
)

func notifyOnProcExit(pid int, exitMon *ExitMonitor) error {
	pidfd, err := unix.PidfdOpen(pid, 0)
	switch {
	case err == nil:
		return notifyPidfdExited(pidfd, exitMon)
	default:
		// If we are on an older version of Linux (< 5.3)
		// or if pidfdopen fails, fallback to using procfs.
		// If needed, we can test if pidfd is not supported
		// using:
		//   errors.Is(err, unix.ENOSYS)
		return notifyProcfsEntryGone(pid, exitMon)
	}
}

func notifyPidfdExited(pidFd int, exitMon *ExitMonitor) error {
	eventFd, err := unix.Eventfd(0, unix.EFD_NONBLOCK)
	if err != nil {
		_ = unix.Close(pidFd)

		return err
	}

	go func() {
		<-exitMon.Done()

		// This somehow cancels the poll call
		// according to claude.
		_, _ = unix.Write(eventFd, []byte{1, 0, 0, 0, 0, 0, 0, 0})
	}()

	go func() {
		defer unix.Close(pidFd)
		defer unix.Close(eventFd)

		fds := []unix.PollFd{
			{Fd: int32(pidFd), Events: unix.POLLIN},
			{Fd: int32(eventFd), Events: unix.POLLIN},
		}

		for {
			n, err := unix.Poll(fds, -1)
			switch {
			case err == nil:
				// Keep going.
			case errors.Is(err, unix.EINTR):
				continue
			default:
				exitMon.SetExited(&ExitMonitorProcExitErr{
					Source:        "pidfd-poll",
					OptMonitorErr: fmt.Errorf("failed to poll proc fds - %w", err),
				})

				return
			}

			if n > 0 {
				if fds[1].Revents&unix.POLLIN != 0 {
					// The other go routine
					// cancelled the poll.
					return
				}

				exitMon.SetExited(&ExitMonitorProcExitErr{
					Source: "pidfd-poll",
				})

				return
			}

		}
	}()

	return nil
}

func notifyProcfsEntryGone(pid int, exitMon *ExitMonitor) error {
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		procFsPath := fmt.Sprintf("/proc/%d", pid)

		for {
			select {
			case <-exitMon.Done():
				return
			case <-ticker.C:
				_, err := os.Stat(procFsPath)
				if err != nil {
					exitMon.SetExited(&ExitMonitorProcExitErr{
						Source: "procfs-checker",
					})

					return
				}
			}
		}
	}()

	return nil
}

func (o *process) Regions() (memory.Regions, error) {
	return linuxmaps.Vmmap(o.pid)
}
