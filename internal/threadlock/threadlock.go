package threadlock

import (
	"context"
	"fmt"
	"runtime"
	"sync"
)

// Config configures a Thread.
type Config[C comparable] struct {
	// Obj is the object that will be made available to the Thread.
	Obj C
}

// New starts a new go routine which locks itself to an operating system
// thread and executes functions on the dedicated OS thread using the Do
// method.
//
// Callers should call the Thread's Close method when the Thread is no
// longer needed.
func New[C comparable](config Config[C]) *Thread[C] {
	ctx, cancelFn := context.WithCancel(context.Background())

	thread := &Thread[C]{
		config:    config,
		callbacks: make(chan *callback[C]),
		cancelFn:  cancelFn,
		exited:    make(chan struct{}),
	}

	go thread.loop(ctx)

	return thread
}

// Thread represents a dedicated operating system thread. Callers may use the
// Do method to execute a function on the dedicated thread.
type Thread[C comparable] struct {
	config    Config[C]
	callbacks chan *callback[C]
	closeOnce sync.Once
	cancelFn  func()
	exited    chan struct{}
}

// Do executes function fn in the dedicated OS thread.
//
// Function fn will receive the object passed to New function along with
// a context.Context that represents the Context passed to this method
// and the dedicated OS thread's own Context.
func (o *Thread[C]) Do(ctx context.Context, fn func(ctx context.Context, obj C) error) error {
	cb := &callback[C]{
		ctx:  ctx,
		fn:   fn,
		done: make(chan struct{}),
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-o.exited:
		return fmt.Errorf("thread has been shutdown")
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

type callback[C comparable] struct {
	ctx  context.Context
	fn   func(context.Context, C) error
	done chan struct{}
	err  error
}

func (o *Thread[C]) loop(ctx context.Context) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	defer close(o.exited)

	for {
		select {
		case <-ctx.Done():
			return
		case cb := <-o.callbacks:
			fnCtx, cancelFnCtx := context.WithCancel(ctx)

			doneFn := context.AfterFunc(cb.ctx, cancelFnCtx)

			err := cb.fn(fnCtx, o.config.Obj)

			cancelFnCtx()
			doneFn()

			cb.err = err
			close(cb.done)
		}
	}
}

// Close stops the dedicated thread. Calls to the Do method will fail once
// this method is called.
func (o *Thread[C]) Close() error {
	o.closeOnce.Do(func() {
		o.cancelFn()
	})

	return nil
}

// Done returns a channel that is closed when the thread exits.
func (o *Thread[C]) Done() <-chan struct{} {
	return o.exited
}
