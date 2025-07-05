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
		if err == nil {
			err = ErrExitedNormally
		}

		_ = o.events.Send(context.Background(), ProcessExitedEvent{err})

		o.err = err

		close(o.c)
	})
}

type ProcessExitedEvent struct {
	Reason error
}
