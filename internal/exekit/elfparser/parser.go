package elfparser

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
	ErrInvalidMagic      = errors.New("invalid ELF magic number")
	ErrInvalidClass      = errors.New("invalid ELF class")
	ErrInvalidData       = errors.New("invalid ELF data encoding")
	ErrInvalidVersion    = errors.New("invalid ELF version")
	ErrTruncated         = errors.New("ELF file is truncated")
	ErrInvalidShstrndx   = errors.New("invalid section header string table index")
	ErrInvalidSymtabLink = errors.New("invalid symbol table string table link")
)

// Parser holds the state for parsing an ELF file.
type Parser struct {
	r         io.ReaderAt
	cfg       *epc.ParserConfig
	byteOrder binary.ByteOrder
	class     uint8 // ELFCLASS32 or ELFCLASS64
	exeID     string
	index     uint

	// Cached data
	shstrtab []byte            // Section header string table
	strtabs  map[uint32][]byte // String tables by section index
}

// Parse parses an ELF file using the provided configuration.
func Parse(ctx context.Context, cfg *epc.ParserConfig) error {
	p := &Parser{
		r:       cfg.Src,
		cfg:     cfg,
		exeID:   "main",
		index:   0,
		strtabs: make(map[uint32][]byte),
	}
	return p.parse(ctx)
}

func (p *Parser) parse(ctx context.Context) error {
	// Check for cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Read and validate e_ident
	var ident [EI_NIDENT]byte
	if _, err := p.r.ReadAt(ident[:], 0); err != nil {
		if errors.Is(err, io.EOF) {
			return ErrTruncated
		}
		return fmt.Errorf("reading ELF ident: %w", err)
	}

	if ident[EI_MAG0] != ELFMAG0 || ident[EI_MAG1] != ELFMAG1 ||
		ident[EI_MAG2] != ELFMAG2 || ident[EI_MAG3] != ELFMAG3 {
		return ErrInvalidMagic
	}

	p.class = ident[EI_CLASS]
	if p.class != ELFCLASS32 && p.class != ELFCLASS64 {
		return ErrInvalidClass
	}

	switch ident[EI_DATA] {
	case ELFDATA2LSB:
		p.byteOrder = binary.LittleEndian
	case ELFDATA2MSB:
		p.byteOrder = binary.BigEndian
	default:
		return ErrInvalidData
	}

	if ident[EI_VERSION] != EV_CURRENT {
		return ErrInvalidVersion
	}

	// Parse based on class
	if p.class == ELFCLASS32 {
		return p.parse32(ctx, ident)
	}
	return p.parse64(ctx, ident)
}

