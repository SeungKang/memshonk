// Package machoparser provides Mach-O executable file format parsing.
package machoparser

// Mach-O magic numbers
const (
	MH_MAGIC    = 0xfeedface // 32-bit, native endian
	MH_CIGAM    = 0xcefaedfe // 32-bit, swapped endian
	MH_MAGIC_64 = 0xfeedfacf // 64-bit, native endian
	MH_CIGAM_64 = 0xcffaedfe // 64-bit, swapped endian
)

// Fat binary magic numbers
const (
	FAT_MAGIC    = 0xcafebabe // Fat binary, big-endian
	FAT_CIGAM    = 0xbebafeca // Fat binary, little-endian
	FAT_MAGIC_64 = 0xcafebabf // Fat binary 64-bit, big-endian
	FAT_CIGAM_64 = 0xbfbafeca // Fat binary 64-bit, little-endian
)

// CPU types
const (
	CPU_TYPE_ANY       = -1
	CPU_TYPE_VAX       = 1
	CPU_TYPE_MC680x0   = 6
	CPU_TYPE_X86       = 7
	CPU_TYPE_I386      = CPU_TYPE_X86
	CPU_TYPE_X86_64    = CPU_TYPE_X86 | CPU_ARCH_ABI64
	CPU_TYPE_MC98000   = 10
	CPU_TYPE_HPPA      = 11
	CPU_TYPE_ARM       = 12
	CPU_TYPE_ARM64     = CPU_TYPE_ARM | CPU_ARCH_ABI64
	CPU_TYPE_ARM64_32  = CPU_TYPE_ARM | CPU_ARCH_ABI64_32
	CPU_TYPE_MC88000   = 13
	CPU_TYPE_SPARC     = 14
	CPU_TYPE_I860      = 15
	CPU_TYPE_POWERPC   = 18
	CPU_TYPE_POWERPC64 = CPU_TYPE_POWERPC | CPU_ARCH_ABI64
)

// CPU architecture ABI masks
const (
	CPU_ARCH_MASK     = 0xff000000
	CPU_ARCH_ABI64    = 0x01000000
	CPU_ARCH_ABI64_32 = 0x02000000
)

// CPU subtypes for X86
const (
	CPU_SUBTYPE_X86_ALL   = 3
	CPU_SUBTYPE_X86_64_ALL = 3
	CPU_SUBTYPE_X86_ARCH1 = 4
	CPU_SUBTYPE_X86_64_H  = 8 // Haswell feature subset
)

// CPU subtypes for ARM
const (
	CPU_SUBTYPE_ARM_ALL    = 0
	CPU_SUBTYPE_ARM_V4T    = 5
	CPU_SUBTYPE_ARM_V6     = 6
	CPU_SUBTYPE_ARM_V5TEJ  = 7
	CPU_SUBTYPE_ARM_XSCALE = 8
	CPU_SUBTYPE_ARM_V7     = 9
	CPU_SUBTYPE_ARM_V7F    = 10
	CPU_SUBTYPE_ARM_V7S    = 11
	CPU_SUBTYPE_ARM_V7K    = 12
	CPU_SUBTYPE_ARM_V8     = 13
	CPU_SUBTYPE_ARM_V6M    = 14
	CPU_SUBTYPE_ARM_V7M    = 15
	CPU_SUBTYPE_ARM_V7EM   = 16
)

// CPU subtypes for ARM64
const (
	CPU_SUBTYPE_ARM64_ALL = 0
	CPU_SUBTYPE_ARM64_V8  = 1
	CPU_SUBTYPE_ARM64E    = 2
)

