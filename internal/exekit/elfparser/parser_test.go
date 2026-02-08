package elfparser

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"testing"

	"github.com/SeungKang/memshonk/internal/exekit/epc"
)

// bytesReaderAt wraps a byte slice to implement io.ReaderAt
type bytesReaderAt struct {
	data []byte
}

func (b *bytesReaderAt) ReadAt(p []byte, off int64) (n int, err error) {
	if off < 0 {
		return 0, io.EOF
	}
	if off >= int64(len(b.data)) {
		return 0, io.EOF
	}
	n = copy(p, b.data[off:])
	if n < len(p) {
		err = io.EOF
	}
	return n, err
}

// makeELF64Header creates a minimal valid ELF64 header
func makeELF64Header(t *testing.T, machine uint16, elfType uint16, osabi byte) []byte {
	buf := make([]byte, 64)

	// e_ident
	buf[EI_MAG0] = ELFMAG0
	buf[EI_MAG1] = ELFMAG1
	buf[EI_MAG2] = ELFMAG2
	buf[EI_MAG3] = ELFMAG3
	buf[EI_CLASS] = ELFCLASS64
	buf[EI_DATA] = ELFDATA2LSB
	buf[EI_VERSION] = EV_CURRENT
	buf[EI_OSABI] = osabi

	// Rest of header (little-endian)
	binary.LittleEndian.PutUint16(buf[16:18], elfType)    // e_type
	binary.LittleEndian.PutUint16(buf[18:20], machine)    // e_machine
	binary.LittleEndian.PutUint32(buf[20:24], EV_CURRENT) // e_version
	binary.LittleEndian.PutUint64(buf[24:32], 0x401000)   // e_entry
	binary.LittleEndian.PutUint64(buf[32:40], 0)          // e_phoff
	binary.LittleEndian.PutUint64(buf[40:48], 0)          // e_shoff
	binary.LittleEndian.PutUint32(buf[48:52], 0)          // e_flags
	binary.LittleEndian.PutUint16(buf[52:54], 64)         // e_ehsize
	binary.LittleEndian.PutUint16(buf[54:56], 56)         // e_phentsize
	binary.LittleEndian.PutUint16(buf[56:58], 0)          // e_phnum
	binary.LittleEndian.PutUint16(buf[58:60], 64)         // e_shentsize
	binary.LittleEndian.PutUint16(buf[60:62], 0)          // e_shnum
	binary.LittleEndian.PutUint16(buf[62:64], 0)          // e_shstrndx

	return buf
}

// makeELF32Header creates a minimal valid ELF32 header
func makeELF32Header(t *testing.T, machine uint16, elfType uint16, osabi byte) []byte {
	buf := make([]byte, 52)

	// e_ident
	buf[EI_MAG0] = ELFMAG0
	buf[EI_MAG1] = ELFMAG1
	buf[EI_MAG2] = ELFMAG2
	buf[EI_MAG3] = ELFMAG3
	buf[EI_CLASS] = ELFCLASS32
	buf[EI_DATA] = ELFDATA2LSB
	buf[EI_VERSION] = EV_CURRENT
	buf[EI_OSABI] = osabi

	// Rest of header (little-endian)
	binary.LittleEndian.PutUint16(buf[16:18], elfType)    // e_type
	binary.LittleEndian.PutUint16(buf[18:20], machine)    // e_machine
	binary.LittleEndian.PutUint32(buf[20:24], EV_CURRENT) // e_version
	binary.LittleEndian.PutUint32(buf[24:28], 0x8048000)  // e_entry
	binary.LittleEndian.PutUint32(buf[28:32], 0)          // e_phoff
	binary.LittleEndian.PutUint32(buf[32:36], 0)          // e_shoff
	binary.LittleEndian.PutUint32(buf[36:40], 0)          // e_flags
	binary.LittleEndian.PutUint16(buf[40:42], 52)         // e_ehsize
	binary.LittleEndian.PutUint16(buf[42:44], 32)         // e_phentsize
	binary.LittleEndian.PutUint16(buf[44:46], 0)          // e_phnum
	binary.LittleEndian.PutUint16(buf[46:48], 40)         // e_shentsize
	binary.LittleEndian.PutUint16(buf[48:50], 0)          // e_shnum
	binary.LittleEndian.PutUint16(buf[50:52], 0)          // e_shstrndx

	return buf
}

