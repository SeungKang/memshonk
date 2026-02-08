package peparser

import "bytes"

// DOSHeader is the DOS MZ header at the beginning of every PE file.
// Only the fields we need are defined; the full header is 64 bytes.
type DOSHeader struct {
	Magic  uint16   // "MZ" (0x5A4D)
	_      [58]byte // Unused DOS header fields
	LfaNew uint32   // File offset to PE signature (at offset 0x3C)
}

// COFFHeader is the COFF file header following the PE signature.
type COFFHeader struct {
	Machine              uint16 // CPU type
	NumberOfSections     uint16 // Number of section headers
	TimeDateStamp        uint32 // Build timestamp
	PointerToSymbolTable uint32 // File offset to COFF symbol table (usually 0 for images)
	NumberOfSymbols      uint32 // Number of COFF symbols (usually 0 for images)
	SizeOfOptionalHeader uint16 // Size of optional header
	Characteristics      uint16 // File characteristics flags
}

// OptionalHeader32 is the PE32 optional header (32-bit).
type OptionalHeader32 struct {
	Magic                   uint16 // 0x10B for PE32
	MajorLinkerVersion      uint8
	MinorLinkerVersion      uint8
	SizeOfCode              uint32
	SizeOfInitializedData   uint32
	SizeOfUninitializedData uint32
	AddressOfEntryPoint     uint32 // RVA of entry point
	BaseOfCode              uint32
	BaseOfData              uint32 // PE32 only, not in PE32+
	ImageBase               uint32
	SectionAlignment        uint32
	FileAlignment           uint32
	MajorOperatingSystemVersion uint16
	MinorOperatingSystemVersion uint16
	MajorImageVersion       uint16
	MinorImageVersion       uint16
	MajorSubsystemVersion   uint16
	MinorSubsystemVersion   uint16
	Win32VersionValue       uint32 // Reserved, must be 0
	SizeOfImage             uint32
	SizeOfHeaders           uint32
	CheckSum                uint32
	Subsystem               uint16
	DllCharacteristics      uint16
	SizeOfStackReserve      uint32
	SizeOfStackCommit       uint32
	SizeOfHeapReserve       uint32
	SizeOfHeapCommit        uint32
	LoaderFlags             uint32 // Reserved, must be 0
	NumberOfRvaAndSizes     uint32
}

// OptionalHeader64 is the PE32+ optional header (64-bit).
type OptionalHeader64 struct {
	Magic                   uint16 // 0x20B for PE32+
	MajorLinkerVersion      uint8
	MinorLinkerVersion      uint8
	SizeOfCode              uint32
	SizeOfInitializedData   uint32
	SizeOfUninitializedData uint32
	AddressOfEntryPoint     uint32 // RVA of entry point
	BaseOfCode              uint32
	// No BaseOfData in PE32+
	ImageBase               uint64 // 64-bit image base
	SectionAlignment        uint32
	FileAlignment           uint32
	MajorOperatingSystemVersion uint16
	MinorOperatingSystemVersion uint16
	MajorImageVersion       uint16
	MinorImageVersion       uint16
	MajorSubsystemVersion   uint16
	MinorSubsystemVersion   uint16
	Win32VersionValue       uint32 // Reserved, must be 0
	SizeOfImage             uint32
	SizeOfHeaders           uint32
	CheckSum                uint32
	Subsystem               uint16
	DllCharacteristics      uint16
	SizeOfStackReserve      uint64 // 64-bit
	SizeOfStackCommit       uint64
	SizeOfHeapReserve       uint64
	SizeOfHeapCommit        uint64
	LoaderFlags             uint32 // Reserved, must be 0
	NumberOfRvaAndSizes     uint32
}

// DataDirectory represents an entry in the data directory table.
type DataDirectory struct {
	VirtualAddress uint32 // RVA of the table
	Size           uint32 // Size in bytes
}

// SectionHeader is the IMAGE_SECTION_HEADER structure (40 bytes).
type SectionHeader struct {
	Name                 [8]byte
	VirtualSize          uint32
	VirtualAddress       uint32
	SizeOfRawData        uint32
	PointerToRawData     uint32
	PointerToRelocations uint32
	PointerToLineNumbers uint32
	NumberOfRelocations  uint16
	NumberOfLineNumbers  uint16
	Characteristics      uint32
}

