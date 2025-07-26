package main

import (
	"context"
	"flag"
	"log"
	"time"

	"github.com/SeungKang/memshonk/internal/termkit"
	"github.com/buger/goterm"
)

func main() {
	log.SetFlags(0)

	err := mainWithError()
	if err != nil {
		log.Fatalln("fatal:", err)
	}
}

func mainWithError() error {
	flag.Parse()

	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	resized := termkit.NewResizedMonitor(ctx)

	width := goterm.Width()
	height := goterm.Height()

	goterm.Clear()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case resize := <-resized.Events():
			width = resize.Width
			height = resize.Height

			goterm.MoveCursor(1, 1)
			goterm.Clear()

			goterm.MoveCursor(1, height-5)
			goterm.Printf("%s - w: %d | h: %d", time.Now(), width, height)
			goterm.Flush()
		}
	}
}
