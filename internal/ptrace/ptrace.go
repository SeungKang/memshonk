package ptrace

import (
	"golang.org/x/sys/unix"
)

func Attach(pid int) (*Tracer, error) {
	err := unix.PtraceAttach(pid)
	if err != nil {
		return nil, err
	}

	return &Tracer{
		pid: pid,
	}, nil
}

// Note: This code is based on work by Mahmud "hjr265" Ridwan:
// https://github.com/hjr265/ptrace.go
type Tracer struct {
	pid int
}

func (t *Tracer) Cont(sig unix.Signal) error {
	return unix.PtraceCont(t.pid, int(sig))
}

func (t *Tracer) Detach() error {
	return unix.PtraceDetach(t.pid)
}

func (t *Tracer) Registers() (*unix.Reg, error) {
	regs := &unix.Reg{}

	err := unix.PtraceGetRegs(t.pid, regs)
	if err != nil {
		return nil, err
	}

	return regs, nil
}

func (t *Tracer) PeekData(addr uintptr, out []byte) (int, error) {
	return unix.PtracePeekData(t.pid, addr, out)
}

func (t *Tracer) PeekText(addr uintptr, out []byte) (int, error) {
	return unix.PtracePeekText(t.pid, addr, out)
}

func (t *Tracer) PokeData(addr uintptr, data []byte) (int, error) {
	return unix.PtracePokeData(t.pid, addr, data)
}

func (t *Tracer) PokeText(addr uintptr, data []byte) (int, error) {
	return unix.PtracePokeText(t.pid, addr, data)
}

func (t *Tracer) SetRegs(regs *unix.Reg) error {
	return unix.PtraceSetRegs(t.pid, regs)
}

func (t *Tracer) SingleStep() error {
	return unix.PtraceSingleStep(t.pid)
}

// See also:
// https://www.lemoda.net/freebsd/ptrace/index.html
//
// func (t *Tracer) unix(sig unix.Signal) (uint64, error) {
// 	err := unix.PtraceSyscall(t.Process.Pid, int(sig))
// 	if err != nil {
// 		return 0, err
// 	}

// 	status := unix.WaitStatus(0)
// 	_, err = unix.Wait4(t.Process.Pid, &status, 0, nil)
// 	if err != nil {
// 		return 0, err
// 	}

// 	regs, err := t.GetRegs()
// 	if err != nil {
// 		return 0, err
// 	}

// 	return regs.Orig_rax, nil
// }
