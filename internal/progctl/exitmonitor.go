package progctl

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/SeungKang/memshonk/internal/events"
)

func newExitMonitor(pub *events.Publisher[ProcessExitedEvent]) *ExitMonitor {
	return &ExitMonitor{
		events: pub,
		c:      make(chan struct{}),
	}
}

type ExitMonitor struct {
	events *events.Publisher[ProcessExitedEvent]
	c      chan struct{}
	once   sync.Once
	err    error
}

func (o *ExitMonitor) Done() <-chan struct{} {
	return o.c
}

func (o *ExitMonitor) Err() error {
	return o.err
}

func (o *ExitMonitor) SetDetached() {
	o.SetExited(&ExitMonitorProcExitErr{
		OptMonitorErr: ErrDetached,
	})
}

type ExitMonitorProcExitErr struct {
	Source        string
	OptTime       time.Time
	OptMonitorErr error
	OptExitStatus *int64
}

func (o ExitMonitorProcExitErr) Error() string {
	header := o.Source + "@" + o.OptTime.Format(time.Stamp) + ": "

	if o.OptMonitorErr != nil {
		msg := header + "exit monitor failed - "

		if o.OptMonitorErr == nil {
			return msg + "no additional information available"
		} else {
			return msg + o.OptMonitorErr.Error()
		}
	}

	if o.OptExitStatus != nil {
		return fmt.Sprintf("%sprocess exited with status: %d",
			header, *o.OptExitStatus)
	}

	return header + "process exited with unknown exit status"
}

func (o ExitMonitorProcExitErr) Unwrap() error {
	return o.OptMonitorErr
}

func (o *ExitMonitor) SetExited(err *ExitMonitorProcExitErr) {
	o.once.Do(func() {
		if err.OptTime.Equal(time.Time{}) {
			err.OptTime = time.Now()
		}

		switch err.OptMonitorErr {
		case ErrDetached:
			// Do not send an event.
		default:
			_ = o.events.Send(context.Background(), ProcessExitedEvent{err})
		}

		o.err = err

		close(o.c)
	})
}
