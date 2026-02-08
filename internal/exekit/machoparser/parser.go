package machoparser

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/SeungKang/memshonk/internal/exekit/epc"
)

// Errors returned by the parser.
var (
	ErrInvalidMagic   = errors.New("invalid Mach-O magic number")
	ErrTruncated      = errors.New("Mach-O file is truncated")
	ErrInvalidLoadCmd = errors.New("invalid load command")
)

// Parser holds the state for parsing a Mach-O file.
type Parser struct {
	r         io.ReaderAt
	cfg       *epc.ParserConfig
	byteOrder binary.ByteOrder
	is64Bit   bool
	exeID     string
	index     uint
	baseOff   int64 // Base offset (for fat binaries)

	// Cached data for symbol resolution
	strtab    []byte   // String table
	strtabOff uint64   // String table file offset (for string reporting)
	dylibs    []string // Loaded dylib names (indexed by ordinal-1)
}

// Parse parses a Mach-O file using the provided configuration.
func Parse(ctx context.Context, cfg *epc.ParserConfig) error {
	return ParseAt(ctx, cfg, 0, "main", 0)
}

// ParseAt parses a Mach-O file at a given offset (for fat binary support).
func ParseAt(ctx context.Context, cfg *epc.ParserConfig, offset int64, exeID string, index uint) error {
	p := &Parser{
		r:       cfg.Src,
		cfg:     cfg,
		exeID:   exeID,
		index:   index,
		baseOff: offset,
		dylibs:  make([]string, 0),
	}
	return p.parse(ctx)
}

func (p *Parser) parse(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	// Read magic to determine endianness and bitness
	var magic uint32
	magicBuf := make([]byte, 4)
	if _, err := p.r.ReadAt(magicBuf, p.baseOff); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrTruncated
		}
		return fmt.Errorf("reading Mach-O magic: %w", err)
	}

	// Try little-endian first
	magic = binary.LittleEndian.Uint32(magicBuf)
	switch magic {
	case MH_MAGIC:
		p.byteOrder = binary.LittleEndian
		p.is64Bit = false
	case MH_MAGIC_64:
		p.byteOrder = binary.LittleEndian
		p.is64Bit = true
	case MH_CIGAM:
		p.byteOrder = binary.BigEndian
		p.is64Bit = false
	case MH_CIGAM_64:
		p.byteOrder = binary.BigEndian
		p.is64Bit = true
	default:
		return ErrInvalidMagic
	}

	if p.is64Bit {
		return p.parse64(ctx)
	}
	return p.parse32(ctx)
}

func (p *Parser) parse32(ctx context.Context) error {
	// Read 32-bit header (28 bytes)
	buf := make([]byte, 28)
	if _, err := p.r.ReadAt(buf, p.baseOff); err != nil {
		return fmt.Errorf("reading Mach-O 32-bit header: %w", err)
	}

	hdr := MachHeader32{
		Magic:      p.byteOrder.Uint32(buf[0:4]),
		CPUType:    int32(p.byteOrder.Uint32(buf[4:8])),
		CPUSubtype: int32(p.byteOrder.Uint32(buf[8:12])),
		FileType:   p.byteOrder.Uint32(buf[12:16]),
		NCmds:      p.byteOrder.Uint32(buf[16:20]),
		SizeOfCmds: p.byteOrder.Uint32(buf[20:24]),
		Flags:      p.byteOrder.Uint32(buf[24:28]),
	}

	// Report info
	if p.cfg.OnInfoFn != nil {
		info := epc.Info{
			Format:     "Mach-O",
			Class:      32,
			Endian:     endianString(p.byteOrder),
			Type:       fileTypeString(hdr.FileType),
			Machine:    cpuTypeString(hdr.CPUType),
			EntryPoint: 0, // Will be updated from LC_MAIN or LC_UNIXTHREAD
			OSABI:      "Darwin",
			Flags:      hdr.Flags,
		}
		if err := p.cfg.OnInfoFn(p.exeID, p.index, info); err != nil {
			return err
		}
	}

	// Parse load commands
	return p.parseLoadCommands(ctx, p.baseOff+28, hdr.NCmds, hdr.SizeOfCmds)
}

