package peparser

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/exekit/epc"
)

// parseImports parses the import directory and emits callbacks.
func (p *Parser) parseImports(ctx context.Context) error {
	// Check if we need to parse imports
	if p.cfg.OnImportedLibraryFn == nil && p.cfg.OnImportedCodeFn == nil {
		return nil
	}

	dir := p.getDataDirectory(IMAGE_DIRECTORY_ENTRY_IMPORT)
	if dir.VirtualAddress == 0 || dir.Size == 0 {
		return nil
	}

	offset, ok := p.rvaToOffset(dir.VirtualAddress)
	if !ok {
		return nil
	}

	// Read import descriptors (20 bytes each)
	const descSize = 20

	for i := uint32(0); ; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		descOffset := offset + i*descSize

		// Read the import descriptor
		var buf [20]byte
		if _, err := p.r.ReadAt(buf[:], int64(descOffset)); err != nil {
			return nil // End of file, stop parsing
		}

		desc := ImportDescriptor{
			OriginalFirstThunk: p.byteOrder.Uint32(buf[0:4]),
			TimeDateStamp:      p.byteOrder.Uint32(buf[4:8]),
			ForwarderChain:     p.byteOrder.Uint32(buf[8:12]),
			Name:               p.byteOrder.Uint32(buf[12:16]),
			FirstThunk:         p.byteOrder.Uint32(buf[16:20]),
		}

		// Check for null terminator
		if desc.Name == 0 && desc.FirstThunk == 0 {
			break
		}

		// Skip if no name
		if desc.Name == 0 {
			continue
		}

		// Read DLL name
		dllName := p.readString(desc.Name)
		if dllName == "" {
			continue
		}

		// Emit library callback
		if p.cfg.OnImportedLibraryFn != nil {
			lib := epc.ImportedLibrary{
				Name:   dllName,
				Offset: uint64(descOffset),
			}
			if err := p.cfg.OnImportedLibraryFn(p.exeID, p.index, lib); err != nil {
				return err
			}
		}

		// Parse thunks if we have an imported code callback
		if p.cfg.OnImportedCodeFn != nil {
			// Use OriginalFirstThunk (ILT) if available, otherwise FirstThunk (IAT)
			thunkRVA := desc.OriginalFirstThunk
			if thunkRVA == 0 {
				thunkRVA = desc.FirstThunk
			}

			if thunkRVA != 0 {
				if err := p.parseImportThunks(ctx, thunkRVA, dllName); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// parseImportThunks parses the import lookup table for a single DLL.
func (p *Parser) parseImportThunks(ctx context.Context, thunkRVA uint32, dllName string) error {
	thunkOffset, ok := p.rvaToOffset(thunkRVA)
	if !ok {
		return nil
	}

	if p.is64Bit {
		return p.parseImportThunks64(ctx, thunkOffset, dllName)
	}
	return p.parseImportThunks32(ctx, thunkOffset, dllName)
}

// parseImportThunks32 parses 32-bit import thunks.
func (p *Parser) parseImportThunks32(ctx context.Context, offset uint32, dllName string) error {
	for i := uint32(0); ; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		thunkOffset := offset + i*4

		var buf [4]byte
		if _, err := p.r.ReadAt(buf[:], int64(thunkOffset)); err != nil {
			return nil // End of file
		}

		thunk := p.byteOrder.Uint32(buf[:])
		if thunk == 0 {
			break // Null terminator
		}

		var name string
		var importType uint8 = 2 // STT_FUNC equivalent

		if IsOrdinalImport32(thunk) {
			// Import by ordinal
			ordinal := Ordinal32(thunk)
			name = fmt.Sprintf("#%d", ordinal)
		} else {
			// Import by name
			hintNameRVA := HintNameRVA32(thunk)
			name = p.readHintName(hintNameRVA)
			if name == "" {
				continue
			}
		}

		imp := epc.ImportedCode{
			Name:    name,
			Library: dllName,
			Offset:  uint64(thunkOffset),
			Type:    importType,
			Binding: 1, // STB_GLOBAL equivalent
		}

		if err := p.cfg.OnImportedCodeFn(p.exeID, p.index, imp); err != nil {
			return err
		}
	}

	return nil
}

// parseImportThunks64 parses 64-bit import thunks.
func (p *Parser) parseImportThunks64(ctx context.Context, offset uint32, dllName string) error {
	for i := uint32(0); ; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		thunkOffset := offset + i*8

		var buf [8]byte
		if _, err := p.r.ReadAt(buf[:], int64(thunkOffset)); err != nil {
			return nil // End of file
		}

		thunk := p.byteOrder.Uint64(buf[:])
		if thunk == 0 {
			break // Null terminator
		}

		var name string
		var importType uint8 = 2 // STT_FUNC equivalent

		if IsOrdinalImport64(thunk) {
			// Import by ordinal
			ordinal := Ordinal64(thunk)
			name = fmt.Sprintf("#%d", ordinal)
		} else {
			// Import by name
			hintNameRVA := HintNameRVA64(thunk)
			name = p.readHintName(hintNameRVA)
			if name == "" {
				continue
			}
		}

		imp := epc.ImportedCode{
			Name:    name,
			Library: dllName,
			Offset:  uint64(thunkOffset),
			Type:    importType,
			Binding: 1, // STB_GLOBAL equivalent
		}

		if err := p.cfg.OnImportedCodeFn(p.exeID, p.index, imp); err != nil {
			return err
		}
	}

	return nil
}

// readHintName reads a hint/name entry from an RVA.
// The format is: 2-byte hint, followed by null-terminated name.
func (p *Parser) readHintName(rva uint32) string {
	offset, ok := p.rvaToOffset(rva)
	if !ok {
		return ""
	}

	// Skip the 2-byte hint and read the name
	return p.readStringAt(int64(offset + 2))
}
