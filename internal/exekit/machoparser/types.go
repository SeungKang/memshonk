package machoparser

// MachHeader32 is the 32-bit Mach-O header.
type MachHeader32 struct {
	Magic      uint32 // Mach magic number
	CPUType    int32  // CPU type
	CPUSubtype int32  // CPU subtype
	FileType   uint32 // Type of file
	NCmds      uint32 // Number of load commands
	SizeOfCmds uint32 // Size of load commands
	Flags      uint32 // Flags
}

// MachHeader64 is the 64-bit Mach-O header.
type MachHeader64 struct {
	Magic      uint32 // Mach magic number
	CPUType    int32  // CPU type
	CPUSubtype int32  // CPU subtype
	FileType   uint32 // Type of file
	NCmds      uint32 // Number of load commands
	SizeOfCmds uint32 // Size of load commands
	Flags      uint32 // Flags
	Reserved   uint32 // Reserved
}

// LoadCommand is the common header for all load commands.
type LoadCommand struct {
	Cmd     uint32 // Type of load command
	CmdSize uint32 // Total size of command
}

// SegmentCommand32 is a 32-bit segment load command.
type SegmentCommand32 struct {
	Cmd      uint32     // LC_SEGMENT
	CmdSize  uint32     // Size of this structure plus section structures
	SegName  [16]byte   // Segment name
	VMAddr   uint32     // Virtual memory address
	VMSize   uint32     // Virtual memory size
	FileOff  uint32     // File offset
	FileSize uint32     // File size
	MaxProt  int32      // Maximum VM protection
	InitProt int32      // Initial VM protection
	NSects   uint32     // Number of sections
	Flags    uint32     // Flags
}

// SegmentCommand64 is a 64-bit segment load command.
type SegmentCommand64 struct {
	Cmd      uint32     // LC_SEGMENT_64
	CmdSize  uint32     // Size of this structure plus section structures
	SegName  [16]byte   // Segment name
	VMAddr   uint64     // Virtual memory address
	VMSize   uint64     // Virtual memory size
	FileOff  uint64     // File offset
	FileSize uint64     // File size
	MaxProt  int32      // Maximum VM protection
	InitProt int32      // Initial VM protection
	NSects   uint32     // Number of sections
	Flags    uint32     // Flags
}

// Section32 is a 32-bit section structure.
type Section32 struct {
	SectName  [16]byte // Section name
	SegName   [16]byte // Segment name
	Addr      uint32   // Virtual memory address
	Size      uint32   // Size in bytes
	Offset    uint32   // File offset
	Align     uint32   // Alignment (power of 2)
	RelOff    uint32   // File offset of relocations
	NReloc    uint32   // Number of relocations
	Flags     uint32   // Section type and attributes
	Reserved1 uint32   // Reserved (for symbol stubs)
	Reserved2 uint32   // Reserved
}

// Section64 is a 64-bit section structure.
type Section64 struct {
	SectName  [16]byte // Section name
	SegName   [16]byte // Segment name
	Addr      uint64   // Virtual memory address
	Size      uint64   // Size in bytes
	Offset    uint32   // File offset
	Align     uint32   // Alignment (power of 2)
	RelOff    uint32   // File offset of relocations
	NReloc    uint32   // Number of relocations
	Flags     uint32   // Section type and attributes
	Reserved1 uint32   // Reserved (for symbol stubs)
	Reserved2 uint32   // Reserved
	Reserved3 uint32   // Reserved
}

// SymtabCommand describes the symbol table.
type SymtabCommand struct {
	Cmd     uint32 // LC_SYMTAB
	CmdSize uint32 // Size of this structure
	SymOff  uint32 // Offset to symbol table
	NSyms   uint32 // Number of symbols
	StrOff  uint32 // Offset to string table
	StrSize uint32 // Size of string table
}

// DysymtabCommand describes dynamic symbol table info.
type DysymtabCommand struct {
	Cmd            uint32 // LC_DYSYMTAB
	CmdSize        uint32 // Size of this structure
	ILocalSym      uint32 // Index of first local symbol
	NLocalSym      uint32 // Number of local symbols
	IExtDefSym     uint32 // Index of first external symbol
	NExtDefSym     uint32 // Number of external symbols
	IUndefSym      uint32 // Index of first undefined symbol
	NUndefSym      uint32 // Number of undefined symbols
	TOCOff         uint32 // Offset to TOC
	NTOC           uint32 // Number of TOC entries
	ModTabOff      uint32 // Offset to module table
	NModTab        uint32 // Number of module table entries
	ExtRefSymOff   uint32 // Offset to external reference table
	NExtRefSyms    uint32 // Number of external references
	IndirectSymOff uint32 // Offset to indirect symbol table
	NIndirectSyms  uint32 // Number of indirect symbols
	ExtRelOff      uint32 // Offset to external relocations
	NExtRel        uint32 // Number of external relocations
	LocRelOff      uint32 // Offset to local relocations
	NLocRel        uint32 // Number of local relocations
}

