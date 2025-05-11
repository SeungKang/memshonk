package memory

import (
	"context"
	"fmt"
)

type ReadFromAddr interface {
	ReadFromAddr(ctx context.Context, addr Pointer, size uint64) ([]byte, error)
}

func NewBufferedReader(readFrom ReadFromAddr, start Pointer) *BufferedReader {
	return &BufferedReader{
		reader:  readFrom,
		start:   start,
		hasMore: true,
	}
}

type BufferedReader struct {
	reader     ReadFromAddr
	start      Pointer
	last       ReadChunk
	buf        []byte
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

// Chunk returns the last-read ReadChunk.
func (o *BufferedReader) Chunk() ReadChunk {
	return o.last
}

func (o *BufferedReader) SetAdvanceBy(by uint64) {
	o.advanceBy = by
}

// Next reads the next ReadChunk.
func (o *BufferedReader) Next(ctx context.Context, need uint64) bool {
	if o.err != nil || !o.hasMore {
		return false
	}

	chunk, hasMore, err := o.next(ctx, need)
	if err != nil {
		o.err = fmt.Errorf("next failed - %w", err)

		return false
	}

	o.last = chunk

	o.hasMore = hasMore

	return true
}

func (o *BufferedReader) next(ctx context.Context, need uint64) (ReadChunk, bool, error) {
	err := o.read(ctx, need)
	if err != nil {
		return ReadChunk{}, false, err
	}

	bufLen := uint64(len(o.buf))

	if bufLen == 0 {
		return ReadChunk{}, false, nil
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

	return ReadChunk{
		Data: data,
		Addr: o.start.Advance(advanceBy),
	}, len(o.buf) > 0, nil
}

func (o *BufferedReader) read(ctx context.Context, need uint64) error {
	if o.readerDone {
		return nil
	}

	const minReadSizeBytes uint64 = 1024

	readSizeBytes := need
	if readSizeBytes < minReadSizeBytes {
		readSizeBytes += minReadSizeBytes
	}

	if uint64(len(o.buf)) < need {
		offset := o.readerOff
		o.readerOff += readSizeBytes

		ptr := o.start.Advance(offset)

		b, err := o.reader.ReadFromAddr(ctx, ptr, readSizeBytes)
		switch {
		case err == nil:
			o.buf = append(o.buf, b...)
		default:
			// TODO: Add support for checking if read
			// is within memory-mapped object's address
			// range.
			o.readerDone = true

			return fmt.Errorf("failed to read %d bytes from %s - %w",
				readSizeBytes, ptr.String(), err)
		}
	}

	return nil
}
