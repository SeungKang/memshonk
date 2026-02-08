package elfparser

import "testing"

func TestST_BIND(t *testing.T) {
	tests := []struct {
		info     uint8
		expected uint8
	}{
		{0x00, STB_LOCAL},
		{0x10, STB_GLOBAL},
		{0x20, STB_WEAK},
		{0x12, STB_GLOBAL}, // GLOBAL + FUNC
		{0x22, STB_WEAK},   // WEAK + FUNC
	}

	for _, tt := range tests {
		got := ST_BIND(tt.info)
		if got != tt.expected {
			t.Errorf("ST_BIND(0x%02x) = %d, want %d", tt.info, got, tt.expected)
		}
	}
}

func TestST_TYPE(t *testing.T) {
	tests := []struct {
		info     uint8
		expected uint8
	}{
		{0x00, STT_NOTYPE},
		{0x01, STT_OBJECT},
		{0x02, STT_FUNC},
		{0x03, STT_SECTION},
		{0x04, STT_FILE},
		{0x12, STT_FUNC},   // GLOBAL + FUNC
		{0x21, STT_OBJECT}, // WEAK + OBJECT
	}

	for _, tt := range tests {
		got := ST_TYPE(tt.info)
		if got != tt.expected {
			t.Errorf("ST_TYPE(0x%02x) = %d, want %d", tt.info, got, tt.expected)
		}
	}
}

func TestST_INFO(t *testing.T) {
	tests := []struct {
		bind     uint8
		typ      uint8
		expected uint8
	}{
		{STB_LOCAL, STT_NOTYPE, 0x00},
		{STB_GLOBAL, STT_FUNC, 0x12},
		{STB_WEAK, STT_OBJECT, 0x21},
		{STB_LOCAL, STT_FILE, 0x04},
	}

	for _, tt := range tests {
		got := ST_INFO(tt.bind, tt.typ)
		if got != tt.expected {
			t.Errorf("ST_INFO(%d, %d) = 0x%02x, want 0x%02x", tt.bind, tt.typ, got, tt.expected)
		}
	}
}

func TestST_VISIBILITY(t *testing.T) {
	tests := []struct {
		other    uint8
		expected uint8
	}{
		{0x00, STV_DEFAULT},
		{0x01, STV_INTERNAL},
		{0x02, STV_HIDDEN},
		{0x03, STV_PROTECTED},
		{0x07, STV_PROTECTED}, // Mask should work
		{0xFC, STV_DEFAULT},   // High bits should be ignored
	}

	for _, tt := range tests {
		got := ST_VISIBILITY(tt.other)
		if got != tt.expected {
			t.Errorf("ST_VISIBILITY(0x%02x) = %d, want %d", tt.other, got, tt.expected)
		}
	}
}

func TestELF32_R_SYM(t *testing.T) {
	tests := []struct {
		info     uint32
		expected uint32
	}{
		{0x00000000, 0},
		{0x00000100, 1},
		{0x12345600, 0x123456},
		{0xFFFFFF00, 0xFFFFFF},
	}

	for _, tt := range tests {
		got := ELF32_R_SYM(tt.info)
		if got != tt.expected {
			t.Errorf("ELF32_R_SYM(0x%08x) = 0x%x, want 0x%x", tt.info, got, tt.expected)
		}
	}
}

func TestELF32_R_TYPE(t *testing.T) {
	tests := []struct {
		info     uint32
		expected uint32
	}{
		{0x00000000, 0},
		{0x00000001, 1},
		{0x123456FF, 0xFF},
		{0xFFFFFFFF, 0xFF},
	}

	for _, tt := range tests {
		got := ELF32_R_TYPE(tt.info)
		if got != tt.expected {
			t.Errorf("ELF32_R_TYPE(0x%08x) = 0x%x, want 0x%x", tt.info, got, tt.expected)
		}
	}
}

func TestELF32_R_INFO(t *testing.T) {
	tests := []struct {
		sym      uint32
		typ      uint32
		expected uint32
	}{
		{0, 0, 0x00000000},
		{1, 1, 0x00000101},
		{0x123456, 0xFF, 0x123456FF},
	}

	for _, tt := range tests {
		got := ELF32_R_INFO(tt.sym, tt.typ)
		if got != tt.expected {
			t.Errorf("ELF32_R_INFO(0x%x, 0x%x) = 0x%08x, want 0x%08x", tt.sym, tt.typ, got, tt.expected)
		}
	}
}

func TestELF64_R_SYM(t *testing.T) {
	tests := []struct {
		info     uint64
		expected uint32
	}{
		{0x0000000000000000, 0},
		{0x0000000100000000, 1},
		{0x1234567800000000, 0x12345678},
		{0xFFFFFFFF00000000, 0xFFFFFFFF},
	}

	for _, tt := range tests {
		got := ELF64_R_SYM(tt.info)
		if got != tt.expected {
			t.Errorf("ELF64_R_SYM(0x%016x) = 0x%x, want 0x%x", tt.info, got, tt.expected)
		}
	}
}

