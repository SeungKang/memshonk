package goterm

import (
	"context"
	"fmt"
	"sync"

	"golang.org/x/term"
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
	NewSize Size
	Err     error
}

func monitorResizeEvents(ctx context.Context, fd uintptr) <-chan ResizeEvent {
	events := make(chan ResizeEvent, 1)

	width, height, err := term.GetSize(int(fd))
	if err == nil {
		events <- ResizeEvent{
			NewSize: Size{
				Cols: width,
				Rows: height,
			},
		}
	} else {
		events <- ResizeEvent{
			Err: fmt.Errorf("failed to get initial terminal size - %w", err),
		}
	}

	monitorResizeEventsOS(ctx, fd, events)

	return events
}
