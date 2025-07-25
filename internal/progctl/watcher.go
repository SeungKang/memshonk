package progctl

import (
	"bytes"
	"context"
	"sync"
)

func newWatcher(ctx context.Context, addr uintptr, size uint64) *Watcher {
	var cancelFn func()
	ctx, cancelFn = context.WithCancel(ctx)

	return &Watcher{
		addr:      addr,
		size:      size,
		reads:     make(chan WatcherReadResult, 10),
		cancelled: ctx.Done(),
		cancelFn:  cancelFn,
	}
}

type Watcher struct {
	addr      uintptr
	size      uint64
	readsOnce sync.Once
	reads     chan WatcherReadResult
	cancelled <-chan struct{}
	canFnOnce sync.Once
	cancelFn  func()
	lastRead  []byte
	lastErr   error
}

func (o *Watcher) Close() error {
	o.canFnOnce.Do(o.cancelFn)

	return nil
}

func (o *Watcher) Addr() uintptr {
	return o.addr
}

func (o *Watcher) Results() <-chan WatcherReadResult {
	return o.reads
}

func (o *Watcher) Err() error {
	return o.lastErr
}

func (o *Watcher) run(proc attachedProcess) bool {
	select {
	case <-o.cancelled:
		o.lastErr = context.Canceled

		o.stop()

		return false
	default:
		// Keep going.
	}

	b, err := proc.ReadBytes(o.addr, o.size)
	if err != nil {
		o.lastErr = err

		o.stop()

		return false
	}

	o.lastErr = nil

	if bytes.Equal(o.lastRead, b) {
		return true
	}

	o.lastRead = b

	select {
	case <-o.cancelled:
		o.lastErr = context.Canceled

		o.stop()

		return false
	case o.reads <- WatcherReadResult{
		Data: b,
	}:
		return true
	default:
		// Allow write failures in case peer Go routine
		// is falling behind or is unresponsive.
		return true
	}
}

func (o *Watcher) stop() {
	o.readsOnce.Do(func() {
		close(o.reads)
	})
}

type WatcherReadResult struct {
	Data []byte
}
