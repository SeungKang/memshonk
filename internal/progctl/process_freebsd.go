//go:build freebsd

package progctl

import (
	"context"
	"fmt"
	"os"

	"github.com/SeungKang/memshonk/internal/fbsdmaps"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/ptrace"

	"golang.org/x/sys/unix"
)

func notifyOnProcExit(pid int, exitMon *ExitMonitor) error {
	kq, err := unix.Kqueue()
	if err != nil {
		return fmt.Errorf("failed to open kqueue - %w", err)
	}

	// We need a way to cancel the kqueue call below which
	// blocks until the specified PID's process exits.
	// There are (at least) two ways to do this. We opted
	// to use a pipe  because the other option (EVFILT_USER)
	// requires FreeBSD 8.1 (released in 2009ish timeframe).
	// We prefer using APIs that provide wider compatibility,
	// so we went with the pipe cancellation strategy.
	//
	// If we want to use the EVFILT_USER approach, here is
	// what that code may look like:
	//
	//   kq, _ := unix.Kqueue()
	//
	//   const wakeupIdent = 1
	//
	//   events := []unix.Kevent_t{
	//     {
	//         Ident:  uint64(pid),
	//         Filter: unix.EVFILT_PROC,
	//         Flags:  unix.EV_ADD | unix.EV_ONESHOT,
	//         Fflags: unix.NOTE_EXIT,
	//     },
	//     {
	//         Ident:  wakeupIdent,
	//         Filter: unix.EVFILT_USER,
	//         Flags:  unix.EV_ADD | unix.EV_CLEAR,
	//      },
	//   }
	//
	//   unix.Kevent(kq, events, nil, nil)
	//
	//   //
	//   // To trigger from another goroutine:
	//   //
	//
	//   trigger := []unix.Kevent_t{{
	//        Ident:  wakeupIdent,
	//        Filter: unix.EVFILT_USER,
	//        Fflags: unix.NOTE_TRIGGER,
	//   }}
	//
	//   unix.Kevent(kq, trigger, nil, nil)
	r, w, err := os.Pipe()
	if err != nil {
		return fmt.Errorf("failed to create pipe for waking up kqueue - %w", err)
	}

	go func() {
		<-exitMon.Done()

		w.Write([]byte{0x00})
	}()

	go func() {
		changes := []unix.Kevent_t{
			{
				Ident:  uint64(pid),
				Filter: unix.EVFILT_PROC,
				Flags:  unix.EV_ADD | unix.EV_ONESHOT,
				Fflags: unix.NOTE_EXIT,
			},
			{
				Ident:  uint64(r.Fd()),
				Filter: unix.EVFILT_READ,
				Flags:  unix.EV_ADD,
			},
		}

		events := make([]unix.Kevent_t, 2)

		// This blocks until the process exits or
		// the pipe is written to.
		n, err := unix.Kevent(kq, changes, events, nil)

		if n > 0 && events[0].Fflags&unix.NOTE_EXIT != 0 {
			exitStatus := events[0].Data

			exitMon.SetExited(&ExitMonitorProcExitErr{
				Source:        "kqueue",
				OptExitStatus: &exitStatus,
			})
		} else {
			exitMon.SetExited(&ExitMonitorProcExitErr{
				Source:        "kqueue",
				OptMonitorErr: err,
			})
		}

		r.Close()
		w.Close()
	}()

	return nil
}

func (o *process) Regions() (memory.Regions, error) {
	if o.optPtrace == nil {
		return memory.Regions{}, errPtraceNotEnabled
	}

	needToResume := false

	if !o.stopped {
		needToResume = true

		err := o.Suspend()
		if err != nil {
			return memory.Regions{}, fmt.Errorf("failed to suspend process prior to getting regions - %w", err)
		}
	}

	var regions memory.Regions

	regionsErr := o.optPtrace.Do(context.Background(), func(_ context.Context, pt *ptrace.Tracer) error {
		var err error
		regions, err = fbsdmaps.Vmmap(pt)
		return err
	})

	if needToResume {
		err := o.Resume()
		if err != nil {
			return memory.Regions{}, fmt.Errorf("failed to resume process after getting regions - %w",
				err)
		}
	}

	return regions, regionsErr
}
