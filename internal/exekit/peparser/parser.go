package peparser

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
	ErrInvalidDOSMagic = errors.New("invalid DOS magic number")
	ErrInvalidPEMagic  = errors.New("invalid PE signature")
	ErrInvalidOptMagic = errors.New("invalid optional header magic")
	ErrTruncated       = errors.New("PE file is truncated")
)

// sectionMapping maps RVA ranges to file offsets.
type sectionMapping struct {
	virtualAddr uint32
	virtualSize uint32
	rawAddr     uint32
	rawSize     uint32
}

// Parser holds the state for parsing a PE file.
type Parser struct {
	r         io.ReaderAt
	cfg       *epc.ParserConfig
	byteOrder binary.ByteOrder
	is64Bit   bool
	exeID     string
	index     uint

	coffHeader      COFFHeader
	optHeader32     OptionalHeader32
	optHeader64     OptionalHeader64
	sections        []SectionHeader
	dataDirectories []DataDirectory
	sectionMap      []sectionMapping

	// Cached values
	imageBase uint64
}

// Parse parses a PE file using the provided configuration.
func Parse(ctx context.Context, cfg *epc.ParserConfig) error {
	p := &Parser{
		r:         cfg.Src,
		cfg:       cfg,
		byteOrder: binary.LittleEndian, // PE is always little-endian
		exeID:     "main",
		index:     0,
	}
	return p.parse(ctx)
}

// parse performs the actual parsing.
func (p *Parser) parse(ctx context.Context) error {
	// Check for context cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Read and validate DOS header
	peOffset, err := p.readDOSHeader()
	if err != nil {
		return err
	}

	// Read and validate PE signature
	if err := p.readPESignature(peOffset); err != nil {
		return err
	}

	// Read COFF header (immediately after PE signature)
	coffOffset := peOffset + 4
	if err := p.readCOFFHeader(coffOffset); err != nil {
		return err
	}

	// Read optional header
	optOffset := coffOffset + 20 // COFF header is 20 bytes
	if err := p.readOptionalHeader(optOffset); err != nil {
		return err
	}

	// Calculate where section headers begin
	sectionOffset := optOffset + uint32(p.coffHeader.SizeOfOptionalHeader)

	// Read section headers
	if err := p.readSectionHeaders(sectionOffset); err != nil {
		return err
	}

	// Build section mapping for RVA conversion
	p.buildSectionMap()

	// Emit Info callback
	if err := p.emitInfo(); err != nil {
		return err
	}

	// Emit Section callbacks
	if err := p.emitSections(ctx); err != nil {
		return err
	}

	// Parse imports
	if err := p.parseImports(ctx); err != nil {
		return err
	}

	// Parse exports
	if err := p.parseExports(ctx); err != nil {
		return err
	}

	// Parse relocations
	if err := p.parseRelocations(ctx); err != nil {
		return err
	}

	// Parse debug info
	if err := p.parseDebugInfo(ctx); err != nil {
		return err
	}

	// Parse COFF string table
	if err := p.parseCOFFStrings(ctx); err != nil {
		return err
	}

	return nil
}

// readDOSHeader reads and validates the DOS header, returning the PE offset.
func (p *Parser) readDOSHeader() (uint32, error) {
	var buf [64]byte
	if _, err := p.r.ReadAt(buf[:], 0); err != nil {
		if errors.Is(err, io.EOF) {
			return 0, ErrTruncated
		}
		return 0, fmt.Errorf("reading DOS header: %w", err)
	}

	// Check DOS magic "MZ"
	magic := p.byteOrder.Uint16(buf[0:2])
	if magic != DOSMagic {
		return 0, ErrInvalidDOSMagic
	}

	// Get PE offset from e_lfanew at offset 0x3C
	peOffset := p.byteOrder.Uint32(buf[0x3C:0x40])
	return peOffset, nil
}

// readPESignature reads and validates the PE signature at the given offset.
func (p *Parser) readPESignature(offset uint32) error {
	var buf [4]byte
	if _, err := p.r.ReadAt(buf[:], int64(offset)); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrTruncated
		}
		return fmt.Errorf("reading PE signature: %w", err)
	}

	sig := p.byteOrder.Uint32(buf[:])
	if sig != PEMagic {
		return ErrInvalidPEMagic
	}
	return nil
}