func TestParseInvalidMagic(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too short", []byte{0x7f, 'E'}},
		{"wrong magic", []byte{0x00, 0x00, 0x00, 0x00}},
		{"partial magic", []byte{0x7f, 'E', 'L', 'X'}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &epc.ParserConfig{
				Src: &bytesReaderAt{data: tt.data},
			}
			err := Parse(context.Background(), cfg)
			if err == nil {
				t.Error("expected error for invalid magic")
			}
		})
	}
}

func TestParseInvalidClass(t *testing.T) {
	data := make([]byte, 64)
	data[EI_MAG0] = ELFMAG0
	data[EI_MAG1] = ELFMAG1
	data[EI_MAG2] = ELFMAG2
	data[EI_MAG3] = ELFMAG3
	data[EI_CLASS] = 99 // Invalid class
	data[EI_DATA] = ELFDATA2LSB
	data[EI_VERSION] = EV_CURRENT

	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
	}
	err := Parse(context.Background(), cfg)
	if err != ErrInvalidClass {
		t.Errorf("expected ErrInvalidClass, got %v", err)
	}
}

func TestParseInvalidDataEncoding(t *testing.T) {
	data := make([]byte, 64)
	data[EI_MAG0] = ELFMAG0
	data[EI_MAG1] = ELFMAG1
	data[EI_MAG2] = ELFMAG2
	data[EI_MAG3] = ELFMAG3
	data[EI_CLASS] = ELFCLASS64
	data[EI_DATA] = 99 // Invalid data encoding
	data[EI_VERSION] = EV_CURRENT

	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
	}
	err := Parse(context.Background(), cfg)
	if err != ErrInvalidData {
		t.Errorf("expected ErrInvalidData, got %v", err)
	}
}

func TestParseInvalidVersion(t *testing.T) {
	data := make([]byte, 64)
	data[EI_MAG0] = ELFMAG0
	data[EI_MAG1] = ELFMAG1
	data[EI_MAG2] = ELFMAG2
	data[EI_MAG3] = ELFMAG3
	data[EI_CLASS] = ELFCLASS64
	data[EI_DATA] = ELFDATA2LSB
	data[EI_VERSION] = 99 // Invalid version

	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
	}
	err := Parse(context.Background(), cfg)
	if err != ErrInvalidVersion {
		t.Errorf("expected ErrInvalidVersion, got %v", err)
	}
}

func TestParseELF64Header(t *testing.T) {
	data := makeELF64Header(t, EM_X86_64, ET_EXEC, ELFOSABI_LINUX)

	var gotInfo epc.Info
	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
		OnInfoFn: func(exeID string, index uint, info epc.Info) error {
			gotInfo = info
			return nil
		},
	}

	err := Parse(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotInfo.Format != "ELF" {
		t.Errorf("expected format ELF, got %s", gotInfo.Format)
	}
	if gotInfo.Class != 64 {
		t.Errorf("expected class 64, got %d", gotInfo.Class)
	}
	if gotInfo.Endian != "little" {
		t.Errorf("expected endian little, got %s", gotInfo.Endian)
	}
	if gotInfo.Type != "executable" {
		t.Errorf("expected type executable, got %s", gotInfo.Type)
	}
	if gotInfo.Machine != "x86_64" {
		t.Errorf("expected machine x86_64, got %s", gotInfo.Machine)
	}
	if gotInfo.OSABI != "Linux" {
		t.Errorf("expected OSABI Linux, got %s", gotInfo.OSABI)
	}
	if gotInfo.EntryPoint != 0x401000 {
		t.Errorf("expected entry point 0x401000, got 0x%x", gotInfo.EntryPoint)
	}
}

func TestParseELF32Header(t *testing.T) {
	data := makeELF32Header(t, EM_386, ET_DYN, ELFOSABI_FREEBSD)

	var gotInfo epc.Info
	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
		OnInfoFn: func(exeID string, index uint, info epc.Info) error {
			gotInfo = info
			return nil
		},
	}

	err := Parse(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotInfo.Format != "ELF" {
		t.Errorf("expected format ELF, got %s", gotInfo.Format)
	}
	if gotInfo.Class != 32 {
		t.Errorf("expected class 32, got %d", gotInfo.Class)
	}
	if gotInfo.Type != "shared" {
		t.Errorf("expected type shared, got %s", gotInfo.Type)
	}
	if gotInfo.Machine != "i386" {
		t.Errorf("expected machine i386, got %s", gotInfo.Machine)
	}
	if gotInfo.OSABI != "FreeBSD" {
		t.Errorf("expected OSABI FreeBSD, got %s", gotInfo.OSABI)
	}
}

