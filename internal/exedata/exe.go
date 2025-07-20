package exedata

import (
	"os"
	"runtime"
)

type ParserOptions struct{}

func ParsePathForCurrentPlatform(exeFilePath string, options ParserOptions) (Exe, error) {
	// Note: It is safe to close the exeFile when this function ends
	// because this library's parsers use the NewFile function
	// from each respective exe Go library. The NewFile functions
	// do all of their parsing up front and do not require an
	// open exeFile descriptor to work.
	exeFile, err := os.Open(exeFilePath)
	if err != nil {
		return nil, err
	}
	defer exeFile.Close()

	switch runtime.GOOS {
	case "darwin", "ios":
		return ParseMachoCurrentArch(exeFile, options)
	case "windows":
		return ParsePe(exeFile, options)
	default:
		return ParseElf(exeFile, options)
	}
}

type Exe interface {
	Symbols() []Symbol
}

type Symbol struct {
	Name     string
	Location uint64
	Size     uint64
}