// readCOFFHeader reads the COFF header at the given offset.
func (p *Parser) readCOFFHeader(offset uint32) error {
	var buf [20]byte
	if _, err := p.r.ReadAt(buf[:], int64(offset)); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrTruncated
		}
		return fmt.Errorf("reading COFF header: %w", err)
	}

	p.coffHeader = COFFHeader{
		Machine:              p.byteOrder.Uint16(buf[0:2]),
		NumberOfSections:     p.byteOrder.Uint16(buf[2:4]),
		TimeDateStamp:        p.byteOrder.Uint32(buf[4:8]),
		PointerToSymbolTable: p.byteOrder.Uint32(buf[8:12]),
		NumberOfSymbols:      p.byteOrder.Uint32(buf[12:16]),
		SizeOfOptionalHeader: p.byteOrder.Uint16(buf[16:18]),
		Characteristics:      p.byteOrder.Uint16(buf[18:20]),
	}
	return nil
}

// readOptionalHeader reads the optional header and data directories.
func (p *Parser) readOptionalHeader(offset uint32) error {
	// Read magic to determine PE32 vs PE32+
	var magicBuf [2]byte
	if _, err := p.r.ReadAt(magicBuf[:], int64(offset)); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrTruncated
		}
		return fmt.Errorf("reading optional header magic: %w", err)
	}

	magic := p.byteOrder.Uint16(magicBuf[:])

	switch magic {
	case PE32Magic:
		p.is64Bit = false
		if err := p.readOptionalHeader32(offset); err != nil {
			return err
		}
		p.imageBase = uint64(p.optHeader32.ImageBase)
	case PE32PMagic:
		p.is64Bit = true
		if err := p.readOptionalHeader64(offset); err != nil {
			return err
		}
		p.imageBase = p.optHeader64.ImageBase
	default:
		return ErrInvalidOptMagic
	}

	return nil
}

// readOptionalHeader32 reads a PE32 optional header.
func (p *Parser) readOptionalHeader32(offset uint32) error {
	// Standard fields are 28 bytes + Windows-specific 68 bytes = 96 bytes
	var buf [96]byte
	if _, err := p.r.ReadAt(buf[:], int64(offset)); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrTruncated
		}
		return fmt.Errorf("reading PE32 optional header: %w", err)
	}

	p.optHeader32 = OptionalHeader32{
		Magic:                       p.byteOrder.Uint16(buf[0:2]),
		MajorLinkerVersion:          buf[2],
		MinorLinkerVersion:          buf[3],
		SizeOfCode:                  p.byteOrder.Uint32(buf[4:8]),
		SizeOfInitializedData:       p.byteOrder.Uint32(buf[8:12]),
		SizeOfUninitializedData:     p.byteOrder.Uint32(buf[12:16]),
		AddressOfEntryPoint:         p.byteOrder.Uint32(buf[16:20]),
		BaseOfCode:                  p.byteOrder.Uint32(buf[20:24]),
		BaseOfData:                  p.byteOrder.Uint32(buf[24:28]),
		ImageBase:                   p.byteOrder.Uint32(buf[28:32]),
		SectionAlignment:            p.byteOrder.Uint32(buf[32:36]),
		FileAlignment:               p.byteOrder.Uint32(buf[36:40]),
		MajorOperatingSystemVersion: p.byteOrder.Uint16(buf[40:42]),
		MinorOperatingSystemVersion: p.byteOrder.Uint16(buf[42:44]),
		MajorImageVersion:           p.byteOrder.Uint16(buf[44:46]),
		MinorImageVersion:           p.byteOrder.Uint16(buf[46:48]),
		MajorSubsystemVersion:       p.byteOrder.Uint16(buf[48:50]),
		MinorSubsystemVersion:       p.byteOrder.Uint16(buf[50:52]),
		Win32VersionValue:           p.byteOrder.Uint32(buf[52:56]),
		SizeOfImage:                 p.byteOrder.Uint32(buf[56:60]),
		SizeOfHeaders:               p.byteOrder.Uint32(buf[60:64]),
		CheckSum:                    p.byteOrder.Uint32(buf[64:68]),
		Subsystem:                   p.byteOrder.Uint16(buf[68:70]),
		DllCharacteristics:          p.byteOrder.Uint16(buf[70:72]),
		SizeOfStackReserve:          p.byteOrder.Uint32(buf[72:76]),
		SizeOfStackCommit:           p.byteOrder.Uint32(buf[76:80]),
		SizeOfHeapReserve:           p.byteOrder.Uint32(buf[80:84]),
		SizeOfHeapCommit:            p.byteOrder.Uint32(buf[84:88]),
		LoaderFlags:                 p.byteOrder.Uint32(buf[88:92]),
		NumberOfRvaAndSizes:         p.byteOrder.Uint32(buf[92:96]),
	}

	// Read data directories
	return p.readDataDirectories(offset+96, p.optHeader32.NumberOfRvaAndSizes)
}