// CPU subtypes for PowerPC
const (
	CPU_SUBTYPE_POWERPC_ALL   = 0
	CPU_SUBTYPE_POWERPC_601   = 1
	CPU_SUBTYPE_POWERPC_602   = 2
	CPU_SUBTYPE_POWERPC_603   = 3
	CPU_SUBTYPE_POWERPC_603e  = 4
	CPU_SUBTYPE_POWERPC_603ev = 5
	CPU_SUBTYPE_POWERPC_604   = 6
	CPU_SUBTYPE_POWERPC_604e  = 7
	CPU_SUBTYPE_POWERPC_620   = 8
	CPU_SUBTYPE_POWERPC_750   = 9
	CPU_SUBTYPE_POWERPC_7400  = 10
	CPU_SUBTYPE_POWERPC_7450  = 11
	CPU_SUBTYPE_POWERPC_970   = 100
)

// File types
const (
	MH_OBJECT      = 0x1 // Relocatable object file
	MH_EXECUTE     = 0x2 // Demand paged executable file
	MH_FVMLIB      = 0x3 // Fixed VM shared library file
	MH_CORE        = 0x4 // Core file
	MH_PRELOAD     = 0x5 // Preloaded executable file
	MH_DYLIB       = 0x6 // Dynamically bound shared library
	MH_DYLINKER    = 0x7 // Dynamic link editor
	MH_BUNDLE      = 0x8 // Dynamically bound bundle file
	MH_DYLIB_STUB  = 0x9 // Shared library stub
	MH_DSYM        = 0xa // Companion debug-only file
	MH_KEXT_BUNDLE = 0xb // Kernel extension
)

// Header flags
const (
	MH_NOUNDEFS                = 0x00000001
	MH_INCRLINK                = 0x00000002
	MH_DYLDLINK                = 0x00000004
	MH_BINDATLOAD              = 0x00000008
	MH_PREBOUND                = 0x00000010
	MH_SPLIT_SEGS              = 0x00000020
	MH_LAZY_INIT               = 0x00000040
	MH_TWOLEVEL                = 0x00000080
	MH_FORCE_FLAT              = 0x00000100
	MH_NOMULTIDEFS             = 0x00000200
	MH_NOFIXPREBINDING         = 0x00000400
	MH_PREBINDABLE             = 0x00000800
	MH_ALLMODSBOUND            = 0x00001000
	MH_SUBSECTIONS_VIA_SYMBOLS = 0x00002000
	MH_CANONICAL               = 0x00004000
	MH_WEAK_DEFINES            = 0x00008000
	MH_BINDS_TO_WEAK           = 0x00010000
	MH_ALLOW_STACK_EXECUTION   = 0x00020000
	MH_ROOT_SAFE               = 0x00040000
	MH_SETUID_SAFE             = 0x00080000
	MH_NO_REEXPORTED_DYLIBS    = 0x00100000
	MH_PIE                     = 0x00200000
	MH_DEAD_STRIPPABLE_DYLIB   = 0x00400000
	MH_HAS_TLV_DESCRIPTORS     = 0x00800000
	MH_NO_HEAP_EXECUTION       = 0x01000000
	MH_APP_EXTENSION_SAFE      = 0x02000000
)