func (p *Parser) parse64(ctx context.Context) error {
	// Read 64-bit header (32 bytes)
	buf := make([]byte, 32)
	if _, err := p.r.ReadAt(buf, p.baseOff); err != nil {
		return fmt.Errorf("reading Mach-O 64-bit header: %w", err)
	}

	hdr := MachHeader64{
		Magic:      p.byteOrder.Uint32(buf[0:4]),
		CPUType:    int32(p.byteOrder.Uint32(buf[4:8])),
		CPUSubtype: int32(p.byteOrder.Uint32(buf[8:12])),
		FileType:   p.byteOrder.Uint32(buf[12:16]),
		NCmds:      p.byteOrder.Uint32(buf[16:20]),
		SizeOfCmds: p.byteOrder.Uint32(buf[20:24]),
		Flags:      p.byteOrder.Uint32(buf[24:28]),
		Reserved:   p.byteOrder.Uint32(buf[28:32]),
	}

	// Report info
	if p.cfg.OnInfoFn != nil {
		info := epc.Info{
			Format:     "Mach-O",
			Class:      64,
			Endian:     endianString(p.byteOrder),
			Type:       fileTypeString(hdr.FileType),
			Machine:    cpuTypeString(hdr.CPUType),
			EntryPoint: 0,
			OSABI:      "Darwin",
			Flags:      hdr.Flags,
		}
		if err := p.cfg.OnInfoFn(p.exeID, p.index, info); err != nil {
			return err
		}
	}

	// Parse load commands
	return p.parseLoadCommands(ctx, p.baseOff+32, hdr.NCmds, hdr.SizeOfCmds)
}

func (p *Parser) parseLoadCommands(ctx context.Context, offset int64, ncmds, sizeofcmds uint32) error {
	// First pass: collect dylib names for symbol library resolution
	// and load the string table
	if err := p.collectDylibs(ctx, offset, ncmds); err != nil {
		return err
	}

	// Second pass: parse all load commands
	cmdOffset := offset
	for i := uint32(0); i < ncmds; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Read load command header
		lcBuf := make([]byte, 8)
		if _, err := p.r.ReadAt(lcBuf, cmdOffset); err != nil {
			return fmt.Errorf("reading load command %d: %w", i, err)
		}

		cmd := p.byteOrder.Uint32(lcBuf[0:4])
		cmdSize := p.byteOrder.Uint32(lcBuf[4:8])

		if cmdSize < 8 {
			return ErrInvalidLoadCmd
		}

		// Read full command data
		cmdData := make([]byte, cmdSize)
		if _, err := p.r.ReadAt(cmdData, cmdOffset); err != nil {
			return fmt.Errorf("reading load command %d data: %w", i, err)
		}

		// Parse based on command type
		switch cmd {
		case LC_SEGMENT:
			if err := p.parseSegment32(ctx, cmdData); err != nil {
				return err
			}
		case LC_SEGMENT_64:
			if err := p.parseSegment64(ctx, cmdData); err != nil {
				return err
			}
		case LC_SYMTAB:
			if err := p.parseSymtab(ctx, cmdData); err != nil {
				return err
			}
			// Emit string table entries
			if err := p.emitStrings(ctx); err != nil {
				return err
			}
		}

		cmdOffset += int64(cmdSize)
	}

	return nil
}

func (p *Parser) collectDylibs(ctx context.Context, offset int64, ncmds uint32) error {
	cmdOffset := offset
	for i := uint32(0); i < ncmds; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		lcBuf := make([]byte, 8)
		if _, err := p.r.ReadAt(lcBuf, cmdOffset); err != nil {
			return err
		}

		cmd := p.byteOrder.Uint32(lcBuf[0:4])
		cmdSize := p.byteOrder.Uint32(lcBuf[4:8])

		if cmdSize < 8 {
			return ErrInvalidLoadCmd
		}

		// Collect dylib names
		switch cmd {
		case LC_LOAD_DYLIB, LC_LOAD_WEAK_DYLIB, LC_REEXPORT_DYLIB, LC_LAZY_LOAD_DYLIB, LC_LOAD_UPWARD_DYLIB:
			cmdData := make([]byte, cmdSize)
			if _, err := p.r.ReadAt(cmdData, cmdOffset); err != nil {
				return err
			}
			nameOffset := p.byteOrder.Uint32(cmdData[8:12])
			if nameOffset < cmdSize {
				name := p.readCString(cmdData[nameOffset:])
				p.dylibs = append(p.dylibs, name)

				// Report imported library
				if p.cfg.OnImportedLibraryFn != nil {
					lib := epc.ImportedLibrary{
						Name:   name,
						Offset: uint64(cmdOffset),
					}
					if err := p.cfg.OnImportedLibraryFn(p.exeID, p.index, lib); err != nil {
						return err
					}
				}
			}
		case LC_SYMTAB:
			// Load string table for symbol name resolution
			cmdData := make([]byte, cmdSize)
			if _, err := p.r.ReadAt(cmdData, cmdOffset); err != nil {
				return err
			}
			strOff := p.byteOrder.Uint32(cmdData[16:20])
			strSize := p.byteOrder.Uint32(cmdData[20:24])
			if strSize > 0 {
				p.strtab = make([]byte, strSize)
				p.strtabOff = uint64(p.baseOff) + uint64(strOff)
				if _, err := p.r.ReadAt(p.strtab, p.baseOff+int64(strOff)); err != nil {
					return fmt.Errorf("reading string table: %w", err)
				}
			}
		}

		cmdOffset += int64(cmdSize)
	}
	return nil
}

