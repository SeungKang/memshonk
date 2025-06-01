//go:build linux

package ptrace

import "golang.org/x/sys/unix"

// Note: Normally PtracePeek(...) and PtracePoke(...) are limited
// to 4 or 8 bytes by default. Luckily, the Go maintainers implemented
// wrappers *specifically* for their Linux implementations that deal
// with this limitiation for us.

func (o *Tracer) PeekData(addr uintptr, out []byte) (int, error) {
	return unix.PtracePeekData(o.pid, addr, out)
}

func (o *Tracer) PokeData(addr uintptr, data []byte) (int, error) {
	return unix.PtracePokeData(o.pid, addr, data)
}