// Load command types
const (
	LC_REQ_DYLD = 0x80000000 // Required by dynamic linker

	LC_SEGMENT              = 0x1
	LC_SYMTAB               = 0x2
	LC_SYMSEG               = 0x3
	LC_THREAD               = 0x4
	LC_UNIXTHREAD           = 0x5
	LC_LOADFVMLIB           = 0x6
	LC_IDFVMLIB             = 0x7
	LC_IDENT                = 0x8
	LC_FVMFILE              = 0x9
	LC_PREPAGE              = 0xa
	LC_DYSYMTAB             = 0xb
	LC_LOAD_DYLIB           = 0xc
	LC_ID_DYLIB             = 0xd
	LC_LOAD_DYLINKER        = 0xe
	LC_ID_DYLINKER          = 0xf
	LC_PREBOUND_DYLIB       = 0x10
	LC_ROUTINES             = 0x11
	LC_SUB_FRAMEWORK        = 0x12
	LC_SUB_UMBRELLA         = 0x13
	LC_SUB_CLIENT           = 0x14
	LC_SUB_LIBRARY          = 0x15
	LC_TWOLEVEL_HINTS       = 0x16
	LC_PREBIND_CKSUM        = 0x17
	LC_LOAD_WEAK_DYLIB      = 0x18 | LC_REQ_DYLD
	LC_SEGMENT_64           = 0x19
	LC_ROUTINES_64          = 0x1a
	LC_UUID                 = 0x1b
	LC_RPATH                = 0x1c | LC_REQ_DYLD
	LC_CODE_SIGNATURE       = 0x1d
	LC_SEGMENT_SPLIT_INFO   = 0x1e
	LC_REEXPORT_DYLIB       = 0x1f | LC_REQ_DYLD
	LC_LAZY_LOAD_DYLIB      = 0x20
	LC_ENCRYPTION_INFO      = 0x21
	LC_DYLD_INFO            = 0x22
	LC_DYLD_INFO_ONLY       = 0x22 | LC_REQ_DYLD
	LC_LOAD_UPWARD_DYLIB    = 0x23 | LC_REQ_DYLD
	LC_VERSION_MIN_MACOSX   = 0x24
	LC_VERSION_MIN_IPHONEOS = 0x25
	LC_FUNCTION_STARTS      = 0x26
	LC_DYLD_ENVIRONMENT     = 0x27
	LC_MAIN                 = 0x28 | LC_REQ_DYLD
	LC_DATA_IN_CODE         = 0x29
	LC_SOURCE_VERSION       = 0x2a
	LC_DYLIB_CODE_SIGN_DRS  = 0x2b
	LC_ENCRYPTION_INFO_64   = 0x2c
	LC_LINKER_OPTION        = 0x2d
	LC_LINKER_OPTIMIZATION_HINT = 0x2e
	LC_VERSION_MIN_TVOS     = 0x2f
	LC_VERSION_MIN_WATCHOS  = 0x30
	LC_NOTE                 = 0x31
	LC_BUILD_VERSION        = 0x32
)

// Segment flags
const (
	SG_HIGHVM  = 0x1 // High part of VM space
	SG_FVMLIB  = 0x2 // FVM library segment
	SG_NORELOC = 0x4 // No relocation in this segment
)

// Section types (low 8 bits of flags)
const (
	S_REGULAR                    = 0x0
	S_ZEROFILL                   = 0x1
	S_CSTRING_LITERALS           = 0x2
	S_4BYTE_LITERALS             = 0x3
	S_8BYTE_LITERALS             = 0x4
	S_LITERAL_POINTERS           = 0x5
	S_NON_LAZY_SYMBOL_POINTERS   = 0x6
	S_LAZY_SYMBOL_POINTERS       = 0x7
	S_SYMBOL_STUBS               = 0x8
	S_MOD_INIT_FUNC_POINTERS     = 0x9
	S_MOD_TERM_FUNC_POINTERS     = 0xa
	S_COALESCED                  = 0xb
	S_GB_ZEROFILL                = 0xc
	S_INTERPOSING                = 0xd
	S_16BYTE_LITERALS            = 0xe
	S_DTRACE_DOF                 = 0xf
	S_LAZY_DYLIB_SYMBOL_POINTERS = 0x10
	S_THREAD_LOCAL_REGULAR       = 0x11
	S_THREAD_LOCAL_ZEROFILL      = 0x12
	S_THREAD_LOCAL_VARIABLES     = 0x13
	S_THREAD_LOCAL_VARIABLE_POINTERS = 0x14
	S_THREAD_LOCAL_INIT_FUNCTION_POINTERS = 0x15
)

