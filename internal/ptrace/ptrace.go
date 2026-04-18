package ptrace

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

// New creates a new Tracer.
//
// Warning: Only call this function if you know what you are doing.
//
// Callers should access the Tracer using the TracerThread to ensure
// that per-process ptrace calls originate from the same thread.
//
// Failing to use a dedicated OS thread for ptrace calls will likely
// result in the ptrace call failing with errors like ECHILD (no
// child process).
func New(pid int) *Tracer {
	return &Tracer{
		pid: pid,
	}
}

// Tracer provides a simple API for consuming ptrace functionality for
// a specific process.
//
// Callers should access the Tracer using the TracerThread to ensure
// that per-process ptrace calls originate from the same thread.
//
// Failing to use a dedicated OS thread for ptrace calls will likely
// result in the ptrace call failing with errors like ECHILD (no
// child process).
type Tracer struct {
	pid     int
	stopped bool
}

func (o *Tracer) AttachAndWaitStopped() error {
	err := o.Attach()
	if err != nil {
		return fmt.Errorf("attach failed - %w", err)
	}

	_, _, err = o.WaitStopped()
	if err != nil {
		o.stopped = false

		return fmt.Errorf("wait stopped failed - %w", err)
	}

	return nil
}

func (o *Tracer) Attach() error {
	err := unix.PtraceAttach(o.pid)
	if err != nil {
		return err
	}

	o.stopped = true

	return nil
}

func (o *Tracer) Stopped() bool {
	return o.stopped
}

func (o *Tracer) Stop() error {
	err := o.Signal(syscall.SIGSTOP)
	if err != nil {
		return fmt.Errorf("failed to signal stop - %w", err)
	}

	_, _, err = o.WaitStopped()
	if err != nil {
		return fmt.Errorf("failed to wait for process to stop - %w", err)
	}

	o.stopped = true

	return nil
}

func (o *Tracer) Signal(sig syscall.Signal) error {
	return syscall.Kill(o.pid, sig)
}

func (o *Tracer) Wait(options int) (unix.WaitStatus, unix.Rusage, error) {
	var wstatus unix.WaitStatus
	var rusage unix.Rusage

	_, err := unix.Wait4(o.pid, &wstatus, options, &rusage)
	if err != nil {
		return 0, unix.Rusage{}, fmt.Errorf("wait4 failed (options: %v) - %w", options, err)
	}

	return wstatus, rusage, nil
}

func (o *Tracer) WaitStopped() (unix.WaitStatus, unix.Rusage, error) {
	// See "ptrace_<OS>.go" for implementation.
	ws, ru, err := o.waitStoppedOs()
	if err != nil {
		return ws, ru, err
	}

	o.stopped = true

	return ws, ru, err
}

func (o *Tracer) PeekData(addr uintptr, out []byte) (int, error) {
	resume := false

	if !o.stopped {
		err := o.AttachAndWaitStopped()
		if err != nil {
			return 0, err
		}

		resume = true
	}

	// See "ptrace_<OS>.go" for implementation.
	n, err := o.peekDataOs(addr, out)

	if resume {
		_ = o.Detach()
	}

	return n, err
}

func (o *Tracer) PokeData(addr uintptr, data []byte) (int, error) {
	resume := false

	if !o.stopped {
		resume = true

		err := o.AttachAndWaitStopped()
		if err != nil {
			return 0, err
		}
	}

	// See "ptrace_<OS>.go" for implementation.
	n, err := o.pokeDataOs(addr, data)

	if resume {
		_ = o.Detach()
	}

	return n, err
}

// This function is based on code from golang.org/x/sys/unix
// zsyscall_freebsd_arm64.go
func (o *Tracer) RequestPtr(request int, addr unsafe.Pointer, data int) error {
	return o.Request(request, uintptr(addr), data)
}

// This function is based on code from golang.org/x/sys/unix
// zsyscall_freebsd_arm64.go
func (o *Tracer) Request(request int, addr uintptr, data int) error {
	resume := false

	if !o.stopped {
		err := o.AttachAndWaitStopped()
		if err != nil {
			return err
		}

		resume = true
	}

	_, _, e1 := unix.Syscall6(
		unix.SYS_PTRACE,
		uintptr(request),
		uintptr(o.pid),
		uintptr(addr),
		uintptr(data),
		0,
		0)

	if resume {
		_ = o.Detach()
	}

	if e1 != 0 {
		return syscall.Errno(e1)
	}

	return nil
}

func (o *Tracer) Cont() error {
	o.stopped = false

	return unix.PtraceCont(o.pid, 0)
}

func (o *Tracer) ContSignal(sig syscall.Signal) error {
	return unix.PtraceCont(o.pid, int(sig))
}

func (o *Tracer) Detach() error {
	// TODO: on linux the process needs to be stopped according to
	// the PTRACE_DETACH section in the linux manual page unsure
	// what other unix-like operating systems require.
	if !o.stopped {
		_ = o.AttachAndWaitStopped()
	}

	o.stopped = false

	return unix.PtraceDetach(o.pid)
}

func (o *Tracer) SingleStep() error {
	return unix.PtraceSingleStep(o.pid)
}
