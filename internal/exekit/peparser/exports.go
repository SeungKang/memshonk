package peparser

import (
	"context"

	"github.com/SeungKang/memshonk/internal/exekit/epc"
)

// parseExports parses the export directory and emits callbacks.
func (p *Parser) parseExports(ctx context.Context) error {
	// Check if we need to parse exports
	if p.cfg.OnFunctionFn == nil && p.cfg.OnSymbolFn == nil && p.cfg.OnExportedCodeFn == nil {
		return nil
	}

	dir := p.getDataDirectory(IMAGE_DIRECTORY_ENTRY_EXPORT)
	if dir.VirtualAddress == 0 || dir.Size == 0 {
		return nil
	}

	offset, ok := p.rvaToOffset(dir.VirtualAddress)
	if !ok {
		return nil
	}

	// Read export directory (40 bytes)
	var buf [40]byte
	if _, err := p.r.ReadAt(buf[:], int64(offset)); err != nil {
		return nil
	}

	expDir := ExportDirectory{
		Characteristics:       p.byteOrder.Uint32(buf[0:4]),
		TimeDateStamp:         p.byteOrder.Uint32(buf[4:8]),
		MajorVersion:          p.byteOrder.Uint16(buf[8:10]),
		MinorVersion:          p.byteOrder.Uint16(buf[10:12]),
		Name:                  p.byteOrder.Uint32(buf[12:16]),
		Base:                  p.byteOrder.Uint32(buf[16:20]),
		NumberOfFunctions:     p.byteOrder.Uint32(buf[20:24]),
		NumberOfNames:         p.byteOrder.Uint32(buf[24:28]),
		AddressOfFunctions:    p.byteOrder.Uint32(buf[28:32]),
		AddressOfNames:        p.byteOrder.Uint32(buf[32:36]),
		AddressOfNameOrdinals: p.byteOrder.Uint32(buf[36:40]),
	}

	// Nothing to do if no functions
	if expDir.NumberOfFunctions == 0 {
		return nil
	}

	// Read the export tables
	functions, err := p.readRVATable(expDir.AddressOfFunctions, expDir.NumberOfFunctions)
	if err != nil {
		return nil
	}

	names, err := p.readRVATable(expDir.AddressOfNames, expDir.NumberOfNames)
	if err != nil {
		return nil
	}

	ordinals, err := p.readOrdinalTable(expDir.AddressOfNameOrdinals, expDir.NumberOfNames)
	if err != nil {
		return nil
	}

	// Track which function indices have names
	hasName := make(map[uint32]string, expDir.NumberOfNames)
	for i := uint32(0); i < expDir.NumberOfNames; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		ordinal := uint32(ordinals[i])
		funcName := p.readString(names[i])
		if funcName != "" {
			hasName[ordinal] = funcName
		}
	}

	// Emit callbacks for all functions
	for i := uint32(0); i < expDir.NumberOfFunctions; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		funcRVA := functions[i]
		if funcRVA == 0 {
			continue // Empty entry
		}

		biasedOrdinal := i + expDir.Base
		funcName := hasName[i]

		// Check if this is a forwarder (RVA points within the export section)
		isForwarder := funcRVA >= dir.VirtualAddress && funcRVA < dir.VirtualAddress+dir.Size

		var forwarderName string
		if isForwarder {
			forwarderName = p.readString(funcRVA)
		}

		// Calculate file offset
		var fileOffset uint64
		if !isForwarder {
			if off, ok := p.rvaToOffset(funcRVA); ok {
				fileOffset = uint64(off)
			}
		}

		// Emit function callback (skip forwarders)
		if p.cfg.OnFunctionFn != nil && !isForwarder && funcName != "" {
			f := epc.Function{
				Name:   funcName,
				Offset: fileOffset,
				Addr:   p.imageBase + uint64(funcRVA),
				Size:   0, // PE doesn't store function sizes
			}
			if err := p.cfg.OnFunctionFn(p.exeID, p.index, f); err != nil {
				return err
			}
		}

		// Emit exported code callback
		if p.cfg.OnExportedCodeFn != nil {
			var symType uint8 = 2 // STT_FUNC
			if isForwarder {
				symType = 0 // Forwarder
			}

			ec := epc.ExportedCode{
				Name:      funcName,
				Offset:    fileOffset,
				Addr:      p.imageBase + uint64(funcRVA),
				Size:      0, // PE doesn't store function sizes
				Type:      symType,
				Binding:   1, // STB_GLOBAL
				Forwarder: forwarderName,
			}
			if err := p.cfg.OnExportedCodeFn(p.exeID, p.index, ec); err != nil {
				return err
			}
		}

		// Emit symbol callback
		if p.cfg.OnSymbolFn != nil {
			name := funcName
			if name == "" {
				// No name, use ordinal
				name = ""
			}

			var symType uint8 = 2 // STT_FUNC
			if isForwarder {
				symType = 0 // Forwarder, mark as undefined/special
			}

			s := epc.Symbol{
				Name:    name,
				Offset:  fileOffset,
				Addr:    p.imageBase + uint64(funcRVA),
				Size:    0,
				Type:    symType,
				Binding: 1, // STB_GLOBAL
				Section: 0,
				Other:   uint8(biasedOrdinal & 0xFF), // Store ordinal in Other field
			}

			// For forwarders, we could store the forwarder name somehow
			_ = forwarderName

			if err := p.cfg.OnSymbolFn(p.exeID, p.index, s); err != nil {
				return err
			}
		}
	}

	return nil
}

// readRVATable reads an array of 32-bit RVAs from the given RVA.
func (p *Parser) readRVATable(rva uint32, count uint32) ([]uint32, error) {
	if rva == 0 || count == 0 {
		return nil, nil
	}

	offset, ok := p.rvaToOffset(rva)
	if !ok {
		return nil, nil
	}

	buf := make([]byte, count*4)
	if _, err := p.r.ReadAt(buf, int64(offset)); err != nil {
		return nil, err
	}

	result := make([]uint32, count)
	for i := uint32(0); i < count; i++ {
		result[i] = p.byteOrder.Uint32(buf[i*4 : i*4+4])
	}

	return result, nil
}

// readOrdinalTable reads an array of 16-bit ordinals from the given RVA.
func (p *Parser) readOrdinalTable(rva uint32, count uint32) ([]uint16, error) {
	if rva == 0 || count == 0 {
		return nil, nil
	}

	offset, ok := p.rvaToOffset(rva)
	if !ok {
		return nil, nil
	}

	buf := make([]byte, count*2)
	if _, err := p.r.ReadAt(buf, int64(offset)); err != nil {
		return nil, err
	}

	result := make([]uint16, count)
	for i := uint32(0); i < count; i++ {
		result[i] = p.byteOrder.Uint16(buf[i*2 : i*2+2])
	}

	return result, nil
}
