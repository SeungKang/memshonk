package peparser

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

// makePE32Header creates a minimal valid PE32 header
func makePE32Header(t *testing.T, machine uint16, characteristics uint16) []byte {
	t.Helper()

	// Calculate sizes
	dosHeaderSize := 64
	peSignatureSize := 4
	coffHeaderSize := 20
	optHeaderSize := 96 + 16*8 // Standard fields + 16 data directories
	sectionHeaderOffset := dosHeaderSize + peSignatureSize + coffHeaderSize + optHeaderSize

	// Total size: headers + at least one section header
	buf := make([]byte, sectionHeaderOffset+40)

	// DOS Header
	binary.LittleEndian.PutUint16(buf[0:2], DOSMagic)          // e_magic = "MZ"
	binary.LittleEndian.PutUint32(buf[0x3C:0x40], uint32(dosHeaderSize)) // e_lfanew

	// PE Signature
	peOffset := dosHeaderSize
	binary.LittleEndian.PutUint32(buf[peOffset:peOffset+4], PEMagic) // "PE\0\0"

	// COFF Header
	coffOffset := peOffset + 4
	binary.LittleEndian.PutUint16(buf[coffOffset:coffOffset+2], machine)         // Machine
	binary.LittleEndian.PutUint16(buf[coffOffset+2:coffOffset+4], 1)             // NumberOfSections
	binary.LittleEndian.PutUint32(buf[coffOffset+4:coffOffset+8], 0x5F3759DF)    // TimeDateStamp
	binary.LittleEndian.PutUint32(buf[coffOffset+8:coffOffset+12], 0)            // PointerToSymbolTable
	binary.LittleEndian.PutUint32(buf[coffOffset+12:coffOffset+16], 0)           // NumberOfSymbols
	binary.LittleEndian.PutUint16(buf[coffOffset+16:coffOffset+18], uint16(optHeaderSize)) // SizeOfOptionalHeader
	binary.LittleEndian.PutUint16(buf[coffOffset+18:coffOffset+20], characteristics)       // Characteristics

	// Optional Header (PE32)
	optOffset := coffOffset + 20
	binary.LittleEndian.PutUint16(buf[optOffset:optOffset+2], PE32Magic)         // Magic
	buf[optOffset+2] = 14                                                         // MajorLinkerVersion
	buf[optOffset+3] = 0                                                          // MinorLinkerVersion
	binary.LittleEndian.PutUint32(buf[optOffset+4:optOffset+8], 0x1000)          // SizeOfCode
	binary.LittleEndian.PutUint32(buf[optOffset+16:optOffset+20], 0x1000)        // AddressOfEntryPoint
	binary.LittleEndian.PutUint32(buf[optOffset+20:optOffset+24], 0x1000)        // BaseOfCode
	binary.LittleEndian.PutUint32(buf[optOffset+28:optOffset+32], 0x00400000)    // ImageBase
	binary.LittleEndian.PutUint32(buf[optOffset+32:optOffset+36], 0x1000)        // SectionAlignment
	binary.LittleEndian.PutUint32(buf[optOffset+36:optOffset+40], 0x200)         // FileAlignment
	binary.LittleEndian.PutUint16(buf[optOffset+40:optOffset+42], 6)             // MajorOperatingSystemVersion
	binary.LittleEndian.PutUint16(buf[optOffset+42:optOffset+44], 0)             // MinorOperatingSystemVersion
	binary.LittleEndian.PutUint16(buf[optOffset+48:optOffset+50], 6)             // MajorSubsystemVersion
	binary.LittleEndian.PutUint16(buf[optOffset+50:optOffset+52], 0)             // MinorSubsystemVersion
	binary.LittleEndian.PutUint32(buf[optOffset+56:optOffset+60], 0x10000)       // SizeOfImage
	binary.LittleEndian.PutUint32(buf[optOffset+60:optOffset+64], 0x200)         // SizeOfHeaders
	binary.LittleEndian.PutUint16(buf[optOffset+68:optOffset+70], IMAGE_SUBSYSTEM_WINDOWS_CUI) // Subsystem
	binary.LittleEndian.PutUint32(buf[optOffset+92:optOffset+96], 16)            // NumberOfRvaAndSizes

	// Section Header (.text)
	secOffset := sectionHeaderOffset
	copy(buf[secOffset:secOffset+8], []byte(".text\x00\x00\x00"))                // Name
	binary.LittleEndian.PutUint32(buf[secOffset+8:secOffset+12], 0x1000)         // VirtualSize
	binary.LittleEndian.PutUint32(buf[secOffset+12:secOffset+16], 0x1000)        // VirtualAddress
	binary.LittleEndian.PutUint32(buf[secOffset+16:secOffset+20], 0x200)         // SizeOfRawData
	binary.LittleEndian.PutUint32(buf[secOffset+20:secOffset+24], 0x200)         // PointerToRawData
	binary.LittleEndian.PutUint32(buf[secOffset+36:secOffset+40], IMAGE_SCN_CNT_CODE|IMAGE_SCN_MEM_EXECUTE|IMAGE_SCN_MEM_READ) // Characteristics

	return buf
}

