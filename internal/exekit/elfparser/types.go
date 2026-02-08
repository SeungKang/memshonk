package elfparser

// Elf32_Ehdr is the 32-bit ELF header.
type Elf32_Ehdr struct {
	Ident     [EI_NIDENT]byte // Magic number and other info
	Type      uint16          // Object file type
	Machine   uint16          // Architecture
	Version   uint32          // Object file version
	Entry     uint32          // Entry point virtual address
	Phoff     uint32          // Program header table file offset
	Shoff     uint32          // Section header table file offset
	Flags     uint32          // Processor-specific flags
	Ehsize    uint16          // ELF header size in bytes
	Phentsize uint16          // Program header table entry size
	Phnum     uint16          // Program header table entry count
	Shentsize uint16          // Section header table entry size
	Shnum     uint16          // Section header table entry count
	Shstrndx  uint16          // Section header string table index
}

// Elf64_Ehdr is the 64-bit ELF header.
type Elf64_Ehdr struct {
	Ident     [EI_NIDENT]byte // Magic number and other info
	Type      uint16          // Object file type
	Machine   uint16          // Architecture
	Version   uint32          // Object file version
	Entry     uint64          // Entry point virtual address
	Phoff     uint64          // Program header table file offset
	Shoff     uint64          // Section header table file offset
	Flags     uint32          // Processor-specific flags
	Ehsize    uint16          // ELF header size in bytes
	Phentsize uint16          // Program header table entry size
	Phnum     uint16          // Program header table entry count
	Shentsize uint16          // Section header table entry size
	Shnum     uint16          // Section header table entry count
	Shstrndx  uint16          // Section header string table index
}

// Elf32_Phdr is the 32-bit program header.
type Elf32_Phdr struct {
	Type   uint32 // Segment type
	Offset uint32 // Segment file offset
	Vaddr  uint32 // Segment virtual address
	Paddr  uint32 // Segment physical address
	Filesz uint32 // Segment size in file
	Memsz  uint32 // Segment size in memory
	Flags  uint32 // Segment flags
	Align  uint32 // Segment alignment
}

// Elf64_Phdr is the 64-bit program header.
type Elf64_Phdr struct {
	Type   uint32 // Segment type
	Flags  uint32 // Segment flags
	Offset uint64 // Segment file offset
	Vaddr  uint64 // Segment virtual address
	Paddr  uint64 // Segment physical address
	Filesz uint64 // Segment size in file
	Memsz  uint64 // Segment size in memory
	Align  uint64 // Segment alignment
}

// Elf32_Shdr is the 32-bit section header.
type Elf32_Shdr struct {
	Name      uint32 // Section name (index into string table)
	Type      uint32 // Section type
	Flags     uint32 // Section flags
	Addr      uint32 // Section virtual address
	Offset    uint32 // Section file offset
	Size      uint32 // Section size in bytes
	Link      uint32 // Link to another section
	Info      uint32 // Additional section information
	Addralign uint32 // Section alignment
	Entsize   uint32 // Entry size if section holds table
}

// Elf64_Shdr is the 64-bit section header.
type Elf64_Shdr struct {
	Name      uint32 // Section name (index into string table)
	Type      uint32 // Section type
	Flags     uint64 // Section flags
	Addr      uint64 // Section virtual address
	Offset    uint64 // Section file offset
	Size      uint64 // Section size in bytes
	Link      uint32 // Link to another section
	Info      uint32 // Additional section information
	Addralign uint64 // Section alignment
	Entsize   uint64 // Entry size if section holds table
}

// Elf32_Sym is the 32-bit symbol table entry.
type Elf32_Sym struct {
	Name  uint32 // Symbol name (index into string table)
	Value uint32 // Symbol value
	Size  uint32 // Symbol size
	Info  uint8  // Symbol type and binding
	Other uint8  // Symbol visibility
	Shndx uint16 // Section index
}

// Elf64_Sym is the 64-bit symbol table entry.
type Elf64_Sym struct {
	Name  uint32 // Symbol name (index into string table)
	Info  uint8  // Symbol type and binding
	Other uint8  // Symbol visibility
	Shndx uint16 // Section index
	Value uint64 // Symbol value
	Size  uint64 // Symbol size
}

// Elf32_Rel is a 32-bit relocation entry without addend.
type Elf32_Rel struct {
	Offset uint32 // Location at which to apply the action
	Info   uint32 // Relocation type and symbol index
}

// Elf64_Rel is a 64-bit relocation entry without addend.
type Elf64_Rel struct {
	Offset uint64 // Location at which to apply the action
	Info   uint64 // Relocation type and symbol index
}

// Elf32_Rela is a 32-bit relocation entry with addend.
type Elf32_Rela struct {
	Offset uint32 // Location at which to apply the action
	Info   uint32 // Relocation type and symbol index
	Addend int32  // Constant addend
}

// Elf64_Rela is a 64-bit relocation entry with addend.
type Elf64_Rela struct {
	Offset uint64 // Location at which to apply the action
	Info   uint64 // Relocation type and symbol index
	Addend int64  // Constant addend
}

// Elf32_Dyn is a 32-bit dynamic section entry.
type Elf32_Dyn struct {
	Tag int32  // Dynamic entry type
	Val uint32 // Integer or address value
}

// Elf64_Dyn is a 64-bit dynamic section entry.
type Elf64_Dyn struct {
	Tag int64  // Dynamic entry type
	Val uint64 // Integer or address value
}

// Elf32_Chdr is a 32-bit compression header.
type Elf32_Chdr struct {
	Type      uint32 // Compression type
	Size      uint32 // Uncompressed size
	Addralign uint32 // Uncompressed alignment
}

// Elf64_Chdr is a 64-bit compression header.
type Elf64_Chdr struct {
	Type      uint32 // Compression type
	Reserved  uint32 // Reserved (padding)
	Size      uint64 // Uncompressed size
	Addralign uint64 // Uncompressed alignment
}

// Helper functions for symbol info

// ST_BIND extracts binding from st_info.
func ST_BIND(info uint8) uint8 {
	return info >> 4
}

// ST_TYPE extracts type from st_info.
func ST_TYPE(info uint8) uint8 {
	return info & 0xf
}

// ST_INFO creates st_info from binding and type.
func ST_INFO(bind, typ uint8) uint8 {
	return (bind << 4) | (typ & 0xf)
}

// ST_VISIBILITY extracts visibility from st_other.
func ST_VISIBILITY(other uint8) uint8 {
	return other & 0x3
}

// Helper functions for relocation info

// ELF32_R_SYM extracts symbol index from 32-bit r_info.
func ELF32_R_SYM(info uint32) uint32 {
	return info >> 8
}

// ELF32_R_TYPE extracts relocation type from 32-bit r_info.
func ELF32_R_TYPE(info uint32) uint32 {
	return info & 0xff
}

// ELF32_R_INFO creates 32-bit r_info from symbol and type.
func ELF32_R_INFO(sym, typ uint32) uint32 {
	return (sym << 8) | (typ & 0xff)
}

// ELF64_R_SYM extracts symbol index from 64-bit r_info.
func ELF64_R_SYM(info uint64) uint32 {
	return uint32(info >> 32)
}

// ELF64_R_TYPE extracts relocation type from 64-bit r_info.
func ELF64_R_TYPE(info uint64) uint32 {
	return uint32(info & 0xffffffff)
}

// ELF64_R_INFO creates 64-bit r_info from symbol and type.
func ELF64_R_INFO(sym, typ uint32) uint64 {
	return (uint64(sym) << 32) | uint64(typ)
}
