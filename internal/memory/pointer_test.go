package memory

import (
	"reflect"
	"testing"
)

func TestString(t *testing.T) {
	ptr := Pointer{
		Name:      "",
		Addrs:     []uintptr{0x152AED8, 0x4C, 0x284, 0xC, 0x8, 0x18, 0x404, 0x6B4},
		OptModule: "",
	}

	want := "0x152aed8,0x4c,0x284,0xc,0x8,0x18,0x404,0x6b4"

	if ptr.String() != want {
		t.Fatalf("ptr.String() = %q, want %q", ptr.String(), want)
	}
}

func TestCreatePointerFromString(t *testing.T) {
	ptrStr := "buh.dll:0x152AED8,0x4C,0x284,0xC,0x8,0x18,0x404,0x6B4"

	ptr, err := CreatePointerFromString(ptrStr)
	if err != nil {
		t.Fatal(err)
	}

	want := Pointer{
		Name:      "",
		Addrs:     []uintptr{0x152AED8, 0x4C, 0x284, 0xC, 0x8, 0x18, 0x404, 0x6B4},
		OptModule: "buh.dll",
	}

	if !reflect.DeepEqual(ptr, want) {
		t.Fatalf("CreatePointerFromString(%q) = %+v, want %+v", ptrStr, ptr, want)
	}
}