func TestParseELF64BigEndian(t *testing.T) {
	buf := make([]byte, 64)

	// e_ident
	buf[EI_MAG0] = ELFMAG0
	buf[EI_MAG1] = ELFMAG1
	buf[EI_MAG2] = ELFMAG2
	buf[EI_MAG3] = ELFMAG3
	buf[EI_CLASS] = ELFCLASS64
	buf[EI_DATA] = ELFDATA2MSB // Big-endian
	buf[EI_VERSION] = EV_CURRENT
	buf[EI_OSABI] = ELFOSABI_SYSV

	// Rest of header (big-endian)
	binary.BigEndian.PutUint16(buf[16:18], ET_EXEC)
	binary.BigEndian.PutUint16(buf[18:20], EM_PPC64)
	binary.BigEndian.PutUint32(buf[20:24], EV_CURRENT)
	binary.BigEndian.PutUint64(buf[24:32], 0x10000000)
	binary.BigEndian.PutUint16(buf[52:54], 64) // e_ehsize

	var gotInfo epc.Info
	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: buf},
		OnInfoFn: func(exeID string, index uint, info epc.Info) error {
			gotInfo = info
			return nil
		},
	}

	err := Parse(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotInfo.Endian != "big" {
		t.Errorf("expected endian big, got %s", gotInfo.Endian)
	}
	if gotInfo.Machine != "ppc64" {
		t.Errorf("expected machine ppc64, got %s", gotInfo.Machine)
	}
	if gotInfo.EntryPoint != 0x10000000 {
		t.Errorf("expected entry point 0x10000000, got 0x%x", gotInfo.EntryPoint)
	}
}

func TestParseContextCancellation(t *testing.T) {
	data := makeELF64Header(t, EM_X86_64, ET_EXEC, ELFOSABI_LINUX)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
	}

	err := Parse(ctx, cfg)
	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestParseWithSegments(t *testing.T) {
	// Create ELF64 with one program header
	ehdr := makeELF64Header(t, EM_X86_64, ET_EXEC, ELFOSABI_LINUX)

	// Update header to point to program headers
	binary.LittleEndian.PutUint64(ehdr[32:40], 64)   // e_phoff = 64 (right after header)
	binary.LittleEndian.PutUint16(ehdr[56:58], 1)    // e_phnum = 1

	// Create a PT_LOAD program header (56 bytes for 64-bit)
	phdr := make([]byte, 56)
	binary.LittleEndian.PutUint32(phdr[0:4], PT_LOAD)    // p_type
	binary.LittleEndian.PutUint32(phdr[4:8], PF_R|PF_X)  // p_flags
	binary.LittleEndian.PutUint64(phdr[8:16], 0)         // p_offset
	binary.LittleEndian.PutUint64(phdr[16:24], 0x400000) // p_vaddr
	binary.LittleEndian.PutUint64(phdr[24:32], 0x400000) // p_paddr
	binary.LittleEndian.PutUint64(phdr[32:40], 0x1000)   // p_filesz
	binary.LittleEndian.PutUint64(phdr[40:48], 0x1000)   // p_memsz
	binary.LittleEndian.PutUint64(phdr[48:56], 0x1000)   // p_align

	data := append(ehdr, phdr...)

	var segments []epc.Segment
	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
		OnSegmentFn: func(exeID string, index uint, seg epc.Segment) error {
			segments = append(segments, seg)
			return nil
		},
	}

	err := Parse(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segments))
	}

	seg := segments[0]
	if seg.Type != PT_LOAD {
		t.Errorf("expected type PT_LOAD, got %d", seg.Type)
	}
	if seg.Flags != PF_R|PF_X {
		t.Errorf("expected flags PF_R|PF_X, got %d", seg.Flags)
	}
	if seg.VAddr != 0x400000 {
		t.Errorf("expected vaddr 0x400000, got 0x%x", seg.VAddr)
	}
	if seg.FileSize != 0x1000 {
		t.Errorf("expected filesz 0x1000, got 0x%x", seg.FileSize)
	}
}

