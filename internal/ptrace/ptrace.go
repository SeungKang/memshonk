package ptrace

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func New(pid int) *Tracer {
	return &Tracer{
		pid: pid,
	}
}

type Tracer struct {
	pid int
}

func (o *Tracer) StopAndAttach() error {
	err := unix.PtraceAttach(o.pid)
	if err != nil {
		return fmt.Errorf("ptrace attach failed - %w", err)
	}

	_, _, err = o.WaitStopped()
	if err != nil {
		return fmt.Errorf("failed to wait for process to stop after ptrace attach - %w", err)
	}

	return nil
}

func (o *Tracer) Signal(sig syscall.Signal) error {
	return syscall.Kill(o.pid, sig)
}

func (o *Tracer) WaitStopped() (unix.WaitStatus, unix.Rusage, error) {
	var wstatus unix.WaitStatus
	var rusage unix.Rusage

	_, err := unix.Wait4(o.pid, &wstatus, unix.WSTOPPED, &rusage)
	if err != nil {
		return 0, unix.Rusage{}, fmt.Errorf("wait4 failed - %w", err)
	}

	return wstatus, rusage, nil
}

// This function is based on code from golang.org/x/sys/unix
// zsyscall_freebsd_arm64.go
func (o *Tracer) RequestPtr(request int, addr unsafe.Pointer, data int) error {
	return o.Request(request, uintptr(addr), data)
}

// This function is based on code from golang.org/x/sys/unix
// zsyscall_freebsd_arm64.go
func (o *Tracer) Request(request int, addr uintptr, data int) error {
	_, _, e1 := unix.Syscall6(
		unix.SYS_PTRACE,
		uintptr(request),
		uintptr(o.pid),
		uintptr(addr),
		uintptr(data),
		0,
		0)
	if e1 != 0 {
		return syscall.Errno(e1)
	}

	return nil
}

func (o *Tracer) Cont() error {
	return unix.PtraceCont(o.pid, 0)
}

func (o *Tracer) ContSignal(sig syscall.Signal) error {
	return unix.PtraceCont(o.pid, int(sig))
}

func (o *Tracer) Detach() error {
	return unix.PtraceDetach(o.pid)
}

//func (o *Tracer) Registers() (*unix.Reg, error) {
//	regs := &unix.Reg{}
//
//	err := unix.PtraceGetRegs(o.pid, regs)
//	if err != nil {
//		return nil, err
//	}
//
//	return regs, nil
//}
//
//func (o *Tracer) SetRegs(regs *unix.Reg) error {
//	return unix.PtraceSetRegs(o.pid, regs)
//}

func (o *Tracer) SingleStep() error {
	return unix.PtraceSingleStep(o.pid)
}
