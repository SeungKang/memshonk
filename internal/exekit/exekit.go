// Package exekit provides executable file format parsing with a unified data model.
//
// The library supports multiple executable formats (ELF, Mach-O, PE) and handles
// situations where an executable contains multiple sub-executables (such as
// Mach-O "fat" binaries with multiple CPU architecture builds).
package exekit

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/SeungKang/memshonk/internal/exekit/elfparser"
	"github.com/SeungKang/memshonk/internal/exekit/epc"
	"github.com/SeungKang/memshonk/internal/exekit/machoparser"
	"github.com/SeungKang/memshonk/internal/exekit/peparser"
)

// Re-export common types from epc for convenience
type (
	ParserConfig    = epc.ParserConfig
	ExeFmtOption    = epc.ExeFmtOption
	Info            = epc.Info
	Function        = epc.Function
	ExportedCode    = epc.ExportedCode
	ImportedCode    = epc.ImportedCode
	ImportedLibrary = epc.ImportedLibrary
	Reloc           = epc.Reloc
	Section         = epc.Section
	Segment         = epc.Segment
	String          = epc.String
	Symbol          = epc.Symbol
)

// CallbackFn is re-exported as a generic type alias.
// Use it as CallbackFn[T] where T is the object type (e.g., CallbackFn[Info]).
type CallbackFn[T any] = epc.CallbackFn[T]

// Common errors
var (
	ErrUnknownFormat = errors.New("unknown executable format")
	ErrNoSource      = errors.New("no source reader provided")
)

// Magic bytes for format detection
var (
	elfMagic   = []byte{0x7f, 'E', 'L', 'F'}
	machoMagic32 = []byte{0xfe, 0xed, 0xfa, 0xce} // MH_MAGIC
	machoMagic64 = []byte{0xfe, 0xed, 0xfa, 0xcf} // MH_MAGIC_64
	machoMagic32Rev = []byte{0xce, 0xfa, 0xed, 0xfe} // MH_CIGAM
	machoMagic64Rev = []byte{0xcf, 0xfa, 0xed, 0xfe} // MH_CIGAM_64
	machoFat   = []byte{0xca, 0xfe, 0xba, 0xbe} // FAT_MAGIC
	machoFatRev = []byte{0xbe, 0xba, 0xfe, 0xca} // FAT_CIGAM
	peMagic    = []byte{'M', 'Z'}
)

// Parse parses an executable file using the provided configuration.
// It automatically detects the file format based on magic bytes and
// dispatches to the appropriate format-specific parser.
func Parse(ctx context.Context, cfg *epc.ParserConfig) error {
	if cfg == nil || cfg.Src == nil {
		return ErrNoSource
	}

	// Read magic bytes for format detection
	magic := make([]byte, 4)
	if _, err := cfg.Src.ReadAt(magic, 0); err != nil {
		if errors.Is(err, io.EOF) {
			return fmt.Errorf("file too small to determine format: %w", err)
		}
		return fmt.Errorf("reading magic bytes: %w", err)
	}

	// Detect format and parse
	switch {
	case matchMagic(magic, elfMagic):
		return elfparser.Parse(ctx, cfg)

	case matchMagic(magic, machoMagic32),
		matchMagic(magic, machoMagic64),
		matchMagic(magic, machoMagic32Rev),
		matchMagic(magic, machoMagic64Rev):
		return machoparser.Parse(ctx, cfg)

	case matchMagic(magic, machoFat),
		matchMagic(magic, machoFatRev):
		return machoparser.ParseFat(ctx, cfg)

	case matchMagic(magic[:2], peMagic):
		return peparser.Parse(ctx, cfg)

	default:
		return ErrUnknownFormat
	}
}

func matchMagic(data, magic []byte) bool {
	if len(data) < len(magic) {
		return false
	}
	for i, b := range magic {
		if data[i] != b {
			return false
		}
	}
	return true
}
