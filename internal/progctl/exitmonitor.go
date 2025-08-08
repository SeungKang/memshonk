package progctl

import (
	"context"
	"sync"

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

func (o *ExitMonitor) SetExited(err error) {
	o.once.Do(func() {
		switch err {
		case ErrDetached:
			// Do not send an event.
		default:
			_ = o.events.Send(context.Background(), ProcessExitedEvent{err})
		}

		o.err = err

		close(o.c)
	})
}
