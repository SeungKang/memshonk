package progctl

import (
	"context"
	"fmt"
	"github.com/SeungKang/memshonk/internal/memory"
	"runtime"
)

func newProcessThread(exeName string, pid int) (*processThread, error) {
	thread := &processThread{
		callbacks: make(chan *processThreadCallback),
	}

	attachResult := make(chan error, 1)

	go thread.loop(exeName, pid, attachResult)

	err := <-attachResult
	if err != nil {
		return nil, err
	}

	return thread, nil
}

type processThread struct {
	callbacks chan *processThreadCallback
	process   attachedProcess
}

type processThreadCallback struct {
	fn   func(process attachedProcess) error
	done chan struct{}
	err  error
}

func (o *processThread) Do(ctx context.Context, fn func(process attachedProcess) error) error {
	cb := &processThreadCallback{
		fn:   fn,
		done: make(chan struct{}),
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case o.callbacks <- cb:
		// keep going
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-cb.done:
		return cb.err
	}
}

func (o *processThread) loop(exeName string, pid int, attachResult chan error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var err error
	o.process, err = attach(exeName, pid)
	if err != nil {
		attachResult <- fmt.Errorf("failed to attach process - %w", err)
		return
	}
	close(attachResult)

	for {
		select {
		case cb, isOpen := <-o.callbacks:
			if !isOpen {
				return
			}
			err := cb.fn(o.process)
			cb.err = err
			close(cb.done)
		}
	}
}

func (o *processThread) ExitMonitor() *ExitMonitor {
	return o.process.ExitMonitor()
}

func (o *processThread) PID() int {
	return o.process.PID()
}

func (o *processThread) ExeObj() memory.Object {
	return o.process.ExeObj()
}

func (o *processThread) Close(ctx context.Context) error {
	err := o.Do(ctx, func(process attachedProcess) error {
		return process.Close()
	})

	close(o.callbacks)

	return err
}