// makePE64Header creates a minimal valid PE32+ (64-bit) header
func makePE64Header(t *testing.T, machine uint16, characteristics uint16) []byte {
	t.Helper()

	// Calculate sizes
	dosHeaderSize := 64
	peSignatureSize := 4
	coffHeaderSize := 20
	optHeaderSize := 112 + 16*8 // Standard fields + 16 data directories
	sectionHeaderOffset := dosHeaderSize + peSignatureSize + coffHeaderSize + optHeaderSize

	// Total size: headers + at least one section header
	buf := make([]byte, sectionHeaderOffset+40)

	// DOS Header
	binary.LittleEndian.PutUint16(buf[0:2], DOSMagic)          // e_magic = "MZ"
	binary.LittleEndian.PutUint32(buf[0x3C:0x40], uint32(dosHeaderSize)) // e_lfanew

	// PE Signature
	peOffset := dosHeaderSize
	binary.LittleEndian.PutUint32(buf[peOffset:peOffset+4], PEMagic) // "PE\0\0"

	// COFF Header
	coffOffset := peOffset + 4
	binary.LittleEndian.PutUint16(buf[coffOffset:coffOffset+2], machine)         // Machine
	binary.LittleEndian.PutUint16(buf[coffOffset+2:coffOffset+4], 1)             // NumberOfSections
	binary.LittleEndian.PutUint32(buf[coffOffset+4:coffOffset+8], 0x5F3759DF)    // TimeDateStamp
	binary.LittleEndian.PutUint32(buf[coffOffset+8:coffOffset+12], 0)            // PointerToSymbolTable
	binary.LittleEndian.PutUint32(buf[coffOffset+12:coffOffset+16], 0)           // NumberOfSymbols
	binary.LittleEndian.PutUint16(buf[coffOffset+16:coffOffset+18], uint16(optHeaderSize)) // SizeOfOptionalHeader
	binary.LittleEndian.PutUint16(buf[coffOffset+18:coffOffset+20], characteristics)       // Characteristics

	// Optional Header (PE32+)
	optOffset := coffOffset + 20
	binary.LittleEndian.PutUint16(buf[optOffset:optOffset+2], PE32PMagic)        // Magic
	buf[optOffset+2] = 14                                                         // MajorLinkerVersion
	buf[optOffset+3] = 0                                                          // MinorLinkerVersion
	binary.LittleEndian.PutUint32(buf[optOffset+4:optOffset+8], 0x1000)          // SizeOfCode
	binary.LittleEndian.PutUint32(buf[optOffset+16:optOffset+20], 0x1000)        // AddressOfEntryPoint
	binary.LittleEndian.PutUint32(buf[optOffset+20:optOffset+24], 0x1000)        // BaseOfCode
	binary.LittleEndian.PutUint64(buf[optOffset+24:optOffset+32], 0x140000000)   // ImageBase (64-bit)
	binary.LittleEndian.PutUint32(buf[optOffset+32:optOffset+36], 0x1000)        // SectionAlignment
	binary.LittleEndian.PutUint32(buf[optOffset+36:optOffset+40], 0x200)         // FileAlignment
	binary.LittleEndian.PutUint16(buf[optOffset+40:optOffset+42], 6)             // MajorOperatingSystemVersion
	binary.LittleEndian.PutUint16(buf[optOffset+42:optOffset+44], 0)             // MinorOperatingSystemVersion
	binary.LittleEndian.PutUint16(buf[optOffset+48:optOffset+50], 6)             // MajorSubsystemVersion
	binary.LittleEndian.PutUint16(buf[optOffset+50:optOffset+52], 0)             // MinorSubsystemVersion
	binary.LittleEndian.PutUint32(buf[optOffset+56:optOffset+60], 0x10000)       // SizeOfImage
	binary.LittleEndian.PutUint32(buf[optOffset+60:optOffset+64], 0x200)         // SizeOfHeaders
	binary.LittleEndian.PutUint16(buf[optOffset+68:optOffset+70], IMAGE_SUBSYSTEM_WINDOWS_CUI) // Subsystem
	binary.LittleEndian.PutUint32(buf[optOffset+108:optOffset+112], 16)          // NumberOfRvaAndSizes

	// Section Header (.text)
	secOffset := sectionHeaderOffset
	copy(buf[secOffset:secOffset+8], []byte(".text\x00\x00\x00"))                // Name
	binary.LittleEndian.PutUint32(buf[secOffset+8:secOffset+12], 0x1000)         // VirtualSize
	binary.LittleEndian.PutUint32(buf[secOffset+12:secOffset+16], 0x1000)        // VirtualAddress
	binary.LittleEndian.PutUint32(buf[secOffset+16:secOffset+20], 0x200)         // SizeOfRawData
	binary.LittleEndian.PutUint32(buf[secOffset+20:secOffset+24], 0x200)         // PointerToRawData
	binary.LittleEndian.PutUint32(buf[secOffset+36:secOffset+40], IMAGE_SCN_CNT_CODE|IMAGE_SCN_MEM_EXECUTE|IMAGE_SCN_MEM_READ) // Characteristics

	return buf
}

