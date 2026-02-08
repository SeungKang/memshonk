// Package elfparser provides ELF executable file format parsing.
package elfparser

// ELF Magic bytes
const (
	ELFMAG0 = 0x7f // e_ident[EI_MAG0]
	ELFMAG1 = 'E'  // e_ident[EI_MAG1]
	ELFMAG2 = 'L'  // e_ident[EI_MAG2]
	ELFMAG3 = 'F'  // e_ident[EI_MAG3]
)

// e_ident indices
const (
	EI_MAG0       = 0  // Magic number byte 0
	EI_MAG1       = 1  // Magic number byte 1
	EI_MAG2       = 2  // Magic number byte 2
	EI_MAG3       = 3  // Magic number byte 3
	EI_CLASS      = 4  // File class
	EI_DATA       = 5  // Data encoding
	EI_VERSION    = 6  // File version
	EI_OSABI      = 7  // OS/ABI identification
	EI_ABIVERSION = 8  // ABI version
	EI_PAD        = 9  // Start of padding bytes
	EI_NIDENT     = 16 // Size of e_ident[]
)

// ELF Class (e_ident[EI_CLASS])
const (
	ELFCLASSNONE = 0 // Invalid class
	ELFCLASS32   = 1 // 32-bit objects
	ELFCLASS64   = 2 // 64-bit objects
)

// Data encoding (e_ident[EI_DATA])
const (
	ELFDATANONE = 0 // Invalid data encoding
	ELFDATA2LSB = 1 // Little-endian
	ELFDATA2MSB = 2 // Big-endian
)

// ELF Version
const (
	EV_NONE    = 0 // Invalid version
	EV_CURRENT = 1 // Current version
)

// OS/ABI (e_ident[EI_OSABI])
const (
	ELFOSABI_SYSV       = 0   // UNIX System V ABI
	ELFOSABI_HPUX       = 1   // HP-UX
	ELFOSABI_NETBSD     = 2   // NetBSD
	ELFOSABI_LINUX      = 3   // GNU/Linux
	ELFOSABI_HURD       = 4   // GNU/Hurd
	ELFOSABI_86OPEN     = 5   // 86Open
	ELFOSABI_SOLARIS    = 6   // Solaris
	ELFOSABI_MONTEREY   = 7   // Monterey
	ELFOSABI_IRIX       = 8   // IRIX
	ELFOSABI_FREEBSD    = 9   // FreeBSD
	ELFOSABI_TRU64      = 10  // TRU64 UNIX
	ELFOSABI_MODESTO    = 11  // Novell Modesto
	ELFOSABI_OPENBSD    = 12  // OpenBSD
	ELFOSABI_OPENVMS    = 13  // OpenVMS
	ELFOSABI_NSK        = 14  // HP Non-Stop Kernel
	ELFOSABI_AROS       = 15  // AROS
	ELFOSABI_FENIXOS    = 16  // FenixOS
	ELFOSABI_ARM        = 97  // ARM
	ELFOSABI_STANDALONE = 255 // Standalone (embedded)
)

// Object file type (e_type)
const (
	ET_NONE   = 0      // No file type
	ET_REL    = 1      // Relocatable file
	ET_EXEC   = 2      // Executable file
	ET_DYN    = 3      // Shared object file
	ET_CORE   = 4      // Core file
	ET_LOOS   = 0xfe00 // OS-specific range start
	ET_HIOS   = 0xfeff // OS-specific range end
	ET_LOPROC = 0xff00 // Processor-specific range start
	ET_HIPROC = 0xffff // Processor-specific range end
)

