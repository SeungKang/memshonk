package peparser

import (
	"bytes"
	"context"
	"encoding/hex"
)

// DebugInfo contains parsed debug information from a PE file.
type DebugInfo struct {
	Type        uint32 // Debug type (e.g., IMAGE_DEBUG_TYPE_CODEVIEW)
	TypeName    string // Human-readable type name
	TimeDateStamp uint32
	MajorVersion uint16
	MinorVersion uint16
	PDBPath     string // Path to PDB file (for CodeView)
	GUID        string // PDB GUID as hex string (for CodeView 7.0)
	Signature   uint32 // PDB signature (for CodeView 2.0)
	Age         uint32 // PDB age
	FileOffset  uint32 // File offset of debug data
	DataSize    uint32 // Size of debug data
}

// parseDebugInfo parses the debug directory.
// Since there's no standard callback for debug info, this stores the info internally
// and can be accessed via GetDebugInfo() after parsing.
func (p *Parser) parseDebugInfo(ctx context.Context) error {
	dir := p.getDataDirectory(IMAGE_DIRECTORY_ENTRY_DEBUG)
	if dir.VirtualAddress == 0 || dir.Size == 0 {
		return nil
	}

	offset, ok := p.rvaToOffset(dir.VirtualAddress)
	if !ok {
		return nil
	}

	// Calculate number of debug directory entries (28 bytes each)
	const entrySize = 28
	numEntries := dir.Size / entrySize

	for i := uint32(0); i < numEntries; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		entryOffset := offset + i*entrySize

		// Read debug directory entry
		var buf [28]byte
		if _, err := p.r.ReadAt(buf[:], int64(entryOffset)); err != nil {
			return nil
		}

		entry := DebugDirectory{
			Characteristics:  p.byteOrder.Uint32(buf[0:4]),
			TimeDateStamp:    p.byteOrder.Uint32(buf[4:8]),
			MajorVersion:     p.byteOrder.Uint16(buf[8:10]),
			MinorVersion:     p.byteOrder.Uint16(buf[10:12]),
			Type:             p.byteOrder.Uint32(buf[12:16]),
			SizeOfData:       p.byteOrder.Uint32(buf[16:20]),
			AddressOfRawData: p.byteOrder.Uint32(buf[20:24]),
			PointerToRawData: p.byteOrder.Uint32(buf[24:28]),
		}

		// For CodeView entries, try to parse PDB info
		if entry.Type == IMAGE_DEBUG_TYPE_CODEVIEW && entry.SizeOfData > 0 && entry.PointerToRawData > 0 {
			p.parseCodeView(entry)
		}
	}

	return nil
}

// parseCodeView parses CodeView debug information.
func (p *Parser) parseCodeView(entry DebugDirectory) {
	if entry.SizeOfData < 4 {
		return
	}

	// Read signature to determine format
	var sigBuf [4]byte
	if _, err := p.r.ReadAt(sigBuf[:], int64(entry.PointerToRawData)); err != nil {
		return
	}

	sig := string(sigBuf[:])

	switch sig {
	case "RSDS":
		// PDB 7.0 format (CV_INFO_PDB70)
		p.parseCodeViewRSDS(entry)
	case "NB10":
		// PDB 2.0 format (CV_INFO_PDB20)
		p.parseCodeViewNB10(entry)
	}
}

// parseCodeViewRSDS parses RSDS (PDB 7.0) debug info.
// Format: signature[4] + GUID[16] + age[4] + PDB path (null-terminated)
func (p *Parser) parseCodeViewRSDS(entry DebugDirectory) *DebugInfo {
	if entry.SizeOfData < 24 { // 4 + 16 + 4 minimum
		return nil
	}

	buf := make([]byte, entry.SizeOfData)
	if _, err := p.r.ReadAt(buf, int64(entry.PointerToRawData)); err != nil {
		return nil
	}

	// Extract GUID (16 bytes at offset 4)
	guid := buf[4:20]

	// Format GUID as hex string (typical format: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX)
	guidStr := formatGUID(guid)

	// Extract age (4 bytes at offset 20)
	age := p.byteOrder.Uint32(buf[20:24])

	// Extract PDB path (null-terminated string starting at offset 24)
	pdbPath := ""
	if len(buf) > 24 {
		end := bytes.IndexByte(buf[24:], 0)
		if end == -1 {
			end = len(buf) - 24
		}
		pdbPath = string(buf[24 : 24+end])
	}

	return &DebugInfo{
		Type:          entry.Type,
		TypeName:      "CODEVIEW",
		TimeDateStamp: entry.TimeDateStamp,
		MajorVersion:  entry.MajorVersion,
		MinorVersion:  entry.MinorVersion,
		PDBPath:       pdbPath,
		GUID:          guidStr,
		Age:           age,
		FileOffset:    entry.PointerToRawData,
		DataSize:      entry.SizeOfData,
	}
}