func (p *Parser) parse32(ctx context.Context, ident [EI_NIDENT]byte) error {
	// Read 32-bit ELF header
	var hdr Elf32_Ehdr
	hdr.Ident = ident

	buf := make([]byte, 52) // Size of Elf32_Ehdr
	if _, err := p.r.ReadAt(buf, 0); err != nil {
		return fmt.Errorf("reading ELF32 header: %w", err)
	}

	hdr.Type = p.byteOrder.Uint16(buf[16:18])
	hdr.Machine = p.byteOrder.Uint16(buf[18:20])
	hdr.Version = p.byteOrder.Uint32(buf[20:24])
	hdr.Entry = p.byteOrder.Uint32(buf[24:28])
	hdr.Phoff = p.byteOrder.Uint32(buf[28:32])
	hdr.Shoff = p.byteOrder.Uint32(buf[32:36])
	hdr.Flags = p.byteOrder.Uint32(buf[36:40])
	hdr.Ehsize = p.byteOrder.Uint16(buf[40:42])
	hdr.Phentsize = p.byteOrder.Uint16(buf[42:44])
	hdr.Phnum = p.byteOrder.Uint16(buf[44:46])
	hdr.Shentsize = p.byteOrder.Uint16(buf[46:48])
	hdr.Shnum = p.byteOrder.Uint16(buf[48:50])
	hdr.Shstrndx = p.byteOrder.Uint16(buf[50:52])

	// Report info
	if p.cfg.OnInfoFn != nil {
		info := epc.Info{
			Format:     "ELF",
			Class:      32,
			Endian:     endianString(ident[EI_DATA]),
			Type:       elfTypeString(hdr.Type),
			Machine:    machineString(hdr.Machine),
			EntryPoint: uint64(hdr.Entry),
			OSABI:      osabiString(ident[EI_OSABI]),
			ABIVersion: ident[EI_ABIVERSION],
			Flags:      hdr.Flags,
		}
		if err := p.cfg.OnInfoFn(p.exeID, p.index, info); err != nil {
			return err
		}
	}

	// Check for cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Get actual section count (handle extended numbering)
	shnum := uint32(hdr.Shnum)
	shstrndx := uint32(hdr.Shstrndx)

	if hdr.Shoff != 0 && hdr.Shnum == 0 {
		// Extended section numbering: actual count is in sh_size of section 0
		sh0, err := p.readSection32(int64(hdr.Shoff), hdr.Shentsize)
		if err != nil {
			return err
		}
		shnum = sh0.Size
	}

	if hdr.Shstrndx == SHN_XINDEX && hdr.Shoff != 0 {
		// Extended section numbering: actual index is in sh_link of section 0
		sh0, err := p.readSection32(int64(hdr.Shoff), hdr.Shentsize)
		if err != nil {
			return err
		}
		shstrndx = sh0.Link
	}

	// Read section headers and load section name string table
	var sections []Elf32_Shdr
	if hdr.Shoff != 0 && shnum > 0 {
		sections = make([]Elf32_Shdr, shnum)
		for i := uint32(0); i < shnum; i++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			sh, err := p.readSection32(int64(hdr.Shoff)+int64(i)*int64(hdr.Shentsize), hdr.Shentsize)
			if err != nil {
				return err
			}
			sections[i] = sh
		}

		// Load section header string table
		if shstrndx != SHN_UNDEF && shstrndx < shnum {
			shstrtabSec := sections[shstrndx]
			if shstrtabSec.Type == SHT_STRTAB {
				p.shstrtab = make([]byte, shstrtabSec.Size)
				if _, err := p.r.ReadAt(p.shstrtab, int64(shstrtabSec.Offset)); err != nil {
					return fmt.Errorf("reading section string table: %w", err)
				}
			}
		}
	}

	// Parse program headers (segments)
	if hdr.Phoff != 0 && hdr.Phnum > 0 {
		phnum := uint32(hdr.Phnum)
		if hdr.Phnum == PN_XNUM && len(sections) > 0 {
			phnum = sections[0].Info
		}
		if err := p.parseProgHeaders32(ctx, int64(hdr.Phoff), hdr.Phentsize, phnum); err != nil {
			return err
		}
	}

	// Parse sections
	if err := p.parseSections32(ctx, sections); err != nil {
		return err
	}

	return nil
}