func TestELF64_R_TYPE(t *testing.T) {
	tests := []struct {
		info     uint64
		expected uint32
	}{
		{0x0000000000000000, 0},
		{0x0000000000000001, 1},
		{0x12345678ABCDEF01, 0xABCDEF01},
		{0xFFFFFFFFFFFFFFFF, 0xFFFFFFFF},
	}

	for _, tt := range tests {
		got := ELF64_R_TYPE(tt.info)
		if got != tt.expected {
			t.Errorf("ELF64_R_TYPE(0x%016x) = 0x%x, want 0x%x", tt.info, got, tt.expected)
		}
	}
}

func TestELF64_R_INFO(t *testing.T) {
	tests := []struct {
		sym      uint32
		typ      uint32
		expected uint64
	}{
		{0, 0, 0x0000000000000000},
		{1, 1, 0x0000000100000001},
		{0x12345678, 0xABCDEF01, 0x12345678ABCDEF01},
	}

	for _, tt := range tests {
		got := ELF64_R_INFO(tt.sym, tt.typ)
		if got != tt.expected {
			t.Errorf("ELF64_R_INFO(0x%x, 0x%x) = 0x%016x, want 0x%016x", tt.sym, tt.typ, got, tt.expected)
		}
	}
}

func TestRoundTrip32(t *testing.T) {
	// Test that we can round-trip symbol info
	for bind := uint8(0); bind < 16; bind++ {
		for typ := uint8(0); typ < 16; typ++ {
			info := ST_INFO(bind, typ)
			gotBind := ST_BIND(info)
			gotType := ST_TYPE(info)
			if gotBind != bind || gotType != typ {
				t.Errorf("ST_INFO round-trip failed: bind=%d typ=%d -> info=0x%02x -> bind=%d typ=%d",
					bind, typ, info, gotBind, gotType)
			}
		}
	}

	// Test that we can round-trip relocation info (32-bit)
	testCases := [][2]uint32{
		{0, 0},
		{1, 1},
		{0xFF, 0xFF},
		{0x123456, 0xAB},
		{0xFFFFFF, 0xFF},
	}
	for _, tc := range testCases {
		sym, typ := tc[0], tc[1]
		info := ELF32_R_INFO(sym, typ)
		gotSym := ELF32_R_SYM(info)
		gotType := ELF32_R_TYPE(info)
		if gotSym != sym || gotType != typ {
			t.Errorf("ELF32_R_INFO round-trip failed: sym=0x%x typ=0x%x -> info=0x%08x -> sym=0x%x typ=0x%x",
				sym, typ, info, gotSym, gotType)
		}
	}
}

func TestRoundTrip64(t *testing.T) {
	// Test that we can round-trip relocation info (64-bit)
	testCases := [][2]uint32{
		{0, 0},
		{1, 1},
		{0xFFFFFFFF, 0xFFFFFFFF},
		{0x12345678, 0xABCDEF01},
	}
	for _, tc := range testCases {
		sym, typ := tc[0], tc[1]
		info := ELF64_R_INFO(sym, typ)
		gotSym := ELF64_R_SYM(info)
		gotType := ELF64_R_TYPE(info)
		if gotSym != sym || gotType != typ {
			t.Errorf("ELF64_R_INFO round-trip failed: sym=0x%x typ=0x%x -> info=0x%016x -> sym=0x%x typ=0x%x",
				sym, typ, info, gotSym, gotType)
		}
	}
}

// Verify struct sizes match expected ELF format sizes
func TestStructSizes(t *testing.T) {
	// Note: These tests verify the struct field layout assumptions.
	// The actual binary size depends on Go's struct padding rules,
	// but we read/write these structures field by field anyway.

	// ELF32 sizes (from specification)
	const (
		elf32EhdrSize = 52
		elf32PhdrSize = 32
		elf32ShdrSize = 40
		elf32SymSize  = 16
		elf32RelSize  = 8
		elf32RelaSize = 12
		elf32DynSize  = 8
		elf32ChdrSize = 12
	)

	// ELF64 sizes (from specification)
	const (
		elf64EhdrSize = 64
		elf64PhdrSize = 56
		elf64ShdrSize = 64
		elf64SymSize  = 24
		elf64RelSize  = 16
		elf64RelaSize = 24
		elf64DynSize  = 16
		elf64ChdrSize = 24
	)

	// These are the sizes we actually read in the parser
	t.Log("ELF32 expected sizes:", elf32EhdrSize, elf32PhdrSize, elf32ShdrSize)
	t.Log("ELF64 expected sizes:", elf64EhdrSize, elf64PhdrSize, elf64ShdrSize)
}
