package ptrace

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/unix"
)

func StopAndAttach(pid int) (*Tracer, error) {
	err := syscall.Kill(pid, syscall.SIGSTOP)
	if err != nil {
		return nil, err
	}

	return Attach(pid)
}

func Attach(pid int) (*Tracer, error) {
	err := unix.PtraceAttach(pid)
	if err != nil {
		return nil, err
	}

	t := &Tracer{
		pid: pid,
	}

	_, _, err = t.Wait()
	if err != nil {
		t.Detach()
		return nil, err
	}

	return t, nil
}

type Tracer struct {
	pid int
}

func (o *Tracer) SigstopAndSuspend() (unix.WaitStatus, unix.Rusage, error) {
	err := o.Signal(syscall.SIGSTOP)
	if err != nil {
		return 0, unix.Rusage{}, err
	}

	wstatus, rusage, err := o.Wait()
	if err != nil {
		return 0, unix.Rusage{}, err
	}

	err = o.Request(unix.PT_SUSPEND, 0, 0)
	if err != nil {
		return 0, unix.Rusage{}, err
	}

	return wstatus, rusage, nil
}

func (o *Tracer) Signal(sig syscall.Signal) error {
	return syscall.Kill(o.pid, sig)
}

func (o *Tracer) Wait() (unix.WaitStatus, unix.Rusage, error) {
	var wstatus unix.WaitStatus
	var rusage unix.Rusage

	_, err := unix.Wait4(o.pid, &wstatus, 0, &rusage)
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

func (o *Tracer) Registers() (*unix.Reg, error) {
	regs := &unix.Reg{}

	err := unix.PtraceGetRegs(o.pid, regs)
	if err != nil {
		return nil, err
	}

	return regs, nil
}

func (o *Tracer) PeekData(addr uintptr, out []byte) (int, error) {
	return unix.PtracePeekData(o.pid, addr, out)
}

func (o *Tracer) PeekText(addr uintptr, out []byte) (int, error) {
	return unix.PtracePeekText(o.pid, addr, out)
}

func (o *Tracer) PokeData(addr uintptr, data []byte) (int, error) {
	return unix.PtracePokeData(o.pid, addr, data)
}

func (o *Tracer) PokeText(addr uintptr, data []byte) (int, error) {
	return unix.PtracePokeText(o.pid, addr, data)
}

func (o *Tracer) SetRegs(regs *unix.Reg) error {
	return unix.PtraceSetRegs(o.pid, regs)
}

func (o *Tracer) SingleStep() error {
	return unix.PtraceSingleStep(o.pid)
}