func TestParseWithSections(t *testing.T) {
	// Create ELF64 with section headers
	ehdr := makeELF64Header(t, EM_X86_64, ET_EXEC, ELFOSABI_LINUX)

	// Section header string table content
	shstrtab := []byte("\x00.shstrtab\x00.text\x00")

	// Calculate offsets
	shstrtabOffset := uint64(64) // Right after ELF header
	shdrOffset := shstrtabOffset + uint64(len(shstrtab))
	// Align to 8 bytes
	if shdrOffset%8 != 0 {
		shdrOffset += 8 - (shdrOffset % 8)
	}

	// Update ELF header
	binary.LittleEndian.PutUint64(ehdr[40:48], shdrOffset) // e_shoff
	binary.LittleEndian.PutUint16(ehdr[60:62], 3)          // e_shnum (NULL + .shstrtab + .text)
	binary.LittleEndian.PutUint16(ehdr[62:64], 1)          // e_shstrndx

	// Create section headers (64 bytes each for 64-bit)
	// Section 0: NULL
	shdr0 := make([]byte, 64)

	// Section 1: .shstrtab
	shdr1 := make([]byte, 64)
	binary.LittleEndian.PutUint32(shdr1[0:4], 1)                      // sh_name (offset in shstrtab)
	binary.LittleEndian.PutUint32(shdr1[4:8], SHT_STRTAB)             // sh_type
	binary.LittleEndian.PutUint64(shdr1[24:32], shstrtabOffset)       // sh_offset
	binary.LittleEndian.PutUint64(shdr1[32:40], uint64(len(shstrtab))) // sh_size

	// Section 2: .text
	shdr2 := make([]byte, 64)
	binary.LittleEndian.PutUint32(shdr2[0:4], 11)                          // sh_name (offset of ".text" in shstrtab)
	binary.LittleEndian.PutUint32(shdr2[4:8], SHT_PROGBITS)                // sh_type
	binary.LittleEndian.PutUint64(shdr2[8:16], SHF_ALLOC|SHF_EXECINSTR)    // sh_flags
	binary.LittleEndian.PutUint64(shdr2[16:24], 0x401000)                  // sh_addr
	binary.LittleEndian.PutUint64(shdr2[24:32], 0x1000)                    // sh_offset
	binary.LittleEndian.PutUint64(shdr2[32:40], 0x100)                     // sh_size

	// Build the file
	data := make([]byte, shdrOffset+192)
	copy(data[0:], ehdr)
	copy(data[shstrtabOffset:], shstrtab)
	copy(data[shdrOffset:], shdr0)
	copy(data[shdrOffset+64:], shdr1)
	copy(data[shdrOffset+128:], shdr2)

	var sections []epc.Section
	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
		OnSectionFn: func(exeID string, index uint, sec epc.Section) error {
			sections = append(sections, sec)
			return nil
		},
	}

	err := Parse(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(sections) != 3 {
		t.Fatalf("expected 3 sections, got %d", len(sections))
	}

	// Check .text section
	textSec := sections[2]
	if textSec.Name != ".text" {
		t.Errorf("expected section name .text, got %s", textSec.Name)
	}
	if textSec.Type != SHT_PROGBITS {
		t.Errorf("expected type SHT_PROGBITS, got %d", textSec.Type)
	}
	if !textSec.IsCode {
		t.Error("expected .text to be marked as code")
	}
	if textSec.Addr != 0x401000 {
		t.Errorf("expected addr 0x401000, got 0x%x", textSec.Addr)
	}
}

func TestParseNoCallbacks(t *testing.T) {
	data := makeELF64Header(t, EM_X86_64, ET_EXEC, ELFOSABI_LINUX)

	// Parse with no callbacks set - should still succeed
	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
	}

	err := Parse(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCallbackError(t *testing.T) {
	data := makeELF64Header(t, EM_X86_64, ET_EXEC, ELFOSABI_LINUX)

	expectedErr := io.ErrUnexpectedEOF
	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
		OnInfoFn: func(exeID string, index uint, info epc.Info) error {
			return expectedErr
		},
	}

	err := Parse(context.Background(), cfg)
	if err != expectedErr {
		t.Errorf("expected callback error to be propagated, got %v", err)
	}
}

func TestMachineStrings(t *testing.T) {
	tests := []struct {
		machine  uint16
		expected string
	}{
		{EM_NONE, "none"},
		{EM_386, "i386"},
		{EM_X86_64, "x86_64"},
		{EM_ARM, "arm"},
		{EM_AARCH64, "arm64"},
		{EM_MIPS, "mips"},
		{EM_PPC, "ppc"},
		{EM_PPC64, "ppc64"},
		{EM_SPARC, "sparc"},
		{EM_SPARCV9, "sparc64"},
		{EM_RISCV, "riscv"},
		{EM_IA_64, "ia64"},
		{EM_S390, "s390"},
		{9999, "machine(9999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := machineString(tt.machine)
			if got != tt.expected {
				t.Errorf("machineString(%d) = %s, want %s", tt.machine, got, tt.expected)
			}
		})
	}
}