func (p *Parser) parseSegment32(ctx context.Context, data []byte) error {
	if len(data) < 56 {
		return ErrTruncated
	}

	var segName [16]byte
	copy(segName[:], data[8:24])

	seg := SegmentCommand32{
		Cmd:      p.byteOrder.Uint32(data[0:4]),
		CmdSize:  p.byteOrder.Uint32(data[4:8]),
		VMAddr:   p.byteOrder.Uint32(data[24:28]),
		VMSize:   p.byteOrder.Uint32(data[28:32]),
		FileOff:  p.byteOrder.Uint32(data[32:36]),
		FileSize: p.byteOrder.Uint32(data[36:40]),
		MaxProt:  int32(p.byteOrder.Uint32(data[40:44])),
		InitProt: int32(p.byteOrder.Uint32(data[44:48])),
		NSects:   p.byteOrder.Uint32(data[48:52]),
		Flags:    p.byteOrder.Uint32(data[52:56]),
	}
	seg.SegName = segName

	// Report segment
	if p.cfg.OnSegmentFn != nil {
		segment := epc.Segment{
			Type:     seg.Cmd,
			Flags:    uint32(seg.InitProt),
			Offset:   uint64(seg.FileOff),
			VAddr:    uint64(seg.VMAddr),
			PAddr:    0,
			FileSize: uint64(seg.FileSize),
			MemSize:  uint64(seg.VMSize),
			Align:    0,
		}
		if err := p.cfg.OnSegmentFn(p.exeID, p.index, segment); err != nil {
			return err
		}
	}

	// Parse sections
	sectOffset := 56
	for i := uint32(0); i < seg.NSects; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if sectOffset+68 > len(data) {
			return ErrTruncated
		}
		if err := p.parseSection32(ctx, data[sectOffset:sectOffset+68], SegmentName(segName)); err != nil {
			return err
		}
		sectOffset += 68
	}

	return nil
}

func (p *Parser) parseSegment64(ctx context.Context, data []byte) error {
	if len(data) < 72 {
		return ErrTruncated
	}

	var segName [16]byte
	copy(segName[:], data[8:24])

	seg := SegmentCommand64{
		Cmd:      p.byteOrder.Uint32(data[0:4]),
		CmdSize:  p.byteOrder.Uint32(data[4:8]),
		VMAddr:   p.byteOrder.Uint64(data[24:32]),
		VMSize:   p.byteOrder.Uint64(data[32:40]),
		FileOff:  p.byteOrder.Uint64(data[40:48]),
		FileSize: p.byteOrder.Uint64(data[48:56]),
		MaxProt:  int32(p.byteOrder.Uint32(data[56:60])),
		InitProt: int32(p.byteOrder.Uint32(data[60:64])),
		NSects:   p.byteOrder.Uint32(data[64:68]),
		Flags:    p.byteOrder.Uint32(data[68:72]),
	}
	seg.SegName = segName

	// Report segment
	if p.cfg.OnSegmentFn != nil {
		segment := epc.Segment{
			Type:     seg.Cmd,
			Flags:    uint32(seg.InitProt),
			Offset:   seg.FileOff,
			VAddr:    seg.VMAddr,
			PAddr:    0,
			FileSize: seg.FileSize,
			MemSize:  seg.VMSize,
			Align:    0,
		}
		if err := p.cfg.OnSegmentFn(p.exeID, p.index, segment); err != nil {
			return err
		}
	}

	// Parse sections
	sectOffset := 72
	for i := uint32(0); i < seg.NSects; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if sectOffset+80 > len(data) {
			return ErrTruncated
		}
		if err := p.parseSection64(ctx, data[sectOffset:sectOffset+80], SegmentName(segName)); err != nil {
			return err
		}
		sectOffset += 80
	}

	return nil
}