func (p *Parser) parse64(ctx context.Context, ident [EI_NIDENT]byte) error {
	// Read 64-bit ELF header
	var hdr Elf64_Ehdr
	hdr.Ident = ident

	buf := make([]byte, 64) // Size of Elf64_Ehdr
	if _, err := p.r.ReadAt(buf, 0); err != nil {
		return fmt.Errorf("reading ELF64 header: %w", err)
	}

	hdr.Type = p.byteOrder.Uint16(buf[16:18])
	hdr.Machine = p.byteOrder.Uint16(buf[18:20])
	hdr.Version = p.byteOrder.Uint32(buf[20:24])
	hdr.Entry = p.byteOrder.Uint64(buf[24:32])
	hdr.Phoff = p.byteOrder.Uint64(buf[32:40])
	hdr.Shoff = p.byteOrder.Uint64(buf[40:48])
	hdr.Flags = p.byteOrder.Uint32(buf[48:52])
	hdr.Ehsize = p.byteOrder.Uint16(buf[52:54])
	hdr.Phentsize = p.byteOrder.Uint16(buf[54:56])
	hdr.Phnum = p.byteOrder.Uint16(buf[56:58])
	hdr.Shentsize = p.byteOrder.Uint16(buf[58:60])
	hdr.Shnum = p.byteOrder.Uint16(buf[60:62])
	hdr.Shstrndx = p.byteOrder.Uint16(buf[62:64])

	// Report info
	if p.cfg.OnInfoFn != nil {
		info := epc.Info{
			Format:     "ELF",
			Class:      64,
			Endian:     endianString(ident[EI_DATA]),
			Type:       elfTypeString(hdr.Type),
			Machine:    machineString(hdr.Machine),
			EntryPoint: hdr.Entry,
			OSABI:      osabiString(ident[EI_OSABI]),
			ABIVersion: ident[EI_ABIVERSION],
			Flags:      hdr.Flags,
		}
		if err := p.cfg.OnInfoFn(p.exeID, p.index, info); err != nil {
			return err
		}
	}

	// Check for cancellation
	if err := ctx.Err(); err != nil {
		return err
	}

	// Get actual section count (handle extended numbering)
	shnum := uint64(hdr.Shnum)
	shstrndx := uint32(hdr.Shstrndx)

	if hdr.Shoff != 0 && hdr.Shnum == 0 {
		// Extended section numbering: actual count is in sh_size of section 0
		sh0, err := p.readSection64(int64(hdr.Shoff), hdr.Shentsize)
		if err != nil {
			return err
		}
		shnum = sh0.Size
	}

	if hdr.Shstrndx == SHN_XINDEX && hdr.Shoff != 0 {
		// Extended section numbering: actual index is in sh_link of section 0
		sh0, err := p.readSection64(int64(hdr.Shoff), hdr.Shentsize)
		if err != nil {
			return err
		}
		shstrndx = sh0.Link
	}

	// Read section headers and load section name string table
	var sections []Elf64_Shdr
	if hdr.Shoff != 0 && shnum > 0 {
		sections = make([]Elf64_Shdr, shnum)
		for i := uint64(0); i < shnum; i++ {
			if err := ctx.Err(); err != nil {
				return err
			}
			sh, err := p.readSection64(int64(hdr.Shoff)+int64(i)*int64(hdr.Shentsize), hdr.Shentsize)
			if err != nil {
				return err
			}
			sections[i] = sh
		}

		// Load section header string table
		if shstrndx != SHN_UNDEF && uint64(shstrndx) < shnum {
			shstrtabSec := sections[shstrndx]
			if shstrtabSec.Type == SHT_STRTAB {
				p.shstrtab = make([]byte, shstrtabSec.Size)
				if _, err := p.r.ReadAt(p.shstrtab, int64(shstrtabSec.Offset)); err != nil {
					return fmt.Errorf("reading section string table: %w", err)
				}
			}
		}
	}

	// Parse program headers (segments)
	if hdr.Phoff != 0 && hdr.Phnum > 0 {
		phnum := uint64(hdr.Phnum)
		if hdr.Phnum == PN_XNUM && len(sections) > 0 {
			phnum = uint64(sections[0].Info)
		}
		if err := p.parseProgHeaders64(ctx, int64(hdr.Phoff), hdr.Phentsize, phnum); err != nil {
			return err
		}
	}

	// Parse sections
	if err := p.parseSections64(ctx, sections); err != nil {
		return err
	}

	return nil
}

