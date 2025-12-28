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

	tm.Clear() // Clear current screen

	started := 100
	finished := 250

	// Based on http://golang.org/pkg/text/tabwriter
	totals := goterm.NewTable(0, 10, 5, ' ', 0)
	fmt.Fprintf(totals, "Time\tStarted\tActive\tFinished\n")
	fmt.Fprintf(totals, "%s\t%d\t%d\t%d\n", "All", started, started-finished, finished)
	tm.Println(totals)

	tm.Flush()

	return nil
}