// Machine types (e_machine)
const (
	EM_NONE        = 0   // No machine
	EM_M32         = 1   // AT&T WE 32100
	EM_SPARC       = 2   // SPARC
	EM_386         = 3   // Intel 80386
	EM_68K         = 4   // Motorola 68000
	EM_88K         = 5   // Motorola 88000
	EM_486         = 6   // Intel 80486
	EM_860         = 7   // Intel 80860
	EM_MIPS        = 8   // MIPS RS3000
	EM_S370        = 9   // IBM System/370
	EM_MIPS_RS4_BE = 10  // MIPS RS4000 big-endian
	EM_PARISC      = 15  // HP PA-RISC
	EM_SPARC32PLUS = 18  // SPARC v8plus
	EM_PPC         = 20  // PowerPC
	EM_PPC64       = 21  // PowerPC 64-bit
	EM_S390        = 22  // IBM S/390
	EM_ARM         = 40  // ARM
	EM_SPARCV9     = 43  // SPARC v9 64-bit
	EM_IA_64       = 50  // Intel IA-64
	EM_X86_64      = 62  // AMD x86-64
	EM_AARCH64     = 183 // ARM 64-bit
	EM_RISCV       = 243 // RISC-V
)

// Special section indices
const (
	SHN_UNDEF     = 0      // Undefined section
	SHN_LORESERVE = 0xff00 // Start of reserved indices
	SHN_LOPROC    = 0xff00 // Start of processor-specific
	SHN_HIPROC    = 0xff1f // End of processor-specific
	SHN_LOOS      = 0xff20 // Start of OS-specific
	SHN_HIOS      = 0xff3f // End of OS-specific
	SHN_ABS       = 0xfff1 // Absolute values
	SHN_COMMON    = 0xfff2 // Common symbols
	SHN_XINDEX    = 0xffff // Extended section index
	SHN_HIRESERVE = 0xffff // End of reserved indices
)

// Section types (sh_type)
const (
	SHT_NULL          = 0          // Inactive section
	SHT_PROGBITS      = 1          // Program-defined information
	SHT_SYMTAB        = 2          // Symbol table
	SHT_STRTAB        = 3          // String table
	SHT_RELA          = 4          // Relocation entries with addends
	SHT_HASH          = 5          // Symbol hash table
	SHT_DYNAMIC       = 6          // Dynamic linking information
	SHT_NOTE          = 7          // Notes
	SHT_NOBITS        = 8          // Uninitialized space
	SHT_REL           = 9          // Relocation entries without addends
	SHT_SHLIB         = 10         // Reserved
	SHT_DYNSYM        = 11         // Dynamic linking symbol table
	SHT_INIT_ARRAY    = 14         // Array of constructors
	SHT_FINI_ARRAY    = 15         // Array of destructors
	SHT_PREINIT_ARRAY = 16         // Array of pre-constructors
	SHT_GROUP         = 17         // Section group
	SHT_SYMTAB_SHNDX  = 18         // Extended section indices
	SHT_LOOS          = 0x60000000 // Start of OS-specific
	SHT_GNU_HASH      = 0x6ffffff6 // GNU hash table
	SHT_HIOS          = 0x6fffffff // End of OS-specific
	SHT_LOPROC        = 0x70000000 // Start of processor-specific
	SHT_HIPROC        = 0x7fffffff // End of processor-specific
	SHT_LOUSER        = 0x80000000 // Start of application-specific
	SHT_HIUSER        = 0xffffffff // End of application-specific
)

// Section flags (sh_flags)
const (
	SHF_WRITE            = 0x1        // Writable
	SHF_ALLOC            = 0x2        // Occupies memory during execution
	SHF_EXECINSTR        = 0x4        // Executable
	SHF_MERGE            = 0x10       // Might be merged
	SHF_STRINGS          = 0x20       // Contains nul-terminated strings
	SHF_INFO_LINK        = 0x40       // sh_info contains SHT index
	SHF_LINK_ORDER       = 0x80       // Preserve order after combining
	SHF_OS_NONCONFORMING = 0x100      // Non-standard OS handling required
	SHF_GROUP            = 0x200      // Section is member of a group
	SHF_TLS              = 0x400      // Section holds thread-local data
	SHF_COMPRESSED       = 0x800      // Section is compressed
	SHF_MASKOS           = 0x0ff00000 // OS-specific
	SHF_MASKPROC         = 0xf0000000 // Processor-specific
)

