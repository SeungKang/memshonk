//go:build unix

package termkit

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

func monitorResizeEvents(ctx context.Context, fd uintptr) <-chan ResizeEvent {
	events := make(chan ResizeEvent)

	sigWinches := make(chan os.Signal, 1)
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

			width, height, err := term.GetSize(int(fd))
			if err != nil {
				return
			}

			select {
			case <-ctx.Done():
				return
			case events <- ResizeEvent{
				Width:  width,
				Height: height,
			}:
			}
		}
	}()

	return events
}
