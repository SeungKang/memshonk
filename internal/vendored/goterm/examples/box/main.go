package main

import (
	"fmt"
	"log"

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

	tm.Clear()

	// Create Box with 30% width of current screen, and height of 20 lines
	box, err := goterm.NewBox(30|goterm.PCT, 20, 0, tm)
	if err != nil {
		return err
	}

	// Add some content to the box
	// Note that you can add ANY content, even tables
	fmt.Fprint(box, "Some box content")

	// Move Box to approx center of the screen
	boxStr, err := tm.MoveTo(box.String(), 40|goterm.PCT, 40|goterm.PCT)
	if err != nil {
		return err
	}

	tm.Println(boxStr)

	return tm.Flush()
}
