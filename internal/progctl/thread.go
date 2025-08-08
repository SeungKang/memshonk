package progctl

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"time"
)

func newProcessThread(exeName string, pid int, exitMon *ExitMonitor) (*processThread, error) {
	thread := &processThread{
		callbacks:  make(chan *processThreadCallback),
		addWatcher: make(chan *Watcher),
		exitMon:    exitMon,
	}

	attachResult := make(chan error, 1)

	go thread.loop(exeName, pid, attachResult)

	err := <-attachResult
	if err != nil {
		return nil, err
	}

	return thread, nil
}

// we did this because on linux ptrace operations need to be executed by the same thread
// https://stackoverflow.com/questions/16767832/ptrace-not-recognizing-child-process
type processThread struct {
	exitMon    *ExitMonitor
	callbacks  chan *processThreadCallback
	addWatcher chan *Watcher
	process    attachedProcess
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

func (o *processThread) AddWatcher(ctx context.Context, watcher *Watcher) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-o.callbacks:
		return errors.New("process thread exited")
	case o.addWatcher <- watcher:
		return nil
	}
}

func (o *processThread) loop(exeName string, pid int, attachResult chan error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	var err error
	o.process, err = attach(exeName, pid, o.exitMon)
	if err != nil {
		attachResult <- fmt.Errorf("attach failure - %w", err)
		return
	}
	close(attachResult)

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
		case cb, isOpen := <-o.callbacks:
			if !isOpen {
				return
			}

			err := cb.fn(o.process)

			cb.err = err
			close(cb.done)
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

func (o *processThread) ExitMonitor() *ExitMonitor {
	return o.process.ExitMonitor()
}

func (o *processThread) PID() int {
	return o.process.PID()
}

func (o *processThread) ExeObj() ExeInfo {
	return o.process.ExeInfo()
}

func (o *processThread) Close(ctx context.Context) error {
	err := o.Do(ctx, func(process attachedProcess) error {
		return process.Close()
	})

	close(o.callbacks)

	return err
}
