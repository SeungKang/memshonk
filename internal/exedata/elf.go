package exedata

import (
	"debug/elf"
	"errors"
	"fmt"
	"io"
)

var _ Exe = (*Elf)(nil)

func ParseElf(readerAt io.ReaderAt, options ParserOptions) (Elf, error) {
	elfFile, err := elf.NewFile(readerAt)
	if err != nil {
		return Elf{}, fmt.Errorf("failed to create elf file object - %w", err)
	}

	text := elfFile.Section(".text")
	if text == nil {
		return Elf{}, errors.New("elf is missing .text section")
	}

	syms, err := parseElfSymbols(elfFile, text)
	if err != nil {
		return Elf{}, fmt.Errorf("failed to parse symbols - %w", err)
	}

	return Elf{
		elfFile: elfFile,
		text:    text,
		syms:    syms,
	}, nil
}

type Elf struct {
	elfFile *elf.File
	text    *elf.Section
	syms    []Symbol
}

func (o Elf) Symbols() []Symbol {
	return o.syms
}

func parseElfSymbols(elfFile *elf.File, text *elf.Section) ([]Symbol, error) {
	syms, err := elfFile.Symbols()
	if err != nil {
		return nil, err
	}

	symbols := make([]Symbol, len(syms))

	for i, sym := range syms {
		// Calculate ELF symbol offset by Stackoverflow users
		// Norbert Lange and evandrix:
		// https://stackoverflow.com/a/40249502
		//
		// fn symbol VA - .text VA + .text offset
		symbols[i] = Symbol{
			Name:     sym.Name,
			Location: sym.Value - text.Addr + text.Offset,
			Size:     sym.Size,
		}
	}

	return symbols, nil
}
