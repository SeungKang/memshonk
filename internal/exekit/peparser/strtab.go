package peparser

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/SeungKang/memshonk/internal/exekit/epc"
)

// parseCOFFStrings parses the COFF symbol table and string table.
// The COFF symbol table is pointed to by PointerToSymbolTable in the COFF header.
// The string table immediately follows the symbol table.
func (p *Parser) parseCOFFStrings(ctx context.Context) error {
	if p.cfg.OnStringFn == nil {
		return nil
	}

	// Check if COFF symbol table exists
	symTableOffset := p.coffHeader.PointerToSymbolTable
	numSymbols := p.coffHeader.NumberOfSymbols

	if symTableOffset == 0 || numSymbols == 0 {
		// No COFF symbol table present (typical for images)
		return nil
	}

	// String table is immediately after the symbol table
	// Each symbol entry is 18 bytes
	strtabOffset := int64(symTableOffset) + int64(numSymbols)*COFFSymbolSize

	// Read string table size (first 4 bytes)
	strtab, err := p.readCOFFStringTable(strtabOffset)
	if err != nil {
		// Not an error if string table doesn't exist or is empty
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}

	if len(strtab) == 0 {
		return nil
	}

	// Emit strings from the string table
	return p.emitCOFFStrings(ctx, strtab, uint64(strtabOffset))
}

// readCOFFStringTable reads the COFF string table from the given offset.
// Returns the string table contents (excluding the size field).
func (p *Parser) readCOFFStringTable(offset int64) ([]byte, error) {
	// Read the size (first 4 bytes)
	var sizeBuf [4]byte
	if _, err := p.r.ReadAt(sizeBuf[:], offset); err != nil {
		return nil, err
	}

	size := p.byteOrder.Uint32(sizeBuf[:])

	// Size includes the 4-byte size field itself
	// A size of 4 means no strings are present
	if size <= 4 {
		return nil, nil
	}

	// Read the entire string table
	strtab := make([]byte, size)
	if _, err := p.r.ReadAt(strtab, offset); err != nil {
		if errors.Is(err, io.EOF) && len(strtab) > 4 {
			// Partial read is okay, use what we got
			return strtab, nil
		}
		return nil, fmt.Errorf("reading COFF string table: %w", err)
	}

	return strtab, nil
}

// emitCOFFStrings parses and emits individual strings from the COFF string table.
func (p *Parser) emitCOFFStrings(ctx context.Context, strtab []byte, baseOffset uint64) error {
	// Skip the 4-byte size field
	if len(strtab) <= 4 {
		return nil
	}

	i := uint64(4)
	for i < uint64(len(strtab)) {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Skip null bytes
		if strtab[i] == 0 {
			i++
			continue
		}

		// Find end of string
		start := i
		for i < uint64(len(strtab)) && strtab[i] != 0 {
			i++
		}

		str := string(strtab[start:i])
		if len(str) > 0 {
			s := epc.String{
				Value:  str,
				Offset: baseOffset + start,
				Source: "coff",
			}
			if err := p.cfg.OnStringFn(p.exeID, p.index, s); err != nil {
				return err
			}
		}

		i++ // Skip null terminator
	}

	return nil
}
