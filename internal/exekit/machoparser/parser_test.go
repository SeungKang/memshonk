package machoparser

import (
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

// makeMachO64Header creates a minimal valid 64-bit Mach-O header
func makeMachO64Header(cpuType int32, fileType uint32) []byte {
	buf := make([]byte, 32)

	binary.LittleEndian.PutUint32(buf[0:4], MH_MAGIC_64)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(cpuType))
	binary.LittleEndian.PutUint32(buf[8:12], 0)  // cpusubtype
	binary.LittleEndian.PutUint32(buf[12:16], fileType)
	binary.LittleEndian.PutUint32(buf[16:20], 0) // ncmds
	binary.LittleEndian.PutUint32(buf[20:24], 0) // sizeofcmds
	binary.LittleEndian.PutUint32(buf[24:28], 0) // flags
	binary.LittleEndian.PutUint32(buf[28:32], 0) // reserved

	return buf
}

// makeMachO32Header creates a minimal valid 32-bit Mach-O header
func makeMachO32Header(cpuType int32, fileType uint32) []byte {
	buf := make([]byte, 28)

	binary.LittleEndian.PutUint32(buf[0:4], MH_MAGIC)
	binary.LittleEndian.PutUint32(buf[4:8], uint32(cpuType))
	binary.LittleEndian.PutUint32(buf[8:12], 0)  // cpusubtype
	binary.LittleEndian.PutUint32(buf[12:16], fileType)
	binary.LittleEndian.PutUint32(buf[16:20], 0) // ncmds
	binary.LittleEndian.PutUint32(buf[20:24], 0) // sizeofcmds
	binary.LittleEndian.PutUint32(buf[24:28], 0) // flags

	return buf
}

