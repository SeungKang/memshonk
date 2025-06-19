package memory

import (
	"bytes"
	"context"
	"fmt"
)

// TODO: Implement a Reader object for a process that knows its
// bounds based on mapped objects.

type ReadFromAddr interface {
	ResolvePointer(ctx context.Context, ptr Pointer) (uintptr, error)

	ReadFromAddr(ctx context.Context, addr Pointer, size uint64) ([]byte, error)
}

// TODO: Constrain BufferedReader to a range of addresses rather
// than an object. Implement constructor-like functions that
// either constrain the range based on an arbitrary range or
// base and end addrs of a mapped object.
func NewBufferedReader(readFrom ReadFromAddr, start Pointer, size uint64) (*BufferedReader, error) {
	startAddr, err := readFrom.ResolvePointer(context.Background(), start)
	if err != nil {
		return nil, err
	}

	return &BufferedReader{
		reader:     readFrom,
		start:      start,
		readPtr:    AbsoluteAddrPointer(startAddr),
		readRemain: size,
		hasMore:    true,
	}, nil
}

// TODO: Add Addr method to return the Pointer for the last read chunk
// (i.e., move Pointer out of ReadChunk struct).
type BufferedReader struct {
	reader     ReadFromAddr
	start      Pointer
	readPtr    Pointer
	readRemain uint64
	buf        bytes.Buffer
	lastOffset uint64
	lastData   []byte
	bufOffset  uint64
	readerDone bool
	readerOff  uint64
	advanceBy  uint64
	hasMore    bool
	err        error
}

type ReadChunk struct {
	Data []byte
	Addr Pointer
}

// Err returns the last error or nil if no error has occurred.
func (o *BufferedReader) Err() error {
	if o.err != nil {
		return o.err
	}

	return nil
}

func (o *BufferedReader) Bytes() []byte {
	return o.lastData
}

func (o *BufferedReader) Addr() Pointer {
	return o.start.Advance(o.lastOffset)
}

func (o *BufferedReader) SetAdvanceBy(by uint64) {
	o.advanceBy = by
}

// Next reads another "need's" worth of []byte from the underlying reader.
func (o *BufferedReader) Next(ctx context.Context, need uint64) bool {
	if o.err != nil || !o.hasMore {
		return false
	}

	data, err := o.next(ctx, need)
	if err != nil {
		o.err = fmt.Errorf("next failed - %w", err)

		return false
	}

	o.lastData = data

	o.hasMore = !(o.readerDone && o.readerOff == o.bufOffset)

	return true
}

func (o *BufferedReader) next(ctx context.Context, need uint64) ([]byte, error) {
	var advanceBy uint64
	if o.advanceBy == 0 {
		advanceBy = need
	} else {
		advanceBy = o.advanceBy
	}

	if o.bufOffset > 0 {
		// Discard the bytes that we want to advance by.
		o.buf.Next(int(advanceBy))
	}

	err := o.read(ctx, need)
	if err != nil {
		return nil, err
	}

	bufLen := uint64(o.buf.Len())

	if bufLen == 0 {
		return nil, nil
	}

	var dataSize uint64
	if need > bufLen {
		dataSize = bufLen
	} else {
		dataSize = need
	}

	data := o.buf.Bytes()[0:dataSize]

	o.lastOffset = o.bufOffset
	o.bufOffset += advanceBy

	return data, nil
}

func (o *BufferedReader) read(ctx context.Context, need uint64) error {
	if o.readerDone {
		return nil
	}

	if uint64(o.buf.Len()) > need {
		return nil
	}

	const minReadSizeBytes uint64 = 1024

	readSizeBytes := need
	if readSizeBytes < minReadSizeBytes {
		readSizeBytes += minReadSizeBytes
	}

	if readSizeBytes > o.readRemain {
		readSizeBytes = o.readRemain

		o.readerDone = true
	}

	b, err := o.reader.ReadFromAddr(ctx, o.readPtr, readSizeBytes)
	switch {
	case err == nil:
		o.readRemain -= readSizeBytes

		_, err := o.buf.Write(b)
		if err != nil {
			return fmt.Errorf("failed to write to buf - %w", err)
		}

		o.readerOff += readSizeBytes

		// TODO: Replace with MutAdvance.
		o.readPtr.Addrs[0] += uintptr(readSizeBytes)

		return nil
	default:
		o.readerDone = true

		return fmt.Errorf("failed to read %d bytes from 0x%x - %w",
			readSizeBytes, o.readPtr.Addrs[0], err)
	}
}
