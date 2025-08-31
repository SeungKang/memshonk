package memory

import (
	"bytes"
	"context"
	"errors"
	"testing"
)

func TestFindAll_Reader(t *testing.T) {
	parsedPattern, err := ParsePattern("8B 45 ?? C7 00 00 00 ?? ?? 5D C2 08 00 8B 4D")
	if err != nil {
		t.Fatalf("failed to parse pattern - %v", err)
	}

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

	ctx := context.Background()

	matches, err := FindAllReader(ctx, parsedPattern, reader)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("len(matches) = %d, want 1", len(matches))
	}

	if matches[0].Addr.FirstAddr() != 69 {
		t.Fatalf("matches[0].Addrs[0] = %d, want 69", matches[0].Addr.FirstAddr())
	}
}

func TestFindAll_Reader2(t *testing.T) {
	parsedPattern, err := ParsePattern("?? 45 ??")
	if err != nil {
		t.Fatalf("failed to parse pattern - %v", err)
	}

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

	ctx := context.Background()

	matches, err := FindAllReader(ctx, parsedPattern, reader)
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	if len(matches) != 1 {
		t.Fatalf("len(matches) = %d, want 1", len(matches))
	}

	if matches[0].Addr.FirstAddr() != 69 {
		t.Fatalf("matches[0].Addrs[0] = %d, want 69",
			matches[0].Addr.FirstAddr())
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

func (o *fakeReader) ReadFromAddr(_ context.Context, ptr Pointer, size uint64) ([]byte, uintptr, error) {
	addr := ptr.FirstAddr()

	offset := uint64(addr)
	if offset > uint64(len(o.data)) {
		return nil, 0, errors.New("invalid offset")
	}

	upto := offset + size
	if upto > uint64(len(o.data)) {
		return nil, 0, errors.New("size out of range")
	}

	return o.data[offset:upto], addr, nil
}