func TestParseInvalidMagic(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too short", []byte{0xfe, 0xed}},
		{"wrong magic", []byte{0x00, 0x00, 0x00, 0x00}},
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

func TestParseMachO64Header(t *testing.T) {
	data := makeMachO64Header(CPU_TYPE_X86_64, MH_EXECUTE)

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

	if gotInfo.Format != "Mach-O" {
		t.Errorf("expected format Mach-O, got %s", gotInfo.Format)
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
	if gotInfo.OSABI != "Darwin" {
		t.Errorf("expected OSABI Darwin, got %s", gotInfo.OSABI)
	}
}

func TestParseMachO32Header(t *testing.T) {
	data := makeMachO32Header(CPU_TYPE_I386, MH_DYLIB)

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

	if gotInfo.Format != "Mach-O" {
		t.Errorf("expected format Mach-O, got %s", gotInfo.Format)
	}
	if gotInfo.Class != 32 {
		t.Errorf("expected class 32, got %d", gotInfo.Class)
	}
	if gotInfo.Type != "dylib" {
		t.Errorf("expected type dylib, got %s", gotInfo.Type)
	}
	if gotInfo.Machine != "i386" {
		t.Errorf("expected machine i386, got %s", gotInfo.Machine)
	}
}

func TestParseMachO64BigEndian(t *testing.T) {
	buf := make([]byte, 32)

	binary.BigEndian.PutUint32(buf[0:4], MH_MAGIC_64)
	binary.BigEndian.PutUint32(buf[4:8], uint32(CPU_TYPE_POWERPC64))
	binary.BigEndian.PutUint32(buf[8:12], 0)
	binary.BigEndian.PutUint32(buf[12:16], MH_EXECUTE)
	binary.BigEndian.PutUint32(buf[16:20], 0)
	binary.BigEndian.PutUint32(buf[20:24], 0)
	binary.BigEndian.PutUint32(buf[24:28], 0)
	binary.BigEndian.PutUint32(buf[28:32], 0)

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
}

func TestParseContextCancellation(t *testing.T) {
	data := makeMachO64Header(CPU_TYPE_X86_64, MH_EXECUTE)

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

func TestParseMachOWithSegment64(t *testing.T) {
	// Create Mach-O header with one LC_SEGMENT_64 command
	hdr := makeMachO64Header(CPU_TYPE_X86_64, MH_EXECUTE)

	// Update ncmds and sizeofcmds
	binary.LittleEndian.PutUint32(hdr[16:20], 1)  // ncmds = 1
	binary.LittleEndian.PutUint32(hdr[20:24], 72) // sizeofcmds = 72 (segment_command_64 size)

	// Create LC_SEGMENT_64 command (72 bytes, no sections)
	seg := make([]byte, 72)
	binary.LittleEndian.PutUint32(seg[0:4], LC_SEGMENT_64)
	binary.LittleEndian.PutUint32(seg[4:8], 72) // cmdsize
	copy(seg[8:24], "__TEXT")                   // segname
	binary.LittleEndian.PutUint64(seg[24:32], 0x100000000) // vmaddr
	binary.LittleEndian.PutUint64(seg[32:40], 0x1000)      // vmsize
	binary.LittleEndian.PutUint64(seg[40:48], 0)           // fileoff
	binary.LittleEndian.PutUint64(seg[48:56], 0x1000)      // filesize
	binary.LittleEndian.PutUint32(seg[56:60], VM_PROT_READ|VM_PROT_EXECUTE) // maxprot
	binary.LittleEndian.PutUint32(seg[60:64], VM_PROT_READ|VM_PROT_EXECUTE) // initprot
	binary.LittleEndian.PutUint32(seg[64:68], 0) // nsects
	binary.LittleEndian.PutUint32(seg[68:72], 0) // flags

	data := append(hdr, seg...)

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

	seg0 := segments[0]
	if seg0.VAddr != 0x100000000 {
		t.Errorf("expected vaddr 0x100000000, got 0x%x", seg0.VAddr)
	}
	if seg0.MemSize != 0x1000 {
		t.Errorf("expected memsize 0x1000, got 0x%x", seg0.MemSize)
	}
}

func TestParseMachOWithSection64(t *testing.T) {
	// Create Mach-O header with one LC_SEGMENT_64 command containing one section
	hdr := makeMachO64Header(CPU_TYPE_X86_64, MH_EXECUTE)

	// Update ncmds and sizeofcmds
	binary.LittleEndian.PutUint32(hdr[16:20], 1)   // ncmds = 1
	binary.LittleEndian.PutUint32(hdr[20:24], 152) // sizeofcmds = 72 + 80 (segment + section)

	// Create LC_SEGMENT_64 command with 1 section
	seg := make([]byte, 72)
	binary.LittleEndian.PutUint32(seg[0:4], LC_SEGMENT_64)
	binary.LittleEndian.PutUint32(seg[4:8], 152) // cmdsize
	copy(seg[8:24], "__TEXT")
	binary.LittleEndian.PutUint64(seg[24:32], 0x100000000)
	binary.LittleEndian.PutUint64(seg[32:40], 0x1000)
	binary.LittleEndian.PutUint64(seg[40:48], 0)
	binary.LittleEndian.PutUint64(seg[48:56], 0x1000)
	binary.LittleEndian.PutUint32(seg[56:60], VM_PROT_READ|VM_PROT_EXECUTE)
	binary.LittleEndian.PutUint32(seg[60:64], VM_PROT_READ|VM_PROT_EXECUTE)
	binary.LittleEndian.PutUint32(seg[64:68], 1) // nsects = 1
	binary.LittleEndian.PutUint32(seg[68:72], 0)

	// Create section_64 (80 bytes)
	sect := make([]byte, 80)
	copy(sect[0:16], "__text")          // sectname
	copy(sect[16:32], "__TEXT")         // segname
	binary.LittleEndian.PutUint64(sect[32:40], 0x100000100) // addr
	binary.LittleEndian.PutUint64(sect[40:48], 0x100)       // size
	binary.LittleEndian.PutUint32(sect[48:52], 0x100)       // offset
	binary.LittleEndian.PutUint32(sect[52:56], 4)           // align (2^4 = 16)
	binary.LittleEndian.PutUint32(sect[56:60], S_ATTR_PURE_INSTRUCTIONS|S_REGULAR) // flags

	data := append(hdr, seg...)
	data = append(data, sect...)

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

	sec0 := sections[0]
	if sec0.Name != "__text" {
		t.Errorf("expected section name __text, got %s", sec0.Name)
	}
	if sec0.Addr != 0x100000100 {
		t.Errorf("expected addr 0x100000100, got 0x%x", sec0.Addr)
	}
	if sec0.Size != 0x100 {
		t.Errorf("expected size 0x100, got 0x%x", sec0.Size)
	}
	if !sec0.IsCode {
		t.Error("expected section to be marked as code")
	}
}

func TestParseMachOWithDylib(t *testing.T) {
	// Create Mach-O header with one LC_LOAD_DYLIB command
	hdr := makeMachO64Header(CPU_TYPE_X86_64, MH_EXECUTE)

	dylibName := "/usr/lib/libSystem.B.dylib\x00"
	cmdSize := uint32(24 + len(dylibName))
	// Pad to 8-byte alignment
	if cmdSize%8 != 0 {
		cmdSize += 8 - (cmdSize % 8)
	}

	binary.LittleEndian.PutUint32(hdr[16:20], 1)       // ncmds = 1
	binary.LittleEndian.PutUint32(hdr[20:24], cmdSize) // sizeofcmds

	// Create LC_LOAD_DYLIB command
	dylib := make([]byte, cmdSize)
	binary.LittleEndian.PutUint32(dylib[0:4], LC_LOAD_DYLIB)
	binary.LittleEndian.PutUint32(dylib[4:8], cmdSize)
	binary.LittleEndian.PutUint32(dylib[8:12], 24) // name offset
	binary.LittleEndian.PutUint32(dylib[12:16], 0) // timestamp
	binary.LittleEndian.PutUint32(dylib[16:20], 0) // current_version
	binary.LittleEndian.PutUint32(dylib[20:24], 0) // compatibility_version
	copy(dylib[24:], dylibName)

	data := append(hdr, dylib...)

	var libs []epc.ImportedLibrary
	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
		OnImportedLibraryFn: func(exeID string, index uint, lib epc.ImportedLibrary) error {
			libs = append(libs, lib)
			return nil
		},
	}

	err := Parse(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(libs) != 1 {
		t.Fatalf("expected 1 imported library, got %d", len(libs))
	}

	if libs[0].Name != "/usr/lib/libSystem.B.dylib" {
		t.Errorf("expected library name /usr/lib/libSystem.B.dylib, got %s", libs[0].Name)
	}
}

func TestParseNoCallbacks(t *testing.T) {
	data := makeMachO64Header(CPU_TYPE_X86_64, MH_EXECUTE)

	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
	}

	err := Parse(context.Background(), cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCallbackError(t *testing.T) {
	data := makeMachO64Header(CPU_TYPE_X86_64, MH_EXECUTE)

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

func TestFileTypeStrings(t *testing.T) {
	tests := []struct {
		fileType uint32
		expected string
	}{
		{MH_OBJECT, "object"},
		{MH_EXECUTE, "executable"},
		{MH_DYLIB, "dylib"},
		{MH_BUNDLE, "bundle"},
		{MH_CORE, "core"},
		{MH_DYLINKER, "dylinker"},
		{MH_DSYM, "dsym"},
		{MH_KEXT_BUNDLE, "kext"},
		{9999, "unknown(9999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := fileTypeString(tt.fileType)
			if got != tt.expected {
				t.Errorf("fileTypeString(%d) = %s, want %s", tt.fileType, got, tt.expected)
			}
		})
	}
}

func TestCPUTypeStrings(t *testing.T) {
	tests := []struct {
		cpuType  int32
		expected string
	}{
		{CPU_TYPE_I386, "i386"},
		{CPU_TYPE_X86_64, "x86_64"},
		{CPU_TYPE_ARM, "arm"},
		{CPU_TYPE_ARM64, "arm64"},
		{CPU_TYPE_ARM64_32, "arm64_32"},
		{CPU_TYPE_POWERPC, "ppc"},
		{CPU_TYPE_POWERPC64, "ppc64"},
		{9999, "cpu(9999)"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := cpuTypeString(tt.cpuType)
			if got != tt.expected {
				t.Errorf("cpuTypeString(%d) = %s, want %s", tt.cpuType, got, tt.expected)
			}
		})
	}
}

func TestSegmentName(t *testing.T) {
	tests := []struct {
		name     [16]byte
		expected string
	}{
		{[16]byte{'_', '_', 'T', 'E', 'X', 'T'}, "__TEXT"},
		{[16]byte{'_', '_', 'D', 'A', 'T', 'A'}, "__DATA"},
		{[16]byte{}, ""},
		{[16]byte{'a', 'b', 'c', 'd', 'e', 'f', 'g', 'h', 'i', 'j', 'k', 'l', 'm', 'n', 'o', 'p'}, "abcdefghijklmnop"},
	}

	for _, tt := range tests {
		got := SegmentName(tt.name)
		if got != tt.expected {
			t.Errorf("SegmentName(%v) = %s, want %s", tt.name, got, tt.expected)
		}
	}
}

func TestGetLibraryOrdinal(t *testing.T) {
	tests := []struct {
		nDesc    int16
		expected uint8
	}{
		{0x0000, 0},
		{0x0100, 1},
		{0x0200, 2},
		{-256, 0xff}, // 0xff00 as int16
	}

	for _, tt := range tests {
		got := GET_LIBRARY_ORDINAL(tt.nDesc)
		if got != tt.expected {
			t.Errorf("GET_LIBRARY_ORDINAL(%d) = %d, want %d", tt.nDesc, got, tt.expected)
		}
	}
}

func TestExeIDAndIndex(t *testing.T) {
	data := makeMachO64Header(CPU_TYPE_X86_64, MH_EXECUTE)

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

func TestParseMachO32WithSegment(t *testing.T) {
	// Create 32-bit Mach-O header with one LC_SEGMENT command
	hdr := makeMachO32Header(CPU_TYPE_I386, MH_EXECUTE)

	// Update ncmds and sizeofcmds
	binary.LittleEndian.PutUint32(hdr[16:20], 1)  // ncmds = 1
	binary.LittleEndian.PutUint32(hdr[20:24], 56) // sizeofcmds = 56 (segment_command size)

	// Create LC_SEGMENT command (56 bytes, no sections)
	seg := make([]byte, 56)
	binary.LittleEndian.PutUint32(seg[0:4], LC_SEGMENT)
	binary.LittleEndian.PutUint32(seg[4:8], 56) // cmdsize
	copy(seg[8:24], "__DATA")                   // segname
	binary.LittleEndian.PutUint32(seg[24:28], 0x1000) // vmaddr
	binary.LittleEndian.PutUint32(seg[28:32], 0x1000) // vmsize
	binary.LittleEndian.PutUint32(seg[32:36], 0x1000) // fileoff
	binary.LittleEndian.PutUint32(seg[36:40], 0x1000) // filesize
	binary.LittleEndian.PutUint32(seg[40:44], VM_PROT_READ|VM_PROT_WRITE) // maxprot
	binary.LittleEndian.PutUint32(seg[44:48], VM_PROT_READ|VM_PROT_WRITE) // initprot
	binary.LittleEndian.PutUint32(seg[48:52], 0) // nsects
	binary.LittleEndian.PutUint32(seg[52:56], 0) // flags

	data := append(hdr, seg...)

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

	seg0 := segments[0]
	if seg0.VAddr != 0x1000 {
		t.Errorf("expected vaddr 0x1000, got 0x%x", seg0.VAddr)
	}
}

func TestBoolToBinding(t *testing.T) {
	if boolToBinding(true) != 1 {
		t.Error("expected 1 for external symbol")
	}
	if boolToBinding(false) != 0 {
		t.Error("expected 0 for local symbol")
	}
}

func TestEndianString(t *testing.T) {
	if endianString(binary.LittleEndian) != "little" {
		t.Error("expected 'little' for LittleEndian")
	}
	if endianString(binary.BigEndian) != "big" {
		t.Error("expected 'big' for BigEndian")
	}
}

// Benchmark parsing a minimal Mach-O header
func BenchmarkParseMinimalMachO64(b *testing.B) {
	data := makeMachO64Header(CPU_TYPE_X86_64, MH_EXECUTE)
	reader := &bytesReaderAt{data: data}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cfg := &epc.ParserConfig{Src: reader}
		_ = Parse(ctx, cfg)
	}
}

func TestParseTruncatedHeader(t *testing.T) {
	// Valid magic but truncated header
	data := []byte{0xfe, 0xed, 0xfa, 0xcf, 0x00, 0x00} // MH_MAGIC_64 + partial

	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
	}

	err := Parse(context.Background(), cfg)
	if err == nil {
		t.Error("expected error for truncated header")
	}
}

func TestParseTruncatedLoadCommand(t *testing.T) {
	hdr := makeMachO64Header(CPU_TYPE_X86_64, MH_EXECUTE)
	binary.LittleEndian.PutUint32(hdr[16:20], 1)  // ncmds = 1
	binary.LittleEndian.PutUint32(hdr[20:24], 72) // sizeofcmds

	// Add only partial load command
	partialCmd := []byte{0x19, 0x00, 0x00, 0x00} // LC_SEGMENT_64 but truncated

	data := append(hdr, partialCmd...)

	cfg := &epc.ParserConfig{
		Src: &bytesReaderAt{data: data},
	}

	err := Parse(context.Background(), cfg)
	if err == nil {
		t.Error("expected error for truncated load command")
	}
}
