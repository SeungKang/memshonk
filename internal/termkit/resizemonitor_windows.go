//go:build windows

package termkit

import (
	"context"
	"time"

	"github.com/buger/goterm"
)

func monitorResizeEvents(ctx context.Context) <-chan ResizeEvent {
	events := make(chan ResizeEvent)

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

			width := goterm.Width()
			height := goterm.Height()

			if width == lastWidth && height == lastHeight {
				continue
			}

			lastWidth = width
			lastHeight = height

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