func (p *Parser) readSection32(offset int64, size uint16) (Elf32_Shdr, error) {
	buf := make([]byte, size)
	if _, err := p.r.ReadAt(buf, offset); err != nil {
		return Elf32_Shdr{}, fmt.Errorf("reading section header: %w", err)
	}

	return Elf32_Shdr{
		Name:      p.byteOrder.Uint32(buf[0:4]),
		Type:      p.byteOrder.Uint32(buf[4:8]),
		Flags:     p.byteOrder.Uint32(buf[8:12]),
		Addr:      p.byteOrder.Uint32(buf[12:16]),
		Offset:    p.byteOrder.Uint32(buf[16:20]),
		Size:      p.byteOrder.Uint32(buf[20:24]),
		Link:      p.byteOrder.Uint32(buf[24:28]),
		Info:      p.byteOrder.Uint32(buf[28:32]),
		Addralign: p.byteOrder.Uint32(buf[32:36]),
		Entsize:   p.byteOrder.Uint32(buf[36:40]),
	}, nil
}

func (p *Parser) readSection64(offset int64, size uint16) (Elf64_Shdr, error) {
	buf := make([]byte, size)
	if _, err := p.r.ReadAt(buf, offset); err != nil {
		return Elf64_Shdr{}, fmt.Errorf("reading section header: %w", err)
	}

	return Elf64_Shdr{
		Name:      p.byteOrder.Uint32(buf[0:4]),
		Type:      p.byteOrder.Uint32(buf[4:8]),
		Flags:     p.byteOrder.Uint64(buf[8:16]),
		Addr:      p.byteOrder.Uint64(buf[16:24]),
		Offset:    p.byteOrder.Uint64(buf[24:32]),
		Size:      p.byteOrder.Uint64(buf[32:40]),
		Link:      p.byteOrder.Uint32(buf[40:44]),
		Info:      p.byteOrder.Uint32(buf[44:48]),
		Addralign: p.byteOrder.Uint64(buf[48:56]),
		Entsize:   p.byteOrder.Uint64(buf[56:64]),
	}, nil
}