// readOptionalHeader64 reads a PE32+ optional header.
func (p *Parser) readOptionalHeader64(offset uint32) error {
	// Standard fields are 24 bytes + Windows-specific 88 bytes = 112 bytes
	var buf [112]byte
	if _, err := p.r.ReadAt(buf[:], int64(offset)); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrTruncated
		}
		return fmt.Errorf("reading PE32+ optional header: %w", err)
	}

	p.optHeader64 = OptionalHeader64{
		Magic:                       p.byteOrder.Uint16(buf[0:2]),
		MajorLinkerVersion:          buf[2],
		MinorLinkerVersion:          buf[3],
		SizeOfCode:                  p.byteOrder.Uint32(buf[4:8]),
		SizeOfInitializedData:       p.byteOrder.Uint32(buf[8:12]),
		SizeOfUninitializedData:     p.byteOrder.Uint32(buf[12:16]),
		AddressOfEntryPoint:         p.byteOrder.Uint32(buf[16:20]),
		BaseOfCode:                  p.byteOrder.Uint32(buf[20:24]),
		// No BaseOfData in PE32+
		ImageBase:                   p.byteOrder.Uint64(buf[24:32]),
		SectionAlignment:            p.byteOrder.Uint32(buf[32:36]),
		FileAlignment:               p.byteOrder.Uint32(buf[36:40]),
		MajorOperatingSystemVersion: p.byteOrder.Uint16(buf[40:42]),
		MinorOperatingSystemVersion: p.byteOrder.Uint16(buf[42:44]),
		MajorImageVersion:           p.byteOrder.Uint16(buf[44:46]),
		MinorImageVersion:           p.byteOrder.Uint16(buf[46:48]),
		MajorSubsystemVersion:       p.byteOrder.Uint16(buf[48:50]),
		MinorSubsystemVersion:       p.byteOrder.Uint16(buf[50:52]),
		Win32VersionValue:           p.byteOrder.Uint32(buf[52:56]),
		SizeOfImage:                 p.byteOrder.Uint32(buf[56:60]),
		SizeOfHeaders:               p.byteOrder.Uint32(buf[60:64]),
		CheckSum:                    p.byteOrder.Uint32(buf[64:68]),
		Subsystem:                   p.byteOrder.Uint16(buf[68:70]),
		DllCharacteristics:          p.byteOrder.Uint16(buf[70:72]),
		SizeOfStackReserve:          p.byteOrder.Uint64(buf[72:80]),
		SizeOfStackCommit:           p.byteOrder.Uint64(buf[80:88]),
		SizeOfHeapReserve:           p.byteOrder.Uint64(buf[88:96]),
		SizeOfHeapCommit:            p.byteOrder.Uint64(buf[96:104]),
		LoaderFlags:                 p.byteOrder.Uint32(buf[104:108]),
		NumberOfRvaAndSizes:         p.byteOrder.Uint32(buf[108:112]),
	}

	// Read data directories
	return p.readDataDirectories(offset+112, p.optHeader64.NumberOfRvaAndSizes)
}

// readDataDirectories reads the data directory table.
func (p *Parser) readDataDirectories(offset uint32, count uint32) error {
	if count > IMAGE_NUMBEROF_DIRECTORY_ENTRIES {
		count = IMAGE_NUMBEROF_DIRECTORY_ENTRIES
	}

	p.dataDirectories = make([]DataDirectory, count)
	buf := make([]byte, count*8)

	if _, err := p.r.ReadAt(buf, int64(offset)); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrTruncated
		}
		return fmt.Errorf("reading data directories: %w", err)
	}

	for i := uint32(0); i < count; i++ {
		p.dataDirectories[i] = DataDirectory{
			VirtualAddress: p.byteOrder.Uint32(buf[i*8 : i*8+4]),
			Size:           p.byteOrder.Uint32(buf[i*8+4 : i*8+8]),
		}
	}

	return nil
}

// readSectionHeaders reads all section headers.
func (p *Parser) readSectionHeaders(offset uint32) error {
	count := int(p.coffHeader.NumberOfSections)
	p.sections = make([]SectionHeader, count)
	buf := make([]byte, count*40)

	if _, err := p.r.ReadAt(buf, int64(offset)); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrTruncated
		}
		return fmt.Errorf("reading section headers: %w", err)
	}

	for i := 0; i < count; i++ {
		base := i * 40
		var name [8]byte
		copy(name[:], buf[base:base+8])

		p.sections[i] = SectionHeader{
			Name:                 name,
			VirtualSize:          p.byteOrder.Uint32(buf[base+8 : base+12]),
			VirtualAddress:       p.byteOrder.Uint32(buf[base+12 : base+16]),
			SizeOfRawData:        p.byteOrder.Uint32(buf[base+16 : base+20]),
			PointerToRawData:     p.byteOrder.Uint32(buf[base+20 : base+24]),
			PointerToRelocations: p.byteOrder.Uint32(buf[base+24 : base+28]),
			PointerToLineNumbers: p.byteOrder.Uint32(buf[base+28 : base+32]),
			NumberOfRelocations:  p.byteOrder.Uint16(buf[base+32 : base+34]),
			NumberOfLineNumbers:  p.byteOrder.Uint16(buf[base+34 : base+36]),
			Characteristics:      p.byteOrder.Uint32(buf[base+36 : base+40]),
		}
	}

	return nil
}