func (p *Parser) parseSection32(ctx context.Context, data []byte, segmentName string) error {
	var sectName, segName [16]byte
	copy(sectName[:], data[0:16])
	copy(segName[:], data[16:32])

	flags := p.byteOrder.Uint32(data[56:60])
	relOff := p.byteOrder.Uint32(data[48:52])
	nReloc := p.byteOrder.Uint32(data[52:56])
	name := SectionName(sectName)

	if p.cfg.OnSectionFn != nil {
		sec := epc.Section{
			Name:       name,
			Type:       flags & SECTION_TYPE,
			Flags:      uint64(flags),
			Addr:       uint64(p.byteOrder.Uint32(data[32:36])),
			Offset:     uint64(p.byteOrder.Uint32(data[40:44])),
			Size:       uint64(p.byteOrder.Uint32(data[36:40])),
			Align:      1 << p.byteOrder.Uint32(data[44:48]),
			IsCode:     flags&S_ATTR_PURE_INSTRUCTIONS != 0 || flags&S_ATTR_SOME_INSTRUCTIONS != 0,
			IsData:     flags&S_ATTR_PURE_INSTRUCTIONS == 0 && flags&S_ATTR_SOME_INSTRUCTIONS == 0,
			IsWritable: false, // Mach-O sections inherit from segment
		}
		if err := p.cfg.OnSectionFn(p.exeID, p.index, sec); err != nil {
			return err
		}
	}

	// Parse relocations for this section
	if nReloc > 0 {
		secReloc := SectionReloc{
			Name:   name,
			RelOff: relOff,
			NReloc: nReloc,
		}
		if err := p.parseRelocations(ctx, secReloc); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseSection64(ctx context.Context, data []byte, segmentName string) error {
	var sectName, segName [16]byte
	copy(sectName[:], data[0:16])
	copy(segName[:], data[16:32])

	flags := p.byteOrder.Uint32(data[56:60])
	relOff := p.byteOrder.Uint32(data[60:64])
	nReloc := p.byteOrder.Uint32(data[64:68])
	name := SectionName(sectName)

	if p.cfg.OnSectionFn != nil {
		sec := epc.Section{
			Name:       name,
			Type:       flags & SECTION_TYPE,
			Flags:      uint64(flags),
			Addr:       p.byteOrder.Uint64(data[32:40]),
			Offset:     uint64(p.byteOrder.Uint32(data[48:52])),
			Size:       p.byteOrder.Uint64(data[40:48]),
			Align:      1 << p.byteOrder.Uint32(data[52:56]),
			IsCode:     flags&S_ATTR_PURE_INSTRUCTIONS != 0 || flags&S_ATTR_SOME_INSTRUCTIONS != 0,
			IsData:     flags&S_ATTR_PURE_INSTRUCTIONS == 0 && flags&S_ATTR_SOME_INSTRUCTIONS == 0,
			IsWritable: false,
		}
		if err := p.cfg.OnSectionFn(p.exeID, p.index, sec); err != nil {
			return err
		}
	}

	// Parse relocations for this section
	if nReloc > 0 {
		secReloc := SectionReloc{
			Name:   name,
			RelOff: relOff,
			NReloc: nReloc,
		}
		if err := p.parseRelocations(ctx, secReloc); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseSymtab(ctx context.Context, data []byte) error {
	if len(data) < 24 {
		return ErrTruncated
	}

	symOff := p.byteOrder.Uint32(data[8:12])
	nSyms := p.byteOrder.Uint32(data[12:16])

	if nSyms == 0 {
		return nil
	}

	needSymbols := p.cfg.OnSymbolFn != nil
	needFunctions := p.cfg.OnFunctionFn != nil
	needImports := p.cfg.OnImportedCodeFn != nil
	needExports := p.cfg.OnExportedCodeFn != nil

	if !needSymbols && !needFunctions && !needImports && !needExports {
		return nil
	}

	if p.is64Bit {
		return p.parseSymbols64(ctx, symOff, nSyms)
	}
	return p.parseSymbols32(ctx, symOff, nSyms)
}

func (p *Parser) parseSymbols32(ctx context.Context, symOff, nSyms uint32) error {
	const symSize = 12 // Size of nlist (32-bit)
	buf := make([]byte, symSize)
	needExports := p.cfg.OnExportedCodeFn != nil

	for i := uint32(0); i < nSyms; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		offset := p.baseOff + int64(symOff) + int64(i)*symSize
		if _, err := p.r.ReadAt(buf, offset); err != nil {
			return fmt.Errorf("reading symbol %d: %w", i, err)
		}

		sym := Nlist32{
			NStrX:  p.byteOrder.Uint32(buf[0:4]),
			NType:  buf[4],
			NSect:  buf[5],
			NDesc:  int16(p.byteOrder.Uint16(buf[6:8])),
			NValue: p.byteOrder.Uint32(buf[8:12]),
		}

		name := p.getString(sym.NStrX)

		// Skip stab entries
		if sym.NType&N_STAB != 0 {
			continue
		}

		symType := sym.NType & N_TYPE
		isExternal := sym.NType&N_EXT != 0

		// Report symbol
		if p.cfg.OnSymbolFn != nil {
			s := epc.Symbol{
				Name:    name,
				Offset:  uint64(offset),
				Addr:    uint64(sym.NValue),
				Size:    0,
				Type:    symType,
				Binding: boolToBinding(isExternal),
				Section: uint16(sym.NSect),
				Other:   0,
			}
			if err := p.cfg.OnSymbolFn(p.exeID, p.index, s); err != nil {
				return err
			}
		}

		// Report function if defined in a section
		if p.cfg.OnFunctionFn != nil && symType == N_SECT && sym.NValue != 0 && isExternal {
			f := epc.Function{
				Name:   name,
				Offset: uint64(sym.NValue),
				Addr:   uint64(sym.NValue),
				Size:   0,
			}
			if err := p.cfg.OnFunctionFn(p.exeID, p.index, f); err != nil {
				return err
			}
		}

		// Report imported code if undefined
		if p.cfg.OnImportedCodeFn != nil && symType == N_UNDF && name != "" && isExternal {
			libOrdinal := GET_LIBRARY_ORDINAL(sym.NDesc)
			libName := p.getLibraryName(libOrdinal)

			ic := epc.ImportedCode{
				Name:    name,
				Library: libName,
				Offset:  uint64(offset),
				Type:    symType,
				Binding: boolToBinding(isExternal),
			}
			if err := p.cfg.OnImportedCodeFn(p.exeID, p.index, ic); err != nil {
				return err
			}
		}

		// Report exported code if defined in a section and external
		if needExports && symType == N_SECT && name != "" && isExternal {
			ec := epc.ExportedCode{
				Name:    name,
				Offset:  uint64(offset),
				Addr:    uint64(sym.NValue),
				Size:    0, // Mach-O doesn't store symbol sizes
				Type:    symType,
				Binding: boolToBinding(isExternal),
			}
			if err := p.cfg.OnExportedCodeFn(p.exeID, p.index, ec); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Parser) parseSymbols64(ctx context.Context, symOff, nSyms uint32) error {
	const symSize = 16 // Size of nlist_64
	buf := make([]byte, symSize)
	needExports := p.cfg.OnExportedCodeFn != nil

	for i := uint32(0); i < nSyms; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		offset := p.baseOff + int64(symOff) + int64(i)*symSize
		if _, err := p.r.ReadAt(buf, offset); err != nil {
			return fmt.Errorf("reading symbol %d: %w", i, err)
		}

		sym := Nlist64{
			NStrX:  p.byteOrder.Uint32(buf[0:4]),
			NType:  buf[4],
			NSect:  buf[5],
			NDesc:  int16(p.byteOrder.Uint16(buf[6:8])),
			NValue: p.byteOrder.Uint64(buf[8:16]),
		}

		name := p.getString(sym.NStrX)

		// Skip stab entries
		if sym.NType&N_STAB != 0 {
			continue
		}

		symType := sym.NType & N_TYPE
		isExternal := sym.NType&N_EXT != 0

		// Report symbol
		if p.cfg.OnSymbolFn != nil {
			s := epc.Symbol{
				Name:    name,
				Offset:  uint64(offset),
				Addr:    sym.NValue,
				Size:    0,
				Type:    symType,
				Binding: boolToBinding(isExternal),
				Section: uint16(sym.NSect),
				Other:   0,
			}
			if err := p.cfg.OnSymbolFn(p.exeID, p.index, s); err != nil {
				return err
			}
		}

		// Report function if defined in a section
		if p.cfg.OnFunctionFn != nil && symType == N_SECT && sym.NValue != 0 && isExternal {
			f := epc.Function{
				Name:   name,
				Offset: sym.NValue,
				Addr:   sym.NValue,
				Size:   0,
			}
			if err := p.cfg.OnFunctionFn(p.exeID, p.index, f); err != nil {
				return err
			}
		}

		// Report imported code if undefined
		if p.cfg.OnImportedCodeFn != nil && symType == N_UNDF && name != "" && isExternal {
			libOrdinal := GET_LIBRARY_ORDINAL(sym.NDesc)
			libName := p.getLibraryName(libOrdinal)

			ic := epc.ImportedCode{
				Name:    name,
				Library: libName,
				Offset:  uint64(offset),
				Type:    symType,
				Binding: boolToBinding(isExternal),
			}
			if err := p.cfg.OnImportedCodeFn(p.exeID, p.index, ic); err != nil {
				return err
			}
		}

		// Report exported code if defined in a section and external
		if needExports && symType == N_SECT && name != "" && isExternal {
			ec := epc.ExportedCode{
				Name:    name,
				Offset:  uint64(offset),
				Addr:    sym.NValue,
				Size:    0, // Mach-O doesn't store symbol sizes
				Type:    symType,
				Binding: boolToBinding(isExternal),
			}
			if err := p.cfg.OnExportedCodeFn(p.exeID, p.index, ec); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Parser) getString(idx uint32) string {
	if p.strtab == nil || idx >= uint32(len(p.strtab)) {
		return ""
	}
	end := idx
	for end < uint32(len(p.strtab)) && p.strtab[end] != 0 {
		end++
	}
	return string(p.strtab[idx:end])
}

func (p *Parser) getLibraryName(ordinal uint8) string {
	switch ordinal {
	case SELF_LIBRARY_ORDINAL:
		return ""
	case DYNAMIC_LOOKUP_ORDINAL:
		return "(dynamic lookup)"
	case EXECUTABLE_ORDINAL:
		return "(executable)"
	default:
		if int(ordinal) <= len(p.dylibs) && ordinal > 0 {
			return p.dylibs[ordinal-1]
		}
		return ""
	}
}

func (p *Parser) readCString(data []byte) string {
	for i, b := range data {
		if b == 0 {
			return string(data[:i])
		}
	}
	return string(data)
}

// emitStrings iterates through the string table and reports each string
// via the OnStringFn callback.
func (p *Parser) emitStrings(ctx context.Context) error {
	if p.cfg.OnStringFn == nil || p.strtab == nil || len(p.strtab) == 0 {
		return nil
	}

	i := uint64(0)
	for i < uint64(len(p.strtab)) {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Skip null bytes
		if p.strtab[i] == 0 {
			i++
			continue
		}

		// Find end of string
		start := i
		for i < uint64(len(p.strtab)) && p.strtab[i] != 0 {
			i++
		}

		str := string(p.strtab[start:i])
		if len(str) > 0 {
			s := epc.String{
				Value:  str,
				Offset: p.strtabOff + start,
				Source: "symtab",
			}
			if err := p.cfg.OnStringFn(p.exeID, p.index, s); err != nil {
				return err
			}
		}

		i++ // Skip null terminator
	}

	return nil
}

func boolToBinding(isExternal bool) uint8 {
	if isExternal {
		return 1 // Global
	}
	return 0 // Local
}

func endianString(order binary.ByteOrder) string {
	if order == binary.LittleEndian {
		return "little"
	}
	return "big"
}

func fileTypeString(ft uint32) string {
	switch ft {
	case MH_OBJECT:
		return "object"
	case MH_EXECUTE:
		return "executable"
	case MH_FVMLIB:
		return "fvmlib"
	case MH_CORE:
		return "core"
	case MH_PRELOAD:
		return "preload"
	case MH_DYLIB:
		return "dylib"
	case MH_DYLINKER:
		return "dylinker"
	case MH_BUNDLE:
		return "bundle"
	case MH_DYLIB_STUB:
		return "dylib_stub"
	case MH_DSYM:
		return "dsym"
	case MH_KEXT_BUNDLE:
		return "kext"
	default:
		return fmt.Sprintf("unknown(%d)", ft)
	}
}

func cpuTypeString(ct int32) string {
	switch ct {
	case CPU_TYPE_X86:
		return "i386"
	case CPU_TYPE_X86_64:
		return "x86_64"
	case CPU_TYPE_ARM:
		return "arm"
	case CPU_TYPE_ARM64:
		return "arm64"
	case CPU_TYPE_ARM64_32:
		return "arm64_32"
	case CPU_TYPE_POWERPC:
		return "ppc"
	case CPU_TYPE_POWERPC64:
		return "ppc64"
	default:
		return fmt.Sprintf("cpu(%d)", ct)
	}
}