func TestParseInvalidDOSMagic(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too short", []byte{'M'}},
		{"wrong magic", []byte{'E', 'L', 'F', 0}},
		{"partial magic", []byte{'M', 'X'}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &epc.ParserConfig{
				Src: &bytesReaderAt{data: tt.data},
			}
			err := Parse(context.Background(), cfg)
			if err == nil {
				t.Error("expected error for invalid DOS magic")
			}
		})
	}
}

func TestParseInvalidPESignature(t *testing.T) {
	// Valid DOS header but wrong PE signature
	data := make([]byte, 128)
	binary.LittleEndian.PutUint16(data[0:2], DOSMagic)
	binary.LittleEndian.PutUint32(data[0x3C:0x40], 64)
	// Wrong PE signature at offset 64
	copy(data[64:68], []byte{'N', 'E', 0, 0})

	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
	}
	err := Parse(context.Background(), cfg)
	if err != ErrInvalidPEMagic {
		t.Errorf("expected ErrInvalidPEMagic, got %v", err)
	}
}

func TestParseInvalidOptionalMagic(t *testing.T) {
	data := make([]byte, 256)
	binary.LittleEndian.PutUint16(data[0:2], DOSMagic)
	binary.LittleEndian.PutUint32(data[0x3C:0x40], 64)
	binary.LittleEndian.PutUint32(data[64:68], PEMagic)
	// COFF header
	binary.LittleEndian.PutUint16(data[68:70], IMAGE_FILE_MACHINE_AMD64)
	binary.LittleEndian.PutUint16(data[68+16:68+18], 112) // SizeOfOptionalHeader
	// Wrong optional header magic
	binary.LittleEndian.PutUint16(data[88:90], 0x9999)

	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
	}
	err := Parse(context.Background(), cfg)
	if err != ErrInvalidOptMagic {
		t.Errorf("expected ErrInvalidOptMagic, got %v", err)
	}
}

func TestParsePE32Header(t *testing.T) {
	data := makePE32Header(t, IMAGE_FILE_MACHINE_I386, IMAGE_FILE_EXECUTABLE_IMAGE)

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

	if gotInfo.Format != "PE" {
		t.Errorf("expected format PE, got %s", gotInfo.Format)
	}
	if gotInfo.Class != 32 {
		t.Errorf("expected class 32, got %d", gotInfo.Class)
	}
	if gotInfo.Endian != "little" {
		t.Errorf("expected endian little, got %s", gotInfo.Endian)
	}
	if gotInfo.Type != "executable" {
		t.Errorf("expected type executable, got %s", gotInfo.Type)
	}
	if gotInfo.Machine != "i386" {
		t.Errorf("expected machine i386, got %s", gotInfo.Machine)
	}
	if gotInfo.EntryPoint != 0x00401000 {
		t.Errorf("expected entry point 0x00401000, got 0x%x", gotInfo.EntryPoint)
	}
}

func TestParsePE64Header(t *testing.T) {
	data := makePE64Header(t, IMAGE_FILE_MACHINE_AMD64, IMAGE_FILE_EXECUTABLE_IMAGE)

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

	if gotInfo.Format != "PE" {
		t.Errorf("expected format PE, got %s", gotInfo.Format)
	}
	if gotInfo.Class != 64 {
		t.Errorf("expected class 64, got %d", gotInfo.Class)
	}
	if gotInfo.Machine != "x86_64" {
		t.Errorf("expected machine x86_64, got %s", gotInfo.Machine)
	}
	if gotInfo.EntryPoint != 0x140001000 {
		t.Errorf("expected entry point 0x140001000, got 0x%x", gotInfo.EntryPoint)
	}
}

