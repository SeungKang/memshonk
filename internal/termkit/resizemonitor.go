package termkit

import (
	"context"
	"sync"
)

func NewResizedMonitor(ctx context.Context, fd uintptr) *Resized {
	var cancelFn func()
	ctx, cancelFn = context.WithCancel(ctx)

	return &Resized{
		events:   monitorResizeEvents(ctx, fd),
		cancelFn: cancelFn,
		done:     ctx.Done(),
	}
}

type Resized struct {
	events    <-chan ResizeEvent
	canFnOnce sync.Once
	cancelFn  func()
	done      <-chan struct{}
}

func (o *Resized) Done() <-chan struct{} {
	return o.done
}

func (o *Resized) Close() error {
	o.canFnOnce.Do(o.cancelFn)

	return nil
}

func (o *Resized) Events() <-chan ResizeEvent {
	return o.events
}

type ResizeEvent struct {
	Width  int
	Height int
}
