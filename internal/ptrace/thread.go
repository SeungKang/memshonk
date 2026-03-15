package ptrace

import (
	"context"
	"syscall"
	"unsafe"

	"github.com/SeungKang/memshonk/internal/threadlock"

	"golang.org/x/sys/unix"
)

// NewTracerThread creates a new, dedicated operating system thread
// to execute ptrace calls from.
//
// Callers should call TracerThread.Close when finished with the
// TracerThread.
func NewTracerThread(pid int) *TracerThread {
	return &TracerThread{
		thread: threadlock.New(threadlock.Config[*Tracer]{
			Obj: New(pid),
		}),
	}
}

// TracerThread provides access to a dedicated operating system thread
// from which ptrace calls can be executed.
//
// Callers should call the Thread's Close method once finished with
// the TracerThread.
//
// It appears that most (all?) Unix-like systems require ptrace operations
// for a specific process to come from the same OS thread:
// https://stackoverflow.com/questions/16767832/ptrace-not-recognizing-child-process
//
// This abstraction is required since Go's concurrency model does not
// normally expose threads to the programmer and makes no guarantees
// about which OS thread will execute a given function or go routine.
type TracerThread struct {
	thread *threadlock.Thread[*Tracer]
}

func (o *TracerThread) Close() error {
	return o.thread.Close()
}

func (o *TracerThread) Do(ctx context.Context, fn func(context.Context, *Tracer) error) error {
	return o.thread.Do(ctx, fn)
}

func (o *TracerThread) AttachAndWaitStopped(ctx context.Context) error {
	return o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		return pt.AttachAndWaitStopped()
	})
}

func (o *TracerThread) Attach(ctx context.Context) error {
	return o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		return pt.Attach()
	})
}

func (o *TracerThread) Stop(ctx context.Context) error {
	return o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		return pt.Stop()
	})
}

func (o *TracerThread) Signal(ctx context.Context, sig syscall.Signal) error {
	return o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		return pt.Signal(sig)
	})
}

func (o *TracerThread) Wait(ctx context.Context, options int) (unix.WaitStatus, unix.Rusage, error) {
	var wstatus unix.WaitStatus
	var rusage unix.Rusage
	var err error

	err = o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		wstatus, rusage, err = pt.Wait(options)
		return err
	})

	return wstatus, rusage, err
}

func (o *TracerThread) RequestPtr(ctx context.Context, request int, addr unsafe.Pointer, data int) error {
	return o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		return pt.RequestPtr(request, addr, data)
	})
}

func (o *TracerThread) Request(ctx context.Context, request int, addr uintptr, data int) error {
	return o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		return pt.Request(request, addr, data)
	})
}

func (o *TracerThread) Cont(ctx context.Context) error {
	return o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		return pt.Cont()
	})
}

func (o *TracerThread) ContSignal(ctx context.Context, sig syscall.Signal) error {
	return o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		return pt.ContSignal(sig)
	})
}

func (o *TracerThread) Detach(ctx context.Context) error {
	return o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		return pt.Detach()
	})
}

func (o *TracerThread) SingleStep(ctx context.Context) error {
	return o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		return pt.SingleStep()
	})
}

func (o *TracerThread) WaitStopped(ctx context.Context) (unix.WaitStatus, unix.Rusage, error) {
	var ws unix.WaitStatus
	var rusage unix.Rusage
	var err error

	err = o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		ws, rusage, err = pt.WaitStopped()
		return err
	})

	return ws, rusage, err
}

func (o *TracerThread) PeekData(ctx context.Context, addr uintptr, out []byte) (int, error) {
	var i int
	var err error

	err = o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		i, err = pt.PeekData(addr, out)
		return err
	})

	return i, err
}

func (o *TracerThread) PokeData(ctx context.Context, addr uintptr, data []byte) (int, error) {
	var i int
	var err error

	err = o.thread.Do(ctx, func(_ context.Context, pt *Tracer) error {
		i, err = pt.PokeData(addr, data)
		return err
	})

	return i, err
}