func TestParsePE64DLL(t *testing.T) {
	data := makePE64Header(t, IMAGE_FILE_MACHINE_AMD64, IMAGE_FILE_EXECUTABLE_IMAGE|IMAGE_FILE_DLL)

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

	if gotInfo.Type != "dll" {
		t.Errorf("expected type dll, got %s", gotInfo.Type)
	}
}

func TestParseSections(t *testing.T) {
	data := makePE64Header(t, IMAGE_FILE_MACHINE_AMD64, IMAGE_FILE_EXECUTABLE_IMAGE)

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

	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}

	sec := sections[0]
	if sec.Name != ".text" {
		t.Errorf("expected section name .text, got %s", sec.Name)
	}
	if !sec.IsCode {
		t.Error("expected .text to be marked as code")
	}
	if sec.Size != 0x1000 {
		t.Errorf("expected size 0x1000, got 0x%x", sec.Size)
	}
}

func TestParseContextCancellation(t *testing.T) {
	data := makePE64Header(t, IMAGE_FILE_MACHINE_AMD64, IMAGE_FILE_EXECUTABLE_IMAGE)

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

func TestParseNoCallbacks(t *testing.T) {
	data := makePE64Header(t, IMAGE_FILE_MACHINE_AMD64, IMAGE_FILE_EXECUTABLE_IMAGE)

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
	data := makePE64Header(t, IMAGE_FILE_MACHINE_AMD64, IMAGE_FILE_EXECUTABLE_IMAGE)

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

func TestExeIDAndIndex(t *testing.T) {
	data := makePE64Header(t, IMAGE_FILE_MACHINE_AMD64, IMAGE_FILE_EXECUTABLE_IMAGE)

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

func TestParseTruncatedDOSHeader(t *testing.T) {
	// Valid MZ but truncated before e_lfanew
	data := []byte{'M', 'Z', 0, 0, 0, 0}

	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
	}

	err := Parse(context.Background(), cfg)
	if err == nil {
		t.Error("expected error for truncated DOS header")
	}
}

func TestMachineStrings(t *testing.T) {
	tests := []struct {
		machine  uint16
		expected string
	}{
		{IMAGE_FILE_MACHINE_UNKNOWN, "unknown"},
		{IMAGE_FILE_MACHINE_I386, "i386"},
		{IMAGE_FILE_MACHINE_AMD64, "x86_64"},
		{IMAGE_FILE_MACHINE_ARM, "arm"},
		{IMAGE_FILE_MACHINE_ARM64, "arm64"},
		{IMAGE_FILE_MACHINE_IA64, "ia64"},
		{IMAGE_FILE_MACHINE_RISCV64, "riscv64"},
		{0xFFFF, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := MachineString(tt.machine)
			if got != tt.expected {
				t.Errorf("MachineString(%d) = %s, want %s", tt.machine, got, tt.expected)
			}
		})
	}
}

func TestSubsystemStrings(t *testing.T) {
	tests := []struct {
		subsystem uint16
		expected  string
	}{
		{IMAGE_SUBSYSTEM_UNKNOWN, "unknown"},
		{IMAGE_SUBSYSTEM_NATIVE, "native"},
		{IMAGE_SUBSYSTEM_WINDOWS_GUI, "windows_gui"},
		{IMAGE_SUBSYSTEM_WINDOWS_CUI, "windows_cui"},
		{IMAGE_SUBSYSTEM_EFI_APPLICATION, "efi_application"},
		{0xFFFF, "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := SubsystemString(tt.subsystem)
			if got != tt.expected {
				t.Errorf("SubsystemString(%d) = %s, want %s", tt.subsystem, got, tt.expected)
			}
		})
	}
}

