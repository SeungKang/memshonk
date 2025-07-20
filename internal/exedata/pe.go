package exedata

import (
	"debug/pe"
	"encoding/binary"
	"fmt"
	"io"
)

var _ Exe = (*Pe)(nil)

func ParsePe(readerAt io.ReaderAt, options ParserOptions) (Pe, error) {
	peFile, err := pe.NewFile(readerAt)
	if err != nil {
		return Pe{}, fmt.Errorf("failed to create new pe file object - %w", err)
	}

	exports, err := peEports(peFile)
	if err != nil {
		return Pe{}, fmt.Errorf("failed to parse pe exports - %w", err)
	}

	syms := make([]Symbol, len(exports))

	for i, export := range exports {
		syms[i] = Symbol{
			Name:     export.Name,
			Location: uint64(export.VirtualAddress),
			Size:     0, // TOOD
		}
	}

	return Pe{
		peFile: peFile,
		syms:   syms,
	}, nil
}

type Pe struct {
	peFile *pe.File
	syms   []Symbol
}

func (o Pe) Symbols() []Symbol {
	return o.syms
}

// PeExportDirectory - data directory definition for exported functions
//
// Based on work by GitHub user awgh:
// https://github.com/Binject/debug/blob/26db73212a7af7e35ad01ad63061fbc6503ca9d4/pe/exports.go
type PeExportDirectory struct {
	ExportFlags       uint32 // reserved, must be zero
	TimeDateStamp     uint32
	MajorVersion      uint16
	MinorVersion      uint16
	NameRVA           uint32 // pointer to the name of the DLL
	OrdinalBase       uint32
	NumberOfFunctions uint32
	NumberOfNames     uint32 // also Ordinal Table Len
	AddressTableAddr  uint32 // RVA of EAT, relative to image base
	NameTableAddr     uint32 // RVA of export name pointer table, relative to image base
	OrdinalTableAddr  uint32 // address of the ordinal table, relative to iamge base

	DllName string
}

// PeExport - describes a single export entry
//
// Based on work by GitHub user awgh:
// https://github.com/Binject/debug/blob/26db73212a7af7e35ad01ad63061fbc6503ca9d4/pe/exports.go
type PeExport struct {
	Ordinal        uint32
	Name           string
	VirtualAddress uint32
	Forward        string
}

// peExports - gets exports
//
// Based on work by GitHub user awgh:
// https://github.com/Binject/debug/blob/26db73212a7af7e35ad01ad63061fbc6503ca9d4/pe/exports.go
func peEports(f *pe.File) ([]PeExport, error) {
	// grab the number of data directory entries
	var ddLength uint32

	var dataDirectories []pe.DataDirectory

	switch optHeader := f.OptionalHeader.(type) {
	case *pe.OptionalHeader32:
		ddLength = optHeader.NumberOfRvaAndSizes
		dataDirectories = optHeader.DataDirectory[:]
	case *pe.OptionalHeader64:
		ddLength = optHeader.NumberOfRvaAndSizes
		dataDirectories = optHeader.DataDirectory[:]
	default:
		return nil, fmt.Errorf("unsupported pe optional header type: %T", optHeader)
	}

	// check that the length of data directory entries is large
	// enough to include the exports directory.
	if ddLength < pe.IMAGE_DIRECTORY_ENTRY_EXPORT+1 {
		return nil, nil
	}

	edd := dataDirectories[pe.IMAGE_DIRECTORY_ENTRY_EXPORT]

	// figure out which section contains the export directory table
	var ds *pe.Section

	for _, s := range f.Sections {
		if s.VirtualAddress <= edd.VirtualAddress && edd.VirtualAddress < s.VirtualAddress+s.VirtualSize {
			ds = s
			break
		}
	}

	// didn't find a section, so no exports were found
	if ds == nil {
		return nil, nil
	}

	d, err := ds.Data()
	if err != nil {
		return nil, err
	}

	exportDirOffset := edd.VirtualAddress - ds.VirtualAddress

	// seek to the virtual address specified in the export data directory
	dxd := d[exportDirOffset:]

	// deserialize export directory
	var dt PeExportDirectory
	dt.ExportFlags = binary.LittleEndian.Uint32(dxd[0:4])
	dt.TimeDateStamp = binary.LittleEndian.Uint32(dxd[4:8])
	dt.MajorVersion = binary.LittleEndian.Uint16(dxd[8:10])
	dt.MinorVersion = binary.LittleEndian.Uint16(dxd[10:12])
	dt.NameRVA = binary.LittleEndian.Uint32(dxd[12:16])
	dt.OrdinalBase = binary.LittleEndian.Uint32(dxd[16:20])
	dt.NumberOfFunctions = binary.LittleEndian.Uint32(dxd[20:24])
	dt.NumberOfNames = binary.LittleEndian.Uint32(dxd[24:28])
	dt.AddressTableAddr = binary.LittleEndian.Uint32(dxd[28:32])
	dt.NameTableAddr = binary.LittleEndian.Uint32(dxd[32:36])
	dt.OrdinalTableAddr = binary.LittleEndian.Uint32(dxd[36:40])

	dt.DllName, _ = getString(d, int(dt.NameRVA-ds.VirtualAddress))

	ordinalTable := make(map[uint16]uint32)
	if dt.OrdinalTableAddr > ds.VirtualAddress && dt.NameTableAddr > ds.VirtualAddress {
		// seek to ordinal table
		dno := d[dt.OrdinalTableAddr-ds.VirtualAddress:]
		// seek to names table
		dnn := d[dt.NameTableAddr-ds.VirtualAddress:]

		// build whole ordinal->name table
		for n := uint32(0); n < dt.NumberOfNames; n++ {
			ord := binary.LittleEndian.Uint16(dno[n*2 : (n*2)+2])
			nameRVA := binary.LittleEndian.Uint32(dnn[n*4 : (n*4)+4])
			ordinalTable[ord] = nameRVA
		}
		dno = nil
		dnn = nil
	}

	// seek to ordinal table
	dna := d[dt.AddressTableAddr-ds.VirtualAddress:]

	var exports []PeExport
	for i := uint32(0); i < dt.NumberOfFunctions; i++ {
		var export PeExport
		export.VirtualAddress =
			binary.LittleEndian.Uint32(dna[i*4 : (i*4)+4])
		export.Ordinal = dt.OrdinalBase + i

		// if this address is inside the export section, this export is a Forwarder RVA
		if ds.VirtualAddress <= export.VirtualAddress &&
			export.VirtualAddress < ds.VirtualAddress+ds.VirtualSize {
			export.Forward, _ = getString(d, int(export.VirtualAddress-ds.VirtualAddress))
		}

		// check the entire ordinal table looking for this index to see if we have a name
		_, ok := ordinalTable[uint16(i)]
		if ok { // a name exists for this exported function
			nameRVA, _ := ordinalTable[uint16(i)]
			export.Name, _ = getString(d, int(nameRVA-ds.VirtualAddress))
		}
		exports = append(exports, export)
	}

	return exports, nil
}

// getString extracts a string from symbol string table.
//
// Copied from src/debug/pe
func getString(section []byte, start int) (string, bool) {
	if start < 0 || start >= len(section) {
		return "", false
	}

	for end := start; end < len(section); end++ {
		if section[end] == 0 {
			return string(section[start:end]), true
		}
	}
	return "", false
}