// Program header types (p_type)
const (
	PT_NULL    = 0          // Unused entry
	PT_LOAD    = 1          // Loadable segment
	PT_DYNAMIC = 2          // Dynamic linking information
	PT_INTERP  = 3          // Interpreter pathname
	PT_NOTE    = 4          // Auxiliary information
	PT_SHLIB   = 5          // Reserved
	PT_PHDR    = 6          // Program header table
	PT_TLS     = 7          // Thread-local storage
	PT_LOOS    = 0x60000000 // Start of OS-specific
	PT_GNU_EH_FRAME = 0x6474e550 // GCC .eh_frame_hdr segment
	PT_GNU_STACK    = 0x6474e551 // Stack executability
	PT_GNU_RELRO    = 0x6474e552 // Read-only after relocation
	PT_HIOS    = 0x6fffffff // End of OS-specific
	PT_LOPROC  = 0x70000000 // Start of processor-specific
	PT_HIPROC  = 0x7fffffff // End of processor-specific
)

// Program header flags (p_flags)
const (
	PF_X        = 0x1        // Execute permission
	PF_W        = 0x2        // Write permission
	PF_R        = 0x4        // Read permission
	PF_MASKOS   = 0x0ff00000 // OS-specific
	PF_MASKPROC = 0xf0000000 // Processor-specific
)

// Symbol binding (st_info high 4 bits)
const (
	STB_LOCAL  = 0  // Local symbol
	STB_GLOBAL = 1  // Global symbol
	STB_WEAK   = 2  // Weak symbol
	STB_LOOS   = 10 // Start of OS-specific
	STB_HIOS   = 12 // End of OS-specific
	STB_LOPROC = 13 // Start of processor-specific
	STB_HIPROC = 15 // End of processor-specific
)

// Symbol types (st_info low 4 bits)
const (
	STT_NOTYPE  = 0  // Symbol type not specified
	STT_OBJECT  = 1  // Data object
	STT_FUNC    = 2  // Function
	STT_SECTION = 3  // Section
	STT_FILE    = 4  // Source file
	STT_COMMON  = 5  // Uninitialized common block
	STT_TLS     = 6  // TLS object
	STT_LOOS    = 10 // Start of OS-specific
	STT_HIOS    = 12 // End of OS-specific
	STT_LOPROC  = 13 // Start of processor-specific
	STT_HIPROC  = 15 // End of processor-specific
)

// Symbol visibility (st_other low 2 bits)
const (
	STV_DEFAULT   = 0 // Default visibility
	STV_INTERNAL  = 1 // Internal visibility
	STV_HIDDEN    = 2 // Hidden visibility
	STV_PROTECTED = 3 // Protected visibility
)

// Dynamic array tags (d_tag)
const (
	DT_NULL     = 0  // Marks end of dynamic section
	DT_NEEDED   = 1  // Name of needed library
	DT_PLTRELSZ = 2  // Size of PLT relocations
	DT_PLTGOT   = 3  // Address of PLT and/or GOT
	DT_HASH     = 4  // Address of symbol hash table
	DT_STRTAB   = 5  // Address of string table
	DT_SYMTAB   = 6  // Address of symbol table
	DT_RELA     = 7  // Address of Rela relocations
	DT_RELASZ   = 8  // Total size of Rela relocations
	DT_RELAENT  = 9  // Size of one Rela relocation
	DT_STRSZ    = 10 // Size of string table
	DT_SYMENT   = 11 // Size of one symbol table entry
	DT_INIT     = 12 // Address of init function
	DT_FINI     = 13 // Address of finalization function
	DT_SONAME   = 14 // Name of shared object
	DT_RPATH    = 15 // Library search path (deprecated)
	DT_SYMBOLIC = 16 // Start symbol search in shared object
	DT_REL      = 17 // Address of Rel relocations
	DT_RELSZ    = 18 // Total size of Rel relocations
	DT_RELENT   = 19 // Size of one Rel relocation
	DT_PLTREL   = 20 // Type of PLT relocations
	DT_DEBUG    = 21 // Reserved for debugging
	DT_TEXTREL  = 22 // Text relocations exist
	DT_JMPREL   = 23 // Address of PLT relocations
	DT_BIND_NOW = 24 // Process all relocations at load
	DT_RUNPATH  = 29 // Library search path
)

// Extended program header number
const PN_XNUM = 0xffff

// Compression types
const (
	ELFCOMPRESS_ZLIB = 1 // zlib compression
	ELFCOMPRESS_ZSTD = 2 // Zstandard compression
)