func TestSectionName(t *testing.T) {
	tests := []struct {
		input    [8]byte
		expected string
	}{
		{[8]byte{'.', 't', 'e', 'x', 't', 0, 0, 0}, ".text"},
		{[8]byte{'.', 'd', 'a', 't', 'a', 0, 0, 0}, ".data"},
		{[8]byte{'.', 'r', 'd', 'a', 't', 'a', 0, 0}, ".rdata"},
		{[8]byte{'l', 'o', 'n', 'g', 'n', 'a', 'm', 'e'}, "longname"}, // No null terminator
		{[8]byte{0, 0, 0, 0, 0, 0, 0, 0}, ""},
	}

	for _, tt := range tests {
		got := SectionName(tt.input)
		if got != tt.expected {
			t.Errorf("SectionName(%v) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestOrdinalHelpers(t *testing.T) {
	// 32-bit ordinal import
	thunk32Ordinal := uint32(0x80001234)
	if !IsOrdinalImport32(thunk32Ordinal) {
		t.Error("IsOrdinalImport32 should return true for ordinal import")
	}
	if Ordinal32(thunk32Ordinal) != 0x1234 {
		t.Errorf("Ordinal32 = %d, want %d", Ordinal32(thunk32Ordinal), 0x1234)
	}

	// 32-bit name import
	thunk32Name := uint32(0x00001234)
	if IsOrdinalImport32(thunk32Name) {
		t.Error("IsOrdinalImport32 should return false for name import")
	}
	if HintNameRVA32(thunk32Name) != 0x1234 {
		t.Errorf("HintNameRVA32 = %d, want %d", HintNameRVA32(thunk32Name), 0x1234)
	}

	// 64-bit ordinal import
	thunk64Ordinal := uint64(0x8000000000001234)
	if !IsOrdinalImport64(thunk64Ordinal) {
		t.Error("IsOrdinalImport64 should return true for ordinal import")
	}
	if Ordinal64(thunk64Ordinal) != 0x1234 {
		t.Errorf("Ordinal64 = %d, want %d", Ordinal64(thunk64Ordinal), 0x1234)
	}

	// 64-bit name import
	thunk64Name := uint64(0x0000000000001234)
	if IsOrdinalImport64(thunk64Name) {
		t.Error("IsOrdinalImport64 should return false for name import")
	}
	if HintNameRVA64(thunk64Name) != 0x1234 {
		t.Errorf("HintNameRVA64 = %d, want %d", HintNameRVA64(thunk64Name), 0x1234)
	}
}

func TestRelocHelpers(t *testing.T) {
	// Type 3 (HIGHLOW), offset 0x123
	entry := uint16(0x3123)
	if RelocType(entry) != 3 {
		t.Errorf("RelocType = %d, want 3", RelocType(entry))
	}
	if RelocOffset(entry) != 0x123 {
		t.Errorf("RelocOffset = %d, want 0x123", RelocOffset(entry))
	}

	// Type 10 (DIR64), offset 0xABC
	entry2 := uint16(0xAABC)
	if RelocType(entry2) != 10 {
		t.Errorf("RelocType = %d, want 10", RelocType(entry2))
	}
	if RelocOffset(entry2) != 0xABC {
		t.Errorf("RelocOffset = %d, want 0xABC", RelocOffset(entry2))
	}
}

func TestSectionCharacteristics(t *testing.T) {
	// Code section
	codeFlags := uint32(IMAGE_SCN_CNT_CODE | IMAGE_SCN_MEM_EXECUTE | IMAGE_SCN_MEM_READ)
	if !IsSectionCode(codeFlags) {
		t.Error("IsSectionCode should return true for code section")
	}
	if !IsSectionExecutable(codeFlags) {
		t.Error("IsSectionExecutable should return true for executable section")
	}
	if !IsSectionReadable(codeFlags) {
		t.Error("IsSectionReadable should return true for readable section")
	}
	if IsSectionWritable(codeFlags) {
		t.Error("IsSectionWritable should return false for non-writable section")
	}

	// Data section
	dataFlags := uint32(IMAGE_SCN_CNT_INITIALIZED_DATA | IMAGE_SCN_MEM_READ | IMAGE_SCN_MEM_WRITE)
	if !IsSectionData(dataFlags) {
		t.Error("IsSectionData should return true for data section")
	}
	if !IsSectionWritable(dataFlags) {
		t.Error("IsSectionWritable should return true for writable section")
	}

	// BSS section
	bssFlags := uint32(IMAGE_SCN_CNT_UNINITIALIZED_DATA | IMAGE_SCN_MEM_READ | IMAGE_SCN_MEM_WRITE)
	if !IsSectionBSS(bssFlags) {
		t.Error("IsSectionBSS should return true for BSS section")
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

// Benchmark parsing a minimal PE header
func BenchmarkParseMinimalPE64(b *testing.B) {
	data := makePE64Header(&testing.T{}, IMAGE_FILE_MACHINE_AMD64, IMAGE_FILE_EXECUTABLE_IMAGE)
	reader := &bytesReaderAt{data: data}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := &epc.ParserConfig{Src: reader}
		_ = Parse(ctx, cfg)
	}
}
