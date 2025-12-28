package main

import (
	"log"
	"time"

	"github.com/SeungKang/memshonk/internal/vendored/goterm"
)

func main() {
	log.SetFlags(0)

	err := mainWithError()
	if err != nil {
		log.Fatalln("fatal:", err)
	}
}

func mainWithError() error {
	tm, err := goterm.NewStdioScreen()
	if err != nil {
		return err
	}

	tm.Clear() // Clear current screen

	for {
		// By moving cursor to top-left position we ensure that console output
		// will be overwritten each time, instead of adding new.
		tm.MoveCursor(1, 1)

		tm.Println("Current Time:", time.Now().Format(time.RFC1123))

		tm.Flush() // Call it every time at the end of rendering

		time.Sleep(time.Second)
	}
}