// Section attributes (high 24 bits of flags)
const (
	SECTION_TYPE       = 0x000000ff
	SECTION_ATTRIBUTES = 0xffffff00

	S_ATTR_PURE_INSTRUCTIONS   = 0x80000000
	S_ATTR_NO_TOC              = 0x40000000
	S_ATTR_STRIP_STATIC_SYMS   = 0x20000000
	S_ATTR_NO_DEAD_STRIP       = 0x10000000
	S_ATTR_LIVE_SUPPORT        = 0x08000000
	S_ATTR_SELF_MODIFYING_CODE = 0x04000000
	S_ATTR_DEBUG               = 0x02000000
	S_ATTR_SOME_INSTRUCTIONS   = 0x00000400
	S_ATTR_EXT_RELOC           = 0x00000200
	S_ATTR_LOC_RELOC           = 0x00000100
)

// VM protection flags
const (
	VM_PROT_NONE    = 0x00
	VM_PROT_READ    = 0x01
	VM_PROT_WRITE   = 0x02
	VM_PROT_EXECUTE = 0x04
)

// Symbol n_type masks
const (
	N_STAB = 0xe0 // Stab symbol
	N_PEXT = 0x10 // Private external
	N_TYPE = 0x0e // Type mask
	N_EXT  = 0x01 // External symbol
)

// Symbol types (n_type & N_TYPE)
const (
	N_UNDF = 0x0 // Undefined
	N_ABS  = 0x2 // Absolute
	N_SECT = 0xe // Defined in section
	N_PBUD = 0xc // Prebound undefined
	N_INDR = 0xa // Indirect
)

// Reference types for n_desc
const (
	REFERENCE_TYPE                         = 0xf
	REFERENCE_FLAG_UNDEFINED_NON_LAZY      = 0x0
	REFERENCE_FLAG_UNDEFINED_LAZY          = 0x1
	REFERENCE_FLAG_DEFINED                 = 0x2
	REFERENCE_FLAG_PRIVATE_DEFINED         = 0x3
	REFERENCE_FLAG_PRIVATE_UNDEFINED_NON_LAZY = 0x4
	REFERENCE_FLAG_PRIVATE_UNDEFINED_LAZY  = 0x5
)

// Additional n_desc flags
const (
	REFERENCED_DYNAMICALLY = 0x0010
	N_NO_DEAD_STRIP        = 0x0020
	N_DESC_DISCARDED       = 0x0020
	N_WEAK_REF             = 0x0040
	N_WEAK_DEF             = 0x0080
)

// Library ordinal special values
const (
	SELF_LIBRARY_ORDINAL   = 0x0
	MAX_LIBRARY_ORDINAL    = 0xfd
	DYNAMIC_LOOKUP_ORDINAL = 0xfe
	EXECUTABLE_ORDINAL     = 0xff
)

// Special section index
const (
	NO_SECT = 0
)

// GET_LIBRARY_ORDINAL extracts the library ordinal from n_desc
func GET_LIBRARY_ORDINAL(n_desc int16) uint8 {
	return uint8((uint16(n_desc) >> 8) & 0xff)
}

// Relocation constants

// R_SCATTERED indicates a scattered relocation entry (high bit set in r_address)
const R_SCATTERED = 0x80000000

// R_ABS indicates an absolute symbol (r_symbolnum value for no relocation needed)
const R_ABS = 0

// Generic relocation types (x86)
const (
	GENERIC_RELOC_VANILLA        = 0 // Generic relocation
	GENERIC_RELOC_PAIR           = 1 // Second entry of a pair
	GENERIC_RELOC_SECTDIFF       = 2 // Section difference
	GENERIC_RELOC_PB_LA_PTR      = 3 // Prebound lazy pointer
	GENERIC_RELOC_LOCAL_SECTDIFF = 4 // Local section difference
	GENERIC_RELOC_TLV            = 5 // Thread local variable
)

