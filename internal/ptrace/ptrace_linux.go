//go:build linux

package ptrace

import "golang.org/x/sys/unix"

// Note: Normally PtracePeek(...) and PtracePoke(...) are limited
// to 4 or 8 bytes by default. Luckily, the Go maintainers implemented
// wrappers *specifically* for their Linux implementations that deal
// with this limitiation for us.

func (o *Tracer) waitStoppedOs() (unix.WaitStatus, unix.Rusage, error) {
	// From the Linux ptrace manual:
	//
	//   When the running tracee enters ptrace-stop, it notifies
	//   its tracer using waitpid(2) (or one of the other "wait"
	//   system calls). Most of this manual page assumes that
	//   the tracer waits with:
	//
	//     pid = waitpid(pid_or_minus_1, &status, __WALL);
	//
	//   Ptrace-stopped tracees are reported as returns with
	//   pid greater than 0 and WIFSTOPPED(status) true.
	return o.Wait(unix.WALL)
}

func (o *Tracer) peekDataOs(addr uintptr, out []byte) (int, error) {
	return unix.PtracePeekData(o.pid, addr, out)
}

func (o *Tracer) pokeDataOs(addr uintptr, data []byte) (int, error) {
	return unix.PtracePokeData(o.pid, addr, data)
}