func (p *Parser) parseProgHeaders32(ctx context.Context, offset int64, entsize uint16, count uint32) error {
	if p.cfg.OnSegmentFn == nil {
		return nil
	}

	buf := make([]byte, entsize)
	for i := uint32(0); i < count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		if _, err := p.r.ReadAt(buf, offset+int64(i)*int64(entsize)); err != nil {
			return fmt.Errorf("reading program header %d: %w", i, err)
		}

		ph := Elf32_Phdr{
			Type:   p.byteOrder.Uint32(buf[0:4]),
			Offset: p.byteOrder.Uint32(buf[4:8]),
			Vaddr:  p.byteOrder.Uint32(buf[8:12]),
			Paddr:  p.byteOrder.Uint32(buf[12:16]),
			Filesz: p.byteOrder.Uint32(buf[16:20]),
			Memsz:  p.byteOrder.Uint32(buf[20:24]),
			Flags:  p.byteOrder.Uint32(buf[24:28]),
			Align:  p.byteOrder.Uint32(buf[28:32]),
		}

		seg := epc.Segment{
			Type:     ph.Type,
			Flags:    ph.Flags,
			Offset:   uint64(ph.Offset),
			VAddr:    uint64(ph.Vaddr),
			PAddr:    uint64(ph.Paddr),
			FileSize: uint64(ph.Filesz),
			MemSize:  uint64(ph.Memsz),
			Align:    uint64(ph.Align),
		}

		if err := p.cfg.OnSegmentFn(p.exeID, p.index, seg); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseProgHeaders64(ctx context.Context, offset int64, entsize uint16, count uint64) error {
	if p.cfg.OnSegmentFn == nil {
		return nil
	}

	buf := make([]byte, entsize)
	for i := uint64(0); i < count; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}

		if _, err := p.r.ReadAt(buf, offset+int64(i)*int64(entsize)); err != nil {
			return fmt.Errorf("reading program header %d: %w", i, err)
		}

		ph := Elf64_Phdr{
			Type:   p.byteOrder.Uint32(buf[0:4]),
			Flags:  p.byteOrder.Uint32(buf[4:8]),
			Offset: p.byteOrder.Uint64(buf[8:16]),
			Vaddr:  p.byteOrder.Uint64(buf[16:24]),
			Paddr:  p.byteOrder.Uint64(buf[24:32]),
			Filesz: p.byteOrder.Uint64(buf[32:40]),
			Memsz:  p.byteOrder.Uint64(buf[40:48]),
			Align:  p.byteOrder.Uint64(buf[48:56]),
		}

		seg := epc.Segment{
			Type:     ph.Type,
			Flags:    ph.Flags,
			Offset:   ph.Offset,
			VAddr:    ph.Vaddr,
			PAddr:    ph.Paddr,
			FileSize: ph.Filesz,
			MemSize:  ph.Memsz,
			Align:    ph.Align,
		}

		if err := p.cfg.OnSegmentFn(p.exeID, p.index, seg); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) parseSections32(ctx context.Context, sections []Elf32_Shdr) error {
	// First pass: load all string tables so symbol tables can reference them
	for i, sh := range sections {
		if err := ctx.Err(); err != nil {
			return err
		}
		if sh.Type == SHT_STRTAB {
			if err := p.parseStrtab32(ctx, sh, uint32(i)); err != nil {
				return err
			}
		}
	}

	// Second pass: process all sections
	for i, sh := range sections {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Report section if callback is set
		if p.cfg.OnSectionFn != nil {
			sec := epc.Section{
				Name:       p.getSectionName(sh.Name),
				Type:       sh.Type,
				Flags:      uint64(sh.Flags),
				Addr:       uint64(sh.Addr),
				Offset:     uint64(sh.Offset),
				Size:       uint64(sh.Size),
				Link:       sh.Link,
				Info:       sh.Info,
				Align:      uint64(sh.Addralign),
				EntSize:    uint64(sh.Entsize),
				IsCode:     sh.Flags&SHF_EXECINSTR != 0,
				IsData:     sh.Flags&SHF_ALLOC != 0 && sh.Flags&SHF_EXECINSTR == 0,
				IsWritable: sh.Flags&SHF_WRITE != 0,
			}
			if err := p.cfg.OnSectionFn(p.exeID, p.index, sec); err != nil {
				return err
			}
		}

		// Parse section contents based on type (skip strtab, already done)
		switch sh.Type {
		case SHT_SYMTAB, SHT_DYNSYM:
			if err := p.parseSymtab32(ctx, sh, uint32(i)); err != nil {
				return err
			}
		case SHT_REL:
			if err := p.parseRel32(ctx, sh); err != nil {
				return err
			}
		case SHT_RELA:
			if err := p.parseRela32(ctx, sh); err != nil {
				return err
			}
		case SHT_DYNAMIC:
			if err := p.parseDynamic32(ctx, sh); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Parser) parseSections64(ctx context.Context, sections []Elf64_Shdr) error {
	// First pass: load all string tables so symbol tables can reference them
	for i, sh := range sections {
		if err := ctx.Err(); err != nil {
			return err
		}
		if sh.Type == SHT_STRTAB {
			if err := p.parseStrtab64(ctx, sh, uint32(i)); err != nil {
				return err
			}
		}
	}

	// Second pass: process all sections
	for i, sh := range sections {
		if err := ctx.Err(); err != nil {
			return err
		}

		// Report section if callback is set
		if p.cfg.OnSectionFn != nil {
			sec := epc.Section{
				Name:       p.getSectionName(sh.Name),
				Type:       sh.Type,
				Flags:      sh.Flags,
				Addr:       sh.Addr,
				Offset:     sh.Offset,
				Size:       sh.Size,
				Link:       sh.Link,
				Info:       sh.Info,
				Align:      sh.Addralign,
				EntSize:    sh.Entsize,
				IsCode:     sh.Flags&SHF_EXECINSTR != 0,
				IsData:     sh.Flags&SHF_ALLOC != 0 && sh.Flags&SHF_EXECINSTR == 0,
				IsWritable: sh.Flags&SHF_WRITE != 0,
			}
			if err := p.cfg.OnSectionFn(p.exeID, p.index, sec); err != nil {
				return err
			}
		}

		// Parse section contents based on type (skip strtab, already done)
		switch sh.Type {
		case SHT_SYMTAB, SHT_DYNSYM:
			if err := p.parseSymtab64(ctx, sh, uint32(i)); err != nil {
				return err
			}
		case SHT_REL:
			if err := p.parseRel64(ctx, sh); err != nil {
				return err
			}
		case SHT_RELA:
			if err := p.parseRela64(ctx, sh); err != nil {
				return err
			}
		case SHT_DYNAMIC:
			if err := p.parseDynamic64(ctx, sh); err != nil {
				return err
			}
		}
	}

	return nil
}

func (p *Parser) getSectionName(nameIdx uint32) string {
	if p.shstrtab == nil || nameIdx >= uint32(len(p.shstrtab)) {
		return ""
	}
	return p.getString(p.shstrtab, nameIdx)
}

func (p *Parser) getString(strtab []byte, idx uint32) string {
	if idx >= uint32(len(strtab)) {
		return ""
	}
	end := idx
	for end < uint32(len(strtab)) && strtab[end] != 0 {
		end++
	}
	return string(strtab[idx:end])
}

func (p *Parser) loadStrtab(idx uint32) ([]byte, error) {
	if strtab, ok := p.strtabs[idx]; ok {
		return strtab, nil
	}
	// String table not loaded - this shouldn't happen in normal flow
	return nil, nil
}

// Helper functions for converting values to strings

func endianString(data byte) string {
	switch data {
	case ELFDATA2LSB:
		return "little"
	case ELFDATA2MSB:
		return "big"
	default:
		return "unknown"
	}
}

func elfTypeString(t uint16) string {
	switch t {
	case ET_NONE:
		return "none"
	case ET_REL:
		return "relocatable"
	case ET_EXEC:
		return "executable"
	case ET_DYN:
		return "shared"
	case ET_CORE:
		return "core"
	default:
		return "unknown"
	}
}

func machineString(m uint16) string {
	switch m {
	case EM_NONE:
		return "none"
	case EM_386:
		return "i386"
	case EM_X86_64:
		return "x86_64"
	case EM_ARM:
		return "arm"
	case EM_AARCH64:
		return "arm64"
	case EM_MIPS:
		return "mips"
	case EM_PPC:
		return "ppc"
	case EM_PPC64:
		return "ppc64"
	case EM_SPARC:
		return "sparc"
	case EM_SPARCV9:
		return "sparc64"
	case EM_RISCV:
		return "riscv"
	case EM_IA_64:
		return "ia64"
	case EM_S390:
		return "s390"
	default:
		return fmt.Sprintf("machine(%d)", m)
	}
}

func osabiString(osabi byte) string {
	switch osabi {
	case ELFOSABI_SYSV:
		return "SysV"
	case ELFOSABI_HPUX:
		return "HP-UX"
	case ELFOSABI_NETBSD:
		return "NetBSD"
	case ELFOSABI_LINUX:
		return "Linux"
	case ELFOSABI_HURD:
		return "Hurd"
	case ELFOSABI_SOLARIS:
		return "Solaris"
	case ELFOSABI_IRIX:
		return "IRIX"
	case ELFOSABI_FREEBSD:
		return "FreeBSD"
	case ELFOSABI_TRU64:
		return "TRU64"
	case ELFOSABI_ARM:
		return "ARM"
	case ELFOSABI_STANDALONE:
		return "Standalone"
	case ELFOSABI_OPENBSD:
		return "OpenBSD"
	default:
		return fmt.Sprintf("osabi(%d)", osabi)
	}
}
