package progctl

import "github.com/SeungKang/memshonk/internal/events"

type AttachedEvent struct {
	Pid   int
	acker *events.EventAcker
}

func (o *AttachedEvent) Acker() *events.EventAcker {
	return o.acker
}

type DetachedEvent struct {
	acker *events.EventAcker
}

func (o *DetachedEvent) Acker() *events.EventAcker {
	return o.acker
}

type ProcessExitedEvent struct {
	Reason error
}
