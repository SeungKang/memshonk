// Package epc provides common types for executable parsing configuration.
package epc

import "io"

// ExeFmtOption is a type alias for format-specific option keys.
type ExeFmtOption string

// Info describes general information about an executable.
type Info struct {
	Format     string // e.g., "ELF", "Mach-O", "PE"
	Class      uint8  // 32 or 64 bit
	Endian     string // "little" or "big"
	Type       string // e.g., "executable", "shared", "relocatable", "core"
	Machine    string // e.g., "x86_64", "arm64"
	EntryPoint uint64 // Virtual address of entry point
	OSABI      string // e.g., "FreeBSD", "Linux", "SysV"
	ABIVersion uint8
	Flags      uint32
}

// Function describes a function in an executable.
type Function struct {
	Name   string
	Offset uint64 // File offset
	Addr   uint64 // Virtual address
	Size   uint64
}

// ImportedLibrary describes an imported shared library dependency.
type ImportedLibrary struct {
	Name   string
	Offset uint64
}

// ImportedCode describes a function or symbol imported from an external library.
// These are undefined symbols that will be resolved at runtime by the dynamic linker.
type ImportedCode struct {
	Name    string // Symbol name (e.g., "printf", "malloc")
	Library string // Library name if known (e.g., "libc.so.6"), empty if unknown
	Offset  uint64 // File offset of the symbol table entry
	Type    uint8  // Symbol type (e.g., function, object)
	Binding uint8  // Symbol binding (e.g., global, weak)
}

// ExportedCode describes a function or symbol exported by this executable.
// These are defined symbols that can be used by other executables or libraries.
type ExportedCode struct {
	Name      string // Symbol name (e.g., "MyFunction")
	Offset    uint64 // File offset (if determinable)
	Addr      uint64 // Virtual address
	Size      uint64 // Size in bytes (0 if unknown)
	Type      uint8  // Symbol type (e.g., function, object)
	Binding   uint8  // Symbol binding (e.g., global, weak)
	Forwarder string // For PE: forwarded symbol (e.g., "NTDLL.RtlAllocateHeap"), empty if not forwarded
}

// Reloc describes a relocation entry.
type Reloc struct {
	Offset   uint64 // File offset or virtual address
	Type     uint32
	Symbol   string
	Addend   int64
	SymIndex uint32
}

// Section describes a section in an executable.
type Section struct {
	Name      string
	Type      uint32
	Flags     uint64
	Addr      uint64 // Virtual address
	Offset    uint64 // File offset
	Size      uint64
	Link      uint32
	Info      uint32
	Align     uint64
	EntSize   uint64
	IsCode    bool
	IsData    bool
	IsWritable bool
}

// Segment describes a segment (program header) in an executable.
type Segment struct {
	Type     uint32
	Flags    uint32
	Offset   uint64 // File offset
	VAddr    uint64 // Virtual address
	PAddr    uint64 // Physical address
	FileSize uint64
	MemSize  uint64
	Align    uint64
}

// String describes a string found in an executable.
type String struct {
	Value  string
	Offset uint64 // File offset
	Source string // e.g., "strtab", "dynstr", "rodata"
}

// Symbol describes a symbol in an executable.
type Symbol struct {
	Name    string
	Offset  uint64 // File offset (if determinable)
	Addr    uint64 // Virtual address
	Size    uint64
	Type    uint8
	Binding uint8
	Section uint16
	Other   uint8
}

// CallbackFn is the signature for all parser callbacks.
// exeID is "main" for the primary executable, or CPU architecture for fat binaries.
// index is the zero-based index of the executable being parsed.
type CallbackFn[T any] func(exeID string, index uint, object T) error

// ParserConfig configures the parsing behavior and callbacks.
type ParserConfig struct {
	// Src is the executable file to parse.
	Src io.ReaderAt

	// Callback function pointers. If nil, the callback is not invoked
	// and the parser may skip parsing the corresponding data.
	OnInfoFn              CallbackFn[Info]
	OnFunctionFn          CallbackFn[Function]
	OnExportedCodeFn      CallbackFn[ExportedCode]
	OnImportedCodeFn      CallbackFn[ImportedCode]
	OnImportedLibraryFn   CallbackFn[ImportedLibrary]
	OnRelocFn             CallbackFn[Reloc]
	OnSectionFn           CallbackFn[Section]
	OnSegmentFn           CallbackFn[Segment]
	OnStringFn            CallbackFn[String]
	OnSymbolFn            CallbackFn[Symbol]

	// OptCPU optionally overrides the target CPU type for fat executables.
	OptCPU string

	// OptBits optionally overrides the target CPU bit size for fat executables.
	OptBits uint8

	// OptExeSpecific contains additional format-specific options.
	OptExeSpecific map[ExeFmtOption]string
}
