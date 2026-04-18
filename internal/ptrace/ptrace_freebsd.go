//go:build freebsd

package ptrace

import (
	"syscall"

	"golang.org/x/sys/unix"
)

// Note: unix.PtracePeeek(...) and PtracePoke(...) are limited
// to 4 or 8 bytes on FreeBSD. These limitations likely stem
// from the "normal" limits of ptrace working in 4 or 8 bytes.
// FreeBSD seems to have added their own API (PT_IO) for
// reading and writing data of arbitrary lengths.
//
// Unforuntately, the Go APIs enforce the original 4 or 8 byte
// limits when calling the FreeBSD-specific PT_IO API. Thus,
// we can just skip straight to Go's unix.PtraceIO wrapper
// and specify our own sizes.

func (o *Tracer) waitStoppedOs() (unix.WaitStatus, unix.Rusage, error) {
	return o.Wait(syscall.WSTOPPED)
}

func (o *Tracer) peekDataOs(addr uintptr, out []byte) (int, error) {
	return unix.PtraceIO(unix.PIOD_READ_D, o.pid, addr, out, len(out))
}

func (o *Tracer) pokeDataOs(addr uintptr, data []byte) (int, error) {
	return unix.PtraceIO(unix.PIOD_WRITE_D, o.pid, addr, data, len(data))
}