// ImportDescriptor is the IMAGE_IMPORT_DESCRIPTOR structure (20 bytes).
type ImportDescriptor struct {
	OriginalFirstThunk uint32 // RVA to Import Lookup Table (ILT)
	TimeDateStamp      uint32 // 0 unless bound
	ForwarderChain     uint32 // -1 if no forwarders
	Name               uint32 // RVA to DLL name
	FirstThunk         uint32 // RVA to Import Address Table (IAT)
}

// ExportDirectory is the IMAGE_EXPORT_DIRECTORY structure (40 bytes).
type ExportDirectory struct {
	Characteristics       uint32 // Reserved, must be 0
	TimeDateStamp         uint32
	MajorVersion          uint16
	MinorVersion          uint16
	Name                  uint32 // RVA to DLL name
	Base                  uint32 // Ordinal base
	NumberOfFunctions     uint32 // Number of entries in EAT
	NumberOfNames         uint32 // Number of entries in name pointer table
	AddressOfFunctions    uint32 // RVA to Export Address Table
	AddressOfNames        uint32 // RVA to Export Name Pointer Table
	AddressOfNameOrdinals uint32 // RVA to Ordinal Table
}

// BaseRelocation is the IMAGE_BASE_RELOCATION block header (8 bytes).
type BaseRelocation struct {
	VirtualAddress uint32 // Page RVA
	SizeOfBlock    uint32 // Block size including header
}

// DebugDirectory is the IMAGE_DEBUG_DIRECTORY structure (28 bytes).
type DebugDirectory struct {
	Characteristics  uint32 // Reserved, must be 0
	TimeDateStamp    uint32
	MajorVersion     uint16
	MinorVersion     uint16
	Type             uint32 // Debug type (e.g., IMAGE_DEBUG_TYPE_CODEVIEW)
	SizeOfData       uint32
	AddressOfRawData uint32 // RVA (may be 0)
	PointerToRawData uint32 // File offset
}

// TLSDirectory32 is the IMAGE_TLS_DIRECTORY32 structure.
type TLSDirectory32 struct {
	StartAddressOfRawData uint32
	EndAddressOfRawData   uint32
	AddressOfIndex        uint32
	AddressOfCallBacks    uint32
	SizeOfZeroFill        uint32
	Characteristics       uint32
}

// TLSDirectory64 is the IMAGE_TLS_DIRECTORY64 structure.
type TLSDirectory64 struct {
	StartAddressOfRawData uint64
	EndAddressOfRawData   uint64
	AddressOfIndex        uint64
	AddressOfCallBacks    uint64
	SizeOfZeroFill        uint32
	Characteristics       uint32
}

// ResourceDirectory is the IMAGE_RESOURCE_DIRECTORY structure.
type ResourceDirectory struct {
	Characteristics      uint32
	TimeDateStamp        uint32
	MajorVersion         uint16
	MinorVersion         uint16
	NumberOfNamedEntries uint16
	NumberOfIdEntries    uint16
}

// ResourceDirectoryEntry is a resource directory entry.
type ResourceDirectoryEntry struct {
	NameOrID         uint32 // High bit set = name offset, else ID
	OffsetToDataOrDir uint32 // High bit set = subdirectory offset, else data entry offset
}

// ResourceDataEntry is the IMAGE_RESOURCE_DATA_ENTRY structure.
type ResourceDataEntry struct {
	OffsetToData uint32
	Size         uint32
	CodePage     uint32
	Reserved     uint32
}

// SectionName extracts a null-terminated section name from the 8-byte field.
func SectionName(name [8]byte) string {
	// Find the null terminator
	n := bytes.IndexByte(name[:], 0)
	if n == -1 {
		n = 8
	}
	return string(name[:n])
}

// IsOrdinalImport32 checks if a 32-bit thunk imports by ordinal.
func IsOrdinalImport32(thunk uint32) bool {
	return thunk&0x80000000 != 0
}

// IsOrdinalImport64 checks if a 64-bit thunk imports by ordinal.
func IsOrdinalImport64(thunk uint64) bool {
	return thunk&0x8000000000000000 != 0
}