// parseCodeViewNB10 parses NB10 (PDB 2.0) debug info.
// Format: signature[4] + offset[4] + signature[4] + age[4] + PDB path (null-terminated)
func (p *Parser) parseCodeViewNB10(entry DebugDirectory) *DebugInfo {
	if entry.SizeOfData < 16 { // 4 + 4 + 4 + 4 minimum
		return nil
	}

	buf := make([]byte, entry.SizeOfData)
	if _, err := p.r.ReadAt(buf, int64(entry.PointerToRawData)); err != nil {
		return nil
	}

	// Extract signature (4 bytes at offset 8)
	signature := p.byteOrder.Uint32(buf[8:12])

	// Extract age (4 bytes at offset 12)
	age := p.byteOrder.Uint32(buf[12:16])

	// Extract PDB path (null-terminated string starting at offset 16)
	pdbPath := ""
	if len(buf) > 16 {
		end := bytes.IndexByte(buf[16:], 0)
		if end == -1 {
			end = len(buf) - 16
		}
		pdbPath = string(buf[16 : 16+end])
	}

	return &DebugInfo{
		Type:          entry.Type,
		TypeName:      "CODEVIEW",
		TimeDateStamp: entry.TimeDateStamp,
		MajorVersion:  entry.MajorVersion,
		MinorVersion:  entry.MinorVersion,
		PDBPath:       pdbPath,
		Signature:     signature,
		Age:           age,
		FileOffset:    entry.PointerToRawData,
		DataSize:      entry.SizeOfData,
	}
}

// formatGUID formats a 16-byte GUID as a string.
// Format: XXXXXXXX-XXXX-XXXX-XXXX-XXXXXXXXXXXX
func formatGUID(guid []byte) string {
	if len(guid) != 16 {
		return hex.EncodeToString(guid)
	}

	// GUID is stored in mixed-endian format:
	// - Data1 (4 bytes) - little-endian
	// - Data2 (2 bytes) - little-endian
	// - Data3 (2 bytes) - little-endian
	// - Data4 (8 bytes) - big-endian

	return hex.EncodeToString([]byte{guid[3], guid[2], guid[1], guid[0]}) + "-" +
		hex.EncodeToString([]byte{guid[5], guid[4]}) + "-" +
		hex.EncodeToString([]byte{guid[7], guid[6]}) + "-" +
		hex.EncodeToString(guid[8:10]) + "-" +
		hex.EncodeToString(guid[10:16])
}

// DebugTypeString returns a human-readable string for a debug type.
func DebugTypeString(debugType uint32) string {
	switch debugType {
	case IMAGE_DEBUG_TYPE_UNKNOWN:
		return "UNKNOWN"
	case IMAGE_DEBUG_TYPE_COFF:
		return "COFF"
	case IMAGE_DEBUG_TYPE_CODEVIEW:
		return "CODEVIEW"
	case IMAGE_DEBUG_TYPE_FPO:
		return "FPO"
	case IMAGE_DEBUG_TYPE_MISC:
		return "MISC"
	case IMAGE_DEBUG_TYPE_EXCEPTION:
		return "EXCEPTION"
	case IMAGE_DEBUG_TYPE_FIXUP:
		return "FIXUP"
	case IMAGE_DEBUG_TYPE_OMAP_TO_SRC:
		return "OMAP_TO_SRC"
	case IMAGE_DEBUG_TYPE_OMAP_FROM_SRC:
		return "OMAP_FROM_SRC"
	case IMAGE_DEBUG_TYPE_BORLAND:
		return "BORLAND"
	case IMAGE_DEBUG_TYPE_RESERVED10:
		return "RESERVED10"
	case IMAGE_DEBUG_TYPE_CLSID:
		return "CLSID"
	case IMAGE_DEBUG_TYPE_REPRO:
		return "REPRO"
	case IMAGE_DEBUG_TYPE_EMBEDDED_PDB:
		return "EMBEDDED_PDB"
	case IMAGE_DEBUG_TYPE_PDBCHECKSUM:
		return "PDBCHECKSUM"
	case IMAGE_DEBUG_TYPE_EX_DLLCHARACTERISTICS:
		return "EX_DLLCHARACTERISTICS"
	default:
		return "UNKNOWN"
	}
}