// buildSectionMap builds a mapping for RVA to file offset conversion.
func (p *Parser) buildSectionMap() {
	p.sectionMap = make([]sectionMapping, len(p.sections))
	for i, sec := range p.sections {
		p.sectionMap[i] = sectionMapping{
			virtualAddr: sec.VirtualAddress,
			virtualSize: sec.VirtualSize,
			rawAddr:     sec.PointerToRawData,
			rawSize:     sec.SizeOfRawData,
		}
	}
}

// rvaToOffset converts a Relative Virtual Address to a file offset.
// Returns 0 and false if the RVA is invalid or unmapped.
func (p *Parser) rvaToOffset(rva uint32) (uint32, bool) {
	for _, sec := range p.sectionMap {
		if rva >= sec.virtualAddr && rva < sec.virtualAddr+sec.virtualSize {
			offset := rva - sec.virtualAddr + sec.rawAddr
			if offset < sec.rawAddr+sec.rawSize {
				return offset, true
			}
			// RVA is in virtual space beyond raw data (zero-filled)
			return 0, false
		}
	}
	return 0, false
}

// readString reads a null-terminated string from an RVA.
func (p *Parser) readString(rva uint32) string {
	offset, ok := p.rvaToOffset(rva)
	if !ok {
		return ""
	}
	return p.readStringAt(int64(offset))
}

// readStringAt reads a null-terminated string from a file offset.
func (p *Parser) readStringAt(offset int64) string {
	var buf [256]byte
	n, err := p.r.ReadAt(buf[:], offset)
	if err != nil && n == 0 {
		return ""
	}

	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return string(buf[:i])
		}
	}
	return string(buf[:n])
}

// emitInfo emits the Info callback with executable information.
func (p *Parser) emitInfo() error {
	if p.cfg.OnInfoFn == nil {
		return nil
	}

	var class uint8 = 32
	if p.is64Bit {
		class = 64
	}

	// Determine executable type
	exeType := "executable"
	if p.coffHeader.Characteristics&IMAGE_FILE_DLL != 0 {
		exeType = "dll"
	}

	var entryPoint uint32
	var subsystem uint16
	if p.is64Bit {
		entryPoint = p.optHeader64.AddressOfEntryPoint
		subsystem = p.optHeader64.Subsystem
	} else {
		entryPoint = p.optHeader32.AddressOfEntryPoint
		subsystem = p.optHeader32.Subsystem
	}

	info := epc.Info{
		Format:     "PE",
		Class:      class,
		Endian:     "little",
		Type:       exeType,
		Machine:    MachineString(p.coffHeader.Machine),
		EntryPoint: p.imageBase + uint64(entryPoint),
		OSABI:      SubsystemString(subsystem),
		ABIVersion: 0,
		Flags:      uint32(p.coffHeader.Characteristics),
	}

	return p.cfg.OnInfoFn(p.exeID, p.index, info)
}

// emitSections emits Section callbacks for all sections.
func (p *Parser) emitSections(ctx context.Context) error {
	if p.cfg.OnSectionFn == nil {
		return nil
	}

	for _, sec := range p.sections {
		if err := ctx.Err(); err != nil {
			return err
		}

		section := epc.Section{
			Name:       SectionName(sec.Name),
			Type:       sec.Characteristics,
			Flags:      uint64(sec.Characteristics),
			Addr:       p.imageBase + uint64(sec.VirtualAddress),
			Offset:     uint64(sec.PointerToRawData),
			Size:       uint64(sec.VirtualSize),
			Link:       0,
			Info:       0,
			Align:      0, // Could calculate from SectionAlignment
			EntSize:    0,
			IsCode:     IsSectionCode(sec.Characteristics),
			IsData:     IsSectionData(sec.Characteristics),
			IsWritable: IsSectionWritable(sec.Characteristics),
		}

		if err := p.cfg.OnSectionFn(p.exeID, p.index, section); err != nil {
			return err
		}
	}

	return nil
}

// getDataDirectory returns a data directory entry, or empty if out of bounds.
func (p *Parser) getDataDirectory(index int) DataDirectory {
	if index >= len(p.dataDirectories) {
		return DataDirectory{}
	}
	return p.dataDirectories[index]
}
