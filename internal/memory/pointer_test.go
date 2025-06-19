package memory

import (
	"testing"
)

func TestPointer_String(t *testing.T) {
	ptr := Pointer{
		Name:      "",
		Addrs:     []uintptr{0x152AED8, 0x4C, 0x284, 0xC, 0x8, 0x18, 0x404, 0x6B4},
		OptModule: "",
	}

	need := "0x152aed8,0x4c,0x284,0xc,0x8,0x18,0x404,0x6b4"

	got := ptr.String()

	if got != need {
		t.Fatalf("got: %q, need: %q", got, need)
	}
}

func TestCreatePointerFromString(t *testing.T) {
	need := "buh.dll:0x152aed8,0x4c,0x284,0xc,0x8,0x18,0x404,0x6b4"

	got := Pointer{
		Name:      "",
		Addrs:     []uintptr{0x152AED8, 0x4C, 0x284, 0xC, 0x8, 0x18, 0x404, 0x6B4},
		OptModule: "buh.dll",
	}

	gotStr := got.String()

	if gotStr != need {
		t.Fatalf("got: %q, need %q", gotStr, need)
	}
}