// x86_64 relocation types
const (
	X86_64_RELOC_UNSIGNED   = 0 // Absolute address
	X86_64_RELOC_SIGNED     = 1 // Signed 32-bit displacement
	X86_64_RELOC_BRANCH     = 2 // CALL/JMP with 32-bit displacement
	X86_64_RELOC_GOT_LOAD   = 3 // MOVQ load of a GOT entry
	X86_64_RELOC_GOT        = 4 // Other GOT references
	X86_64_RELOC_SUBTRACTOR = 5 // Must be followed by X86_64_RELOC_UNSIGNED
	X86_64_RELOC_SIGNED_1   = 6 // Signed 32-bit displacement with -1 addend
	X86_64_RELOC_SIGNED_2   = 7 // Signed 32-bit displacement with -2 addend
	X86_64_RELOC_SIGNED_4   = 8 // Signed 32-bit displacement with -4 addend
	X86_64_RELOC_TLV        = 9 // Thread local variable
)

// ARM relocation types
const (
	ARM_RELOC_VANILLA        = 0 // Generic relocation
	ARM_RELOC_PAIR           = 1 // Second entry of a pair
	ARM_RELOC_SECTDIFF       = 2 // Section difference
	ARM_RELOC_LOCAL_SECTDIFF = 3 // Local section difference
	ARM_RELOC_PB_LA_PTR      = 4 // Prebound lazy pointer
	ARM_RELOC_BR24           = 5 // 24-bit branch
	ARM_THUMB_RELOC_BR22     = 6 // Thumb 22-bit branch
	ARM_THUMB_32BIT_BRANCH   = 7 // Thumb 32-bit branch
	ARM_RELOC_HALF           = 8 // Half (MOVW/MOVT)
	ARM_RELOC_HALF_SECTDIFF  = 9 // Half section difference
)

// ARM64 relocation types
const (
	ARM64_RELOC_UNSIGNED      = 0  // Pointer-sized relocation
	ARM64_RELOC_SUBTRACTOR    = 1  // Must be followed by ARM64_RELOC_UNSIGNED
	ARM64_RELOC_BRANCH26      = 2  // 26-bit PC-relative branch (B/BL)
	ARM64_RELOC_PAGE21        = 3  // 21-bit page offset (ADRP)
	ARM64_RELOC_PAGEOFF12     = 4  // 12-bit page offset
	ARM64_RELOC_GOT_LOAD_PAGE21    = 5  // GOT page offset (ADRP)
	ARM64_RELOC_GOT_LOAD_PAGEOFF12 = 6  // GOT page offset (LDR)
	ARM64_RELOC_POINTER_TO_GOT     = 7  // Pointer to GOT entry
	ARM64_RELOC_TLVP_LOAD_PAGE21   = 8  // TLV page offset (ADRP)
	ARM64_RELOC_TLVP_LOAD_PAGEOFF12 = 9 // TLV page offset (LDR)
	ARM64_RELOC_ADDEND             = 10 // Addend for subsequent reloc
)

// PowerPC relocation types
const (
	PPC_RELOC_VANILLA        = 0  // Generic relocation
	PPC_RELOC_PAIR           = 1  // Second entry of a pair
	PPC_RELOC_BR14           = 2  // 14-bit branch displacement
	PPC_RELOC_BR24           = 3  // 24-bit branch displacement
	PPC_RELOC_HI16           = 4  // High 16 bits
	PPC_RELOC_LO16           = 5  // Low 16 bits
	PPC_RELOC_HA16           = 6  // High 16 bits adjusted
	PPC_RELOC_LO14           = 7  // Low 14 bits
	PPC_RELOC_SECTDIFF       = 8  // Section difference
	PPC_RELOC_PB_LA_PTR      = 9  // Prebound lazy pointer
	PPC_RELOC_HI16_SECTDIFF  = 10 // High 16 bits section difference
	PPC_RELOC_LO16_SECTDIFF  = 11 // Low 16 bits section difference
	PPC_RELOC_HA16_SECTDIFF  = 12 // High 16 bits adjusted section difference
	PPC_RELOC_JBSR           = 13 // jbsr relocation
	PPC_RELOC_LO14_SECTDIFF  = 14 // Low 14 bits section difference
	PPC_RELOC_LOCAL_SECTDIFF = 15 // Local section difference
)
