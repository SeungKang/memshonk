package memory

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestFindAll_Reader(t *testing.T) {
	pattern := "8B 45 ?? C7 00 00 00 ?? ?? 5D C2 08 00 8B 4D"

	fakeProcess := &fakeReader{}

	fakeProcess.data = append(fakeProcess.data, bytes.Repeat([]byte{0x69}, 69)...)

	fakeProcess.data = append(fakeProcess.data,
		0x8B, 0x45, 0x69, 0xC7, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x5D, 0xC2, 0x08, 0x00, 0x8B, 0x4D)

	fakeProcess.data = append(fakeProcess.data, bytes.Repeat([]byte{0x69}, 1024)...)

	reader, err := NewBufferedReader(fakeProcess, AbsoluteAddrPointer(0), fakeProcess.size())
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	matches, err := FindAllReader(pattern, reader)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("len(matches) = %d, want 1", len(matches))
	}

	if matches[0].FirstAddr() != 69 {
		t.Fatalf("matches[0].Addrs[0] = %d, want 69", matches[0].FirstAddr())
	}
}

func TestFindAll_Reader2(t *testing.T) {
	pattern := "?? 45 ??"

	fakeProcess := &fakeReader{}

	fakeProcess.data = append(fakeProcess.data, bytes.Repeat([]byte{0x69}, 69)...)

	fakeProcess.data = append(fakeProcess.data,
		0x8B, 0x45, 0x69, 0xC7, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x5D, 0xC2, 0x08, 0x00, 0x8B, 0x4D)

	fakeProcess.data = append(fakeProcess.data, bytes.Repeat([]byte{0x69}, 1024)...)

	reader, err := NewBufferedReader(fakeProcess, AbsoluteAddrPointer(0), fakeProcess.size())
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	matches, err := FindAllReader(pattern, reader)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("len(matches) = %d, want 1", len(matches))
	}

	if matches[0].FirstAddr() != 69 {
		t.Fatalf("matches[0].Addrs[0] = %d, want 69", matches[0].FirstAddr())
	}
}

type fakeReader struct {
	data []byte
}

func (o *fakeReader) size() uint64 {
	return uint64(len(o.data))
}

func (o *fakeReader) ResolvePointer(_ context.Context, ptr Pointer) (uintptr, error) {
	return 0, nil
}

func (o *fakeReader) ReadFromAddr(_ context.Context, ptr Pointer, size uint64) ([]byte, error) {
	addr := ptr.FirstAddr()

	offset := uint64(addr)
	if offset > uint64(len(o.data)) {
		return nil, errors.New("invalid offset")
	}

	upto := offset + size
	if upto > uint64(len(o.data)) {
		return nil, errors.New("size out of range")
	}

	return o.data[offset:upto], nil
}
