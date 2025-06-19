package memory

import (
	"context"
	"testing"
)

func TestBufferedReader(t *testing.T) {
	fakeProcess := &fakeReader{}

	fakeProcess.data = append(fakeProcess.data,
		0x8B, 0x45, 0x69, 0xC7, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x5D, 0xC2, 0x08, 0x00, 0x8B, 0x4D)

	reader, err := NewBufferedReader(fakeProcess, AbsoluteAddrPointer(0), fakeProcess.size())
	if err != nil {
		t.Fatalf("want no error, got %v", err)
	}

	reader.SetAdvanceBy(1)

	i := 0

	for reader.Next(context.Background(), 5) {
		addr := reader.Addr().FirstAddr()
		if i != int(addr) {
			t.Fatalf("got: %d, want: %d",
				addr, i)
		}

		i++
	}

	if reader.Err() != nil {
		t.Fatal(err)
	}
}
