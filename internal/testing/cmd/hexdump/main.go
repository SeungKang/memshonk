package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/SeungKang/memshonk/internal/hexdump"
)

func main() {
	log.SetFlags(0)

	err := mainWithError()
	if err != nil {
		log.Fatalln("fatal:", err)
	}

	os.Stdout.WriteString("\n")
}

func mainWithError() error {
	flag.Parse()

	return hexdump.Dump(context.Background(), hexdump.Config{
		Src:    os.Stdin,
		Dst:    os.Stdout,
		Colors: hexdump.NewColors(),
	})
}
