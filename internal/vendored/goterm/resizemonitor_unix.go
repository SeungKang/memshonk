//go:build unix

package goterm

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/term"
)

func monitorResizeEventsOS(ctx context.Context, fd uintptr, events chan ResizeEvent) {
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

			select {
			case <-ctx.Done():
				return
			case events <- ResizeEvent{
				NewSize: Size{
					Cols: width,
					Rows: height,
				},
				Err: err,
			}:
			}
		}
	}()
}
