package exekit

import (
	"context"
	"io"
	"testing"
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

func TestParseNilConfig(t *testing.T) {
	err := Parse(context.Background(), nil)
	if err != ErrNoSource {
		t.Errorf("expected ErrNoSource, got %v", err)
	}
}

func TestParseNilSrc(t *testing.T) {
	cfg := &ParserConfig{Src: nil}
	err := Parse(context.Background(), cfg)
	if err != ErrNoSource {
		t.Errorf("expected ErrNoSource, got %v", err)
	}
}

func TestParseEmptyFile(t *testing.T) {
	cfg := &ParserConfig{
		Src: &bytesReaderAt{data: []byte{}},
	}
	err := Parse(context.Background(), cfg)
	if err == nil {
		t.Error("expected error for empty file")
	}
}

func TestParseTooShort(t *testing.T) {
	cfg := &ParserConfig{
		Src: &bytesReaderAt{data: []byte{0x7f}},
	}
	err := Parse(context.Background(), cfg)
	if err == nil {
		t.Error("expected error for file too short")
	}
}

func TestParseUnknownFormat(t *testing.T) {
	cfg := &ParserConfig{
		Src: &bytesReaderAt{data: []byte{0x00, 0x00, 0x00, 0x00}},
	}
	err := Parse(context.Background(), cfg)
	if err != ErrUnknownFormat {
		t.Errorf("expected ErrUnknownFormat, got %v", err)
	}
}

func TestParseELFMagic(t *testing.T) {
	// Valid ELF magic but incomplete header - should fail with ELF-specific error
	data := []byte{0x7f, 'E', 'L', 'F', 2, 1, 1, 0} // ELF64, little-endian
	cfg := &ParserConfig{
		Src: &bytesReaderAt{data: data},
	}
	err := Parse(context.Background(), cfg)
	// Should fail because header is truncated, but not with ErrUnknownFormat
	if err == ErrUnknownFormat {
		t.Error("ELF magic should be recognized")
	}
	if err == nil {
		t.Error("expected error for truncated ELF")
	}
}

func TestParseMachOMagic(t *testing.T) {
	testCases := []struct {
		name  string
		magic []byte
	}{
		{"MH_MAGIC", []byte{0xfe, 0xed, 0xfa, 0xce}},
		{"MH_MAGIC_64", []byte{0xfe, 0xed, 0xfa, 0xcf}},
		{"MH_CIGAM", []byte{0xce, 0xfa, 0xed, 0xfe}},
		{"MH_CIGAM_64", []byte{0xcf, 0xfa, 0xed, 0xfe}},
		{"FAT_MAGIC", []byte{0xca, 0xfe, 0xba, 0xbe}},
		{"FAT_CIGAM", []byte{0xbe, 0xba, 0xfe, 0xca}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &ParserConfig{
				Src: &bytesReaderAt{data: tc.magic},
			}
			err := Parse(context.Background(), cfg)
			// Should fail with "Mach-O format: unknown executable format"
			// because Mach-O parser is not implemented
			if err == nil {
				t.Error("expected error for unimplemented Mach-O")
			}
			if err == ErrUnknownFormat {
				t.Error("Mach-O magic should be recognized")
			}
		})
	}
}

func TestParsePEMagic(t *testing.T) {
	cfg := &ParserConfig{
		Src: &bytesReaderAt{data: []byte{'M', 'Z', 0x00, 0x00}},
	}
	err := Parse(context.Background(), cfg)
	// Should fail with "PE format: unknown executable format"
	// because PE parser is not implemented
	if err == nil {
		t.Error("expected error for unimplemented PE")
	}
	if err == ErrUnknownFormat {
		t.Error("PE magic should be recognized")
	}
}

func TestMatchMagic(t *testing.T) {
	tests := []struct {
		data     []byte
		magic    []byte
		expected bool
	}{
		{[]byte{0x7f, 'E', 'L', 'F'}, []byte{0x7f, 'E', 'L', 'F'}, true},
		{[]byte{0x7f, 'E', 'L', 'F', 0x00}, []byte{0x7f, 'E', 'L', 'F'}, true},
		{[]byte{0x7f, 'E', 'L', 'X'}, []byte{0x7f, 'E', 'L', 'F'}, false},
		{[]byte{0x7f, 'E'}, []byte{0x7f, 'E', 'L', 'F'}, false},
		{[]byte{}, []byte{0x7f}, false},
		{[]byte{'M', 'Z'}, []byte{'M', 'Z'}, true},
	}

	for _, tt := range tests {
		got := matchMagic(tt.data, tt.magic)
		if got != tt.expected {
			t.Errorf("matchMagic(%v, %v) = %v, want %v", tt.data, tt.magic, got, tt.expected)
		}
	}
}

func TestParseValidELF64(t *testing.T) {
	// Create a minimal valid ELF64 header
	data := make([]byte, 64)
	data[0] = 0x7f
	data[1] = 'E'
	data[2] = 'L'
	data[3] = 'F'
	data[4] = 2 // ELFCLASS64
	data[5] = 1 // ELFDATA2LSB
	data[6] = 1 // EV_CURRENT

	// e_type = ET_EXEC
	data[16] = 2
	data[17] = 0

	// e_machine = EM_X86_64
	data[18] = 62
	data[19] = 0

	// e_version
	data[20] = 1

	var gotInfo Info
	cfg := &ParserConfig{
		Src: &bytesReaderAt{data: data},
		OnInfoFn: func(exeID string, index uint, info Info) error {
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
	if gotInfo.Machine != "x86_64" {
		t.Errorf("expected machine x86_64, got %s", gotInfo.Machine)
	}
}

func TestTypeAliases(t *testing.T) {
	// Verify that the type aliases work correctly
	var _ ParserConfig
	var _ ExeFmtOption
	var _ Info
	var _ Function
	var _ ImportedCode
	var _ ImportedLibrary
	var _ Reloc
	var _ Section
	var _ Segment
	var _ String
	var _ Symbol

	// Verify generic CallbackFn alias works
	var _ CallbackFn[Info]
	var _ CallbackFn[Symbol]

	// These should compile without error
	cfg := ParserConfig{}
	_ = cfg.Src
	_ = cfg.OnInfoFn
	_ = cfg.OnFunctionFn
	_ = cfg.OnImportedCodeFn
	_ = cfg.OnImportedLibraryFn
	_ = cfg.OnRelocFn
	_ = cfg.OnSectionFn
	_ = cfg.OnSegmentFn
	_ = cfg.OnStringFn
	_ = cfg.OnSymbolFn
	_ = cfg.OptCPU
	_ = cfg.OptBits
	_ = cfg.OptExeSpecific
}