func TestOSABIStrings(t *testing.T) {
	tests := []struct {
		osabi    byte
		expected string
	}{
		{ELFOSABI_SYSV, "SysV"},
		{ELFOSABI_LINUX, "Linux"},
		{ELFOSABI_FREEBSD, "FreeBSD"},
		{ELFOSABI_NETBSD, "NetBSD"},
		{ELFOSABI_OPENBSD, "OpenBSD"},
		{ELFOSABI_SOLARIS, "Solaris"},
		{200, "osabi(200)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := osabiString(tt.osabi)
			if got != tt.expected {
				t.Errorf("osabiString(%d) = %s, want %s", tt.osabi, got, tt.expected)
			}
		})
	}
}

func TestELFTypeStrings(t *testing.T) {
	tests := []struct {
		elfType  uint16
		expected string
	}{
		{ET_NONE, "none"},
		{ET_REL, "relocatable"},
		{ET_EXEC, "executable"},
		{ET_DYN, "shared"},
		{ET_CORE, "core"},
		{9999, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := elfTypeString(tt.elfType)
			if got != tt.expected {
				t.Errorf("elfTypeString(%d) = %s, want %s", tt.elfType, got, tt.expected)
			}
		})
	}
}

func TestEndianStrings(t *testing.T) {
	if endianString(ELFDATA2LSB) != "little" {
		t.Error("expected 'little' for ELFDATA2LSB")
	}
	if endianString(ELFDATA2MSB) != "big" {
		t.Error("expected 'big' for ELFDATA2MSB")
	}
	if endianString(99) != "unknown" {
		t.Error("expected 'unknown' for invalid encoding")
	}
}

func TestGetString(t *testing.T) {
	p := &Parser{}

	strtab := []byte("\x00hello\x00world\x00")

	tests := []struct {
		idx      uint32
		expected string
	}{
		{0, ""},
		{1, "hello"},
		{7, "world"},
		{100, ""}, // Out of bounds
	}

	for _, tt := range tests {
		got := p.getString(strtab, tt.idx)
		if got != tt.expected {
			t.Errorf("getString(strtab, %d) = %q, want %q", tt.idx, got, tt.expected)
		}
	}
}

func TestSymbolHelpers(t *testing.T) {
	// Test ST_BIND
	if ST_BIND(0x12) != 1 {
		t.Error("ST_BIND failed")
	}

	// Test ST_TYPE
	if ST_TYPE(0x12) != 2 {
		t.Error("ST_TYPE failed")
	}

	// Test ST_INFO
	if ST_INFO(1, 2) != 0x12 {
		t.Error("ST_INFO failed")
	}

	// Test ST_VISIBILITY
	if ST_VISIBILITY(0x07) != 3 {
		t.Error("ST_VISIBILITY failed")
	}
}

func TestRelocationHelpers(t *testing.T) {
	// 32-bit
	info32 := ELF32_R_INFO(0x12, 0x34)
	if ELF32_R_SYM(info32) != 0x12 {
		t.Error("ELF32_R_SYM failed")
	}
	if ELF32_R_TYPE(info32) != 0x34 {
		t.Error("ELF32_R_TYPE failed")
	}

	// 64-bit
	info64 := ELF64_R_INFO(0x12345678, 0xABCDEF01)
	if ELF64_R_SYM(info64) != 0x12345678 {
		t.Error("ELF64_R_SYM failed")
	}
	if ELF64_R_TYPE(info64) != 0xABCDEF01 {
		t.Error("ELF64_R_TYPE failed")
	}
}

