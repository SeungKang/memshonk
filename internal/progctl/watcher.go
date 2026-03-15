package progctl

import (
	"bytes"
	"context"
	"fmt"
	"sync"
	"time"
)

func newWatcherCtl(proc *process) *watcherCtl {
	ctx, cancelFn := context.WithCancel(context.Background())

	ctl := &watcherCtl{
		addWatcher: make(chan *Watcher),
		process:    proc,
		cancelFn:   cancelFn,
		done:       ctx.Done(),
	}

	go ctl.loop(ctx)

	return ctl
}

type watcherCtl struct {
	addWatcher chan *Watcher
	process    *process
	cancelFn   func()
	done       <-chan struct{}
}

func (o *watcherCtl) AddWatcher(ctx context.Context, watcher *Watcher) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-o.done:
		return fmt.Errorf("watcher controller has been shutdown")
	case o.addWatcher <- watcher:
		return nil
	}
}

func (o *watcherCtl) loop(ctx context.Context) {
	watchers := make(map[*Watcher]struct{})

	runWatchers := time.NewTicker(time.Hour)
	runWatchers.Stop()

	defer func() {
		runWatchers.Stop()

		for w := range watchers {
			w.stop()
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case watcher := <-o.addWatcher:
			watchers[watcher] = struct{}{}

			if len(watchers) == 1 {
				runWatchers.Reset(50 * time.Millisecond)
			}
		case <-runWatchers.C:
			for watcher := range watchers {
				if !watcher.run(o.process) {
					delete(watchers, watcher)
				}
			}

			if len(watchers) == 0 {
				runWatchers.Stop()
			}
		}
	}
}

func (o *watcherCtl) Close() error {
	o.cancelFn()

	return nil
}

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

type readBytesOnly interface {
	ReadBytes(addr uintptr, size uint64) ([]byte, error)
}

func (o *Watcher) run(proc readBytesOnly) bool {
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
