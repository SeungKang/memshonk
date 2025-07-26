//go:build unix

package termkit

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/buger/goterm"
)

func monitorResizeEvents(ctx context.Context) <-chan ResizeEvent {
	events := make(chan ResizeEvent)

	sigWinches := make(chan os.Signal)
	signal.Notify(sigWinches, syscall.SIGWINCH)

	go func() {
		defer signal.Stop(sigWinches)

		for {
			select {
			case <-ctx.Done():
				return
			case <-sigWinches:
				// Keep going.
			}

			select {
			case <-ctx.Done():
				return
			case events <- ResizeEvent{
				Width:  goterm.Width(),
				Height: goterm.Height(),
			}:
			}
		}
	}()

	return events
}