func TestParseELF32WithSegments(t *testing.T) {
	// Create ELF32 with one program header
	ehdr := makeELF32Header(t, EM_386, ET_EXEC, ELFOSABI_LINUX)

	// Update header to point to program headers
	binary.LittleEndian.PutUint32(ehdr[28:32], 52)  // e_phoff = 52 (right after header)
	binary.LittleEndian.PutUint16(ehdr[44:46], 1)   // e_phnum = 1

	// Create a PT_LOAD program header (32 bytes for 32-bit)
	phdr := make([]byte, 32)
	binary.LittleEndian.PutUint32(phdr[0:4], PT_LOAD)     // p_type
	binary.LittleEndian.PutUint32(phdr[4:8], 0)           // p_offset
	binary.LittleEndian.PutUint32(phdr[8:12], 0x8048000)  // p_vaddr
	binary.LittleEndian.PutUint32(phdr[12:16], 0x8048000) // p_paddr
	binary.LittleEndian.PutUint32(phdr[16:20], 0x1000)    // p_filesz
	binary.LittleEndian.PutUint32(phdr[20:24], 0x1000)    // p_memsz
	binary.LittleEndian.PutUint32(phdr[24:28], PF_R|PF_X) // p_flags
	binary.LittleEndian.PutUint32(phdr[28:32], 0x1000)    // p_align

	data := append(ehdr, phdr...)

	var segments []epc.Segment
	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
		OnSegmentFn: func(exeID string, index uint, seg epc.Segment) error {
			segments = append(segments, seg)
			return nil
		},
	}

	err := Parse(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(segments) != 1 {
		t.Fatalf("expected 1 segment, got %d", len(segments))
	}

	seg := segments[0]
	if seg.Type != PT_LOAD {
		t.Errorf("expected type PT_LOAD, got %d", seg.Type)
	}
	if seg.VAddr != 0x8048000 {
		t.Errorf("expected vaddr 0x8048000, got 0x%x", seg.VAddr)
	}
}

func TestExeIDAndIndex(t *testing.T) {
	data := makeELF64Header(t, EM_X86_64, ET_EXEC, ELFOSABI_LINUX)

	var gotExeID string
	var gotIndex uint

	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
		OnInfoFn: func(exeID string, index uint, info epc.Info) error {
			gotExeID = exeID
			gotIndex = index
			return nil
		},
	}

	err := Parse(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotExeID != "main" {
		t.Errorf("expected exeID 'main', got %q", gotExeID)
	}
	if gotIndex != 0 {
		t.Errorf("expected index 0, got %d", gotIndex)
	}
}

// Benchmark parsing a minimal ELF header
func BenchmarkParseMinimalELF64(b *testing.B) {
	data := make([]byte, 64)
	data[EI_MAG0] = ELFMAG0
	data[EI_MAG1] = ELFMAG1
	data[EI_MAG2] = ELFMAG2
	data[EI_MAG3] = ELFMAG3
	data[EI_CLASS] = ELFCLASS64
	data[EI_DATA] = ELFDATA2LSB
	data[EI_VERSION] = EV_CURRENT

	binary.LittleEndian.PutUint16(data[16:18], ET_EXEC)
	binary.LittleEndian.PutUint16(data[18:20], EM_X86_64)
	binary.LittleEndian.PutUint32(data[20:24], EV_CURRENT)

	reader := &bytesReaderAt{data: data}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := &epc.ParserConfig{Src: reader}
		_ = Parse(ctx, cfg)
	}
}

// Test that we handle truncated files gracefully
func TestParseTruncatedHeader(t *testing.T) {
	// Valid magic but truncated header
	data := []byte{0x7f, 'E', 'L', 'F', ELFCLASS64, ELFDATA2LSB, EV_CURRENT, 0}
	// Only 8 bytes, need 64 for ELF64 header

	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
	}

	err := Parse(context.Background(), cfg)
	if err == nil {
		t.Error("expected error for truncated header")
	}
}

// Test reading from an empty bytesReaderAt returns EOF properly
func TestBytesReaderAtEOF(t *testing.T) {
	r := &bytesReaderAt{data: []byte{}}

	buf := make([]byte, 10)
	n, err := r.ReadAt(buf, 0)
	if n != 0 || err != io.EOF {
		t.Errorf("expected (0, EOF), got (%d, %v)", n, err)
	}
}

// Test reading at negative offset
func TestBytesReaderAtNegativeOffset(t *testing.T) {
	r := &bytesReaderAt{data: []byte{1, 2, 3}}

	buf := make([]byte, 1)
	_, err := r.ReadAt(buf, -1)
	if err != io.EOF {
		t.Errorf("expected EOF for negative offset, got %v", err)
	}
}

// Test partial read
func TestBytesReaderAtPartialRead(t *testing.T) {
	r := &bytesReaderAt{data: []byte{1, 2, 3}}

	buf := make([]byte, 5)
	n, err := r.ReadAt(buf, 1)
	if n != 2 {
		t.Errorf("expected 2 bytes read, got %d", n)
	}
	if err != io.EOF {
		t.Errorf("expected EOF, got %v", err)
	}
	if !bytes.Equal(buf[:2], []byte{2, 3}) {
		t.Errorf("unexpected data: %v", buf[:2])
	}
}
