package memory

import (
	"context"
	"fmt"
	"log"
)

type ReadFromAddr interface {
	ResolvePointer(ctx context.Context, ptr Pointer) (uintptr, MappedObject, error)

	ReadFromAddr(ctx context.Context, addr Pointer, size uint64) ([]byte, error)
}

// TODO: Constrain BufferedReader to a range of addresses rather
// than an object. Implement constructor-like functions that
// either constrain the range based on an arbitrary range or
// base and end addrs of a mapped object.
func NewBufferedReader(readFrom ReadFromAddr, start Pointer, size uint64) (*BufferedReader, error) {
	startAddr, _, err := readFrom.ResolvePointer(context.Background(), start)
	if err != nil {
		return nil, err
	}

	return &BufferedReader{
		reader:    readFrom,
		start:     start,
		readPtr:   Pointer{Addrs: []uintptr{startAddr}},
		remaining: size,
		hasMore:   true,
	}, nil
}

// TODO: Add Addr method to return the Pointer for the last read chunk
// (i.e., move Pointer out of ReadChunk struct).
type BufferedReader struct {
	reader     ReadFromAddr
	start      Pointer
	readPtr    Pointer
	remaining  uint64
	buf        []byte
	lastData   []byte
	lastOffset uint64
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

	data, hasMore, err := o.next(ctx, need)
	if err != nil {
		o.err = fmt.Errorf("next failed - %w", err)

		return false
	}

	o.lastData = data
	o.hasMore = hasMore

	return true
}

func (o *BufferedReader) next(ctx context.Context, need uint64) ([]byte, bool, error) {
	err := o.read(ctx, need)
	if err != nil {
		return nil, false, err
	}

	bufLen := uint64(len(o.buf))

	if bufLen == 0 {
		return nil, false, nil
	}

	var dataSize uint64
	if bufLen < need {
		dataSize = bufLen
	} else {
		dataSize = need
	}

	data := o.buf[0:dataSize]

	var advanceBy uint64
	if o.advanceBy == 0 {
		advanceBy = dataSize
	} else if o.advanceBy > bufLen {
		advanceBy = bufLen
	} else {
		advanceBy = o.advanceBy
	}

	o.buf = o.buf[advanceBy:]

	bufOffset := o.bufOffset
	o.bufOffset += advanceBy

	o.lastOffset = bufOffset

	return data, len(o.buf) > 0, nil
}

func (o *BufferedReader) read(ctx context.Context, need uint64) error {
	if o.readerDone {
		return nil
	}

	if uint64(len(o.buf)) > need {
		return nil
	}

	const minReadSizeBytes uint64 = 1024

	readSizeBytes := need
	if readSizeBytes < minReadSizeBytes {
		readSizeBytes += minReadSizeBytes
	}

	if readSizeBytes > o.remaining {
		readSizeBytes = o.remaining

		o.readerDone = true
	}

	b, err := o.reader.ReadFromAddr(ctx, o.readPtr, readSizeBytes)
	switch {
	case err == nil:
		o.remaining -= readSizeBytes

		o.buf = append(o.buf, b...)

		offset := o.readerOff
		o.readerOff += readSizeBytes

		// TODO: Replace with MutAdvance.
		o.readPtr.Addrs[0] += uintptr(offset)

		log.Printf("TODO: offset: %d | buf len: %d | read size bytes: %d",
			offset, len(o.buf), readSizeBytes)

		return nil
	default:
		o.readerDone = true

		return fmt.Errorf("failed to read %d bytes from 0x%x - %w",
			readSizeBytes, o.readPtr.Addrs[0], err)
	}
}