// Ordinal32 extracts the ordinal number from a 32-bit thunk.
func Ordinal32(thunk uint32) uint16 {
	return uint16(thunk & 0xFFFF)
}

// Ordinal64 extracts the ordinal number from a 64-bit thunk.
func Ordinal64(thunk uint64) uint16 {
	return uint16(thunk & 0xFFFF)
}

// HintNameRVA32 extracts the hint/name RVA from a 32-bit thunk.
func HintNameRVA32(thunk uint32) uint32 {
	return thunk & 0x7FFFFFFF
}

// HintNameRVA64 extracts the hint/name RVA from a 64-bit thunk.
func HintNameRVA64(thunk uint64) uint32 {
	return uint32(thunk & 0x7FFFFFFF)
}

// RelocType extracts the relocation type from a 16-bit entry (high 4 bits).
func RelocType(entry uint16) uint8 {
	return uint8(entry >> 12)
}

// RelocOffset extracts the page offset from a 16-bit entry (low 12 bits).
func RelocOffset(entry uint16) uint16 {
	return entry & 0x0FFF
}

// IsSectionCode returns true if the section contains executable code.
func IsSectionCode(characteristics uint32) bool {
	return characteristics&IMAGE_SCN_CNT_CODE != 0
}

// IsSectionData returns true if the section contains initialized data.
func IsSectionData(characteristics uint32) bool {
	return characteristics&IMAGE_SCN_CNT_INITIALIZED_DATA != 0
}

// IsSectionBSS returns true if the section contains uninitialized data.
func IsSectionBSS(characteristics uint32) bool {
	return characteristics&IMAGE_SCN_CNT_UNINITIALIZED_DATA != 0
}

// IsSectionExecutable returns true if the section is executable.
func IsSectionExecutable(characteristics uint32) bool {
	return characteristics&IMAGE_SCN_MEM_EXECUTE != 0
}

// IsSectionReadable returns true if the section is readable.
func IsSectionReadable(characteristics uint32) bool {
	return characteristics&IMAGE_SCN_MEM_READ != 0
}

// IsSectionWritable returns true if the section is writable.
func IsSectionWritable(characteristics uint32) bool {
	return characteristics&IMAGE_SCN_MEM_WRITE != 0
}

// COFFSymbol is the COFF symbol table entry (18 bytes).
// The Name field is a union: if the first 4 bytes are zero, bytes 4-7 are an
// offset into the string table. Otherwise, bytes 0-7 contain the name directly.
type COFFSymbol struct {
	Name               [8]byte // Symbol name or string table reference
	Value              uint32  // Symbol value (interpretation depends on section/class)
	SectionNumber      int16   // Section index (1-based) or special value
	Type               uint16  // Symbol type (LSB=base type, MSB=derived type)
	StorageClass       uint8   // Storage class
	NumberOfAuxSymbols uint8   // Number of auxiliary symbol records following
}

// COFFSymbolSize is the size of a COFF symbol table entry in bytes.
const COFFSymbolSize = 18

// SymbolName extracts the symbol name from a COFFSymbol.
// If the first 4 bytes are zero, it returns the string table offset.
// Otherwise, it returns the inline name (up to 8 bytes, null-terminated).
func (s *COFFSymbol) SymbolName(strtab []byte) string {
	// Check if name is in string table (first 4 bytes are zero)
	if s.Name[0] == 0 && s.Name[1] == 0 && s.Name[2] == 0 && s.Name[3] == 0 {
		// Offset is in bytes 4-7 (little-endian)
		offset := uint32(s.Name[4]) | uint32(s.Name[5])<<8 | uint32(s.Name[6])<<16 | uint32(s.Name[7])<<24
		if strtab == nil || int(offset) >= len(strtab) {
			return ""
		}
		// Find null terminator
		end := int(offset)
		for end < len(strtab) && strtab[end] != 0 {
			end++
		}
		return string(strtab[offset:end])
	}

	// Name is inline (up to 8 bytes)
	n := bytes.IndexByte(s.Name[:], 0)
	if n == -1 {
		n = 8
	}
	return string(s.Name[:n])
}

// IsFunction returns true if the symbol is a function.
func (s *COFFSymbol) IsFunction() bool {
	// MSB of Type is the derived type; 0x20 means function
	return s.Type&0xFF00 == 0x2000
}
