package main

import (
	"fmt"
	"log"
	"os"

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
	filePath := "box-example.txt"

	f, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Tell goterm to use the file we just opened, not stdout
	tm := goterm.NewScreen(goterm.NewVirtualTerminal(goterm.VirtualTerminalConfig{
		Input:  nil,
		Output: f,
	}))

	box, err := goterm.NewBox(30|goterm.PCT, 20, 0, tm)
	if err != nil {
		return err
	}

	fmt.Fprint(box, "Some box content")

	boxStr, err := tm.MoveTo(box.String(), 40|goterm.PCT, 40|goterm.PCT)
	if err != nil {
		return err
	}

	_, err = tm.Println(boxStr)
	if err != nil {
		return err
	}

	err = tm.Flush()
	if err != nil {
		return err
	}

	log.Printf("Now view the contents of '%s' in an ansi-capable terminal",
		filePath)

	return nil
}
