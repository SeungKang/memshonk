//go:build windows

package goterm

import (
	"context"
	"time"

	"golang.org/x/term"
)

func monitorResizeEventsOS(ctx context.Context, fd uintptr, events chan ResizeEvent) {
	go func() {
		// Stackoverflow user ChrisV suggested
		// using a combination of SetConsoleMode
		// and WaitForSingleObject... Not gonna
		// be very fun to implement:
		// https://stackoverflow.com/a/10857339
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()

		var lastWidth int
		var lastHeight int

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				// Keep going.
			}

			width, height, err := term.GetSize(int(fd))

			if width == lastWidth && height == lastHeight {
				continue
			}

			lastWidth = width
			lastHeight = height

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
