package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/SeungKang/memshonk/internal/exedata"
)

func main() {
	log.SetFlags(0)

	err := mainWithError()
	if err != nil {
		log.Fatalln("fatal:", err)
	}
}

func mainWithError() error {
	targetOs := flag.String("o", runtime.GOOS, "Target OS")

	flag.Parse()

	filePath := flag.Arg(0)
	if filePath == "" {
		return errors.New("please specify an executable's file path as the last argument")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	var exe exedata.Exe

	switch *targetOs {
	case "darwin":
		exe, err = exedata.ParseMachoCurrentArch(file)
	case "windows":
		exe, err = exedata.ParsePe(file)
	default:
		exe, err = exedata.ParseElf(file)
	}

	if err != nil {
		return err
	}

	for _, s := range exe.Symbols() {
		fmt.Printf("%+v\n", s)
	}

	return nil
}
