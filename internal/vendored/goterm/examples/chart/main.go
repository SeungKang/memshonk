package main

import (
	"log"
	"math"

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
	tm.MoveCursor(0, 0)

	chart := goterm.NewLineChart(100, 20)
	data := new(goterm.DataTable)
	data.AddColumn("Time")
	data.AddColumn("Sin(x)")
	data.AddColumn("Cos(x+1)")

	for i := 0.1; i < 10; i += 0.1 {
		data.AddRow(i, math.Sin(i), math.Cos(i+1))
	}

	tm.Println(chart.Draw(data))
	tm.Flush()

	return nil
}