// DylibCommand describes a dynamic library dependency.
type DylibCommand struct {
	Cmd                  uint32 // LC_LOAD_DYLIB, LC_ID_DYLIB, etc.
	CmdSize              uint32 // Size of this structure
	NameOffset           uint32 // Offset to library name
	Timestamp            uint32 // Build timestamp
	CurrentVersion       uint32 // Current version
	CompatibilityVersion uint32 // Compatibility version
}

// DylinkerCommand describes the dynamic linker.
type DylinkerCommand struct {
	Cmd        uint32 // LC_LOAD_DYLINKER or LC_ID_DYLINKER
	CmdSize    uint32 // Size of this structure
	NameOffset uint32 // Offset to linker name
}

// UUIDCommand contains the UUID of the binary.
type UUIDCommand struct {
	Cmd     uint32   // LC_UUID
	CmdSize uint32   // Size of this structure
	UUID    [16]byte // 128-bit UUID
}

// EntryPointCommand describes the main entry point (LC_MAIN).
type EntryPointCommand struct {
	Cmd       uint32 // LC_MAIN
	CmdSize   uint32 // Size of this structure
	EntryOff  uint64 // File offset of main()
	StackSize uint64 // Initial stack size
}

// Nlist32 is a 32-bit symbol table entry.
type Nlist32 struct {
	NStrX  uint32 // Index into string table
	NType  uint8  // Type flag
	NSect  uint8  // Section number
	NDesc  int16  // Description
	NValue uint32 // Value
}

// Nlist64 is a 64-bit symbol table entry.
type Nlist64 struct {
	NStrX  uint32 // Index into string table
	NType  uint8  // Type flag
	NSect  uint8  // Section number
	NDesc  int16  // Description
	NValue uint64 // Value
}

// FatHeader is the header of a fat (universal) binary.
type FatHeader struct {
	Magic  uint32 // FAT_MAGIC or FAT_MAGIC_64
	NArch  uint32 // Number of architectures
}

// FatArch32 describes a 32-bit architecture in a fat binary.
type FatArch32 struct {
	CPUType    int32  // CPU type
	CPUSubtype int32  // CPU subtype
	Offset     uint32 // File offset to this architecture
	Size       uint32 // Size of this architecture
	Align      uint32 // Alignment (power of 2)
}

// FatArch64 describes a 64-bit architecture in a fat binary.
type FatArch64 struct {
	CPUType    int32  // CPU type
	CPUSubtype int32  // CPU subtype
	Offset     uint64 // File offset to this architecture
	Size       uint64 // Size of this architecture
	Align      uint32 // Alignment (power of 2)
	Reserved   uint32 // Reserved
}

// Helper functions

// SegmentName extracts the segment name from a byte array.
func SegmentName(name [16]byte) string {
	for i, b := range name {
		if b == 0 {
			return string(name[:i])
		}
	}
	return string(name[:])
}

// SectionName extracts the section name from a byte array.
func SectionName(name [16]byte) string {
	return SegmentName(name) // Same logic
}

// RelocationInfo describes a standard relocation entry.
// The structure is 8 bytes: 4-byte r_address, 4-byte packed bitfield.
type RelocationInfo struct {
	RAddress   int32  // Offset in section or from first segment
	RSymbolNum uint32 // Symbol index (if r_extern=1) or section number (if r_extern=0)
	RPCRel     bool   // PC-relative relocation
	RLength    uint8  // 0=1 byte, 1=2 bytes, 2=4 bytes, 3=8 bytes
	RExtern    bool   // External symbol (1) or section number (0)
	RType      uint8  // Relocation type (architecture-specific)
}

// ScatteredRelocationInfo describes a scattered relocation entry.
// Used when the relocation involves a non-zero constant or section difference.
type ScatteredRelocationInfo struct {
	RScattered bool   // Always true for scattered relocations
	RPCRel     bool   // PC-relative relocation
	RLength    uint8  // 0=1 byte, 1=2 bytes, 2=4 bytes
	RType      uint8  // Relocation type
	RAddress   uint32 // Offset in section (24 bits, max 0x00FFFFFF)
	RValue     int32  // Address of the relocatable expression
}
