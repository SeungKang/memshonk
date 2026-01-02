package cstlv

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"math"
)

const (
	chanStart         = 0
	chanEnd           = 4
	seqStart          = 4
	seqEnd            = 8
	typeStart         = 8
	typeEnd           = 10
	lenStart          = 10
	lenEnd            = 14
	minimumMessageLen = lenEnd
)

// CSTLV represents a single Channel Sequence Type Length
// message read from bits.
//
// The only required field is Len which is used to determine
// the size of the Val (payload) field. Integrators can opt
// to use or skip fields depending on their use case.
//
// Example:
//  00000001 00000001 0001 00000004 abcd
//  -------- -------- ---- -------- ----
//  |        |        |    |        |
//  chan     |        |    |        |
//           seq      |    |        |
//                    type |        |
//                         len(val) |
//                                  val (value / payload)
type CSTLV struct {
	// Chan is the channel number.
	Chan uint32

	// Seq is the message's sequence number.
	Seq uint32

	// Type is the message's type.
	Type uint16

	// Len is the length (size) of the Val field.
	Len uint32 // TODO: Is 32bits of data even possible?

	// Val (value) is the message's payload.
	Val []byte
}

// FromRawBytes parses a []byte into a CSTLV.
func FromRawBytes(b []byte) (*CSTLV, error) {
	m, err := fromHeader(b)
	if err != nil {
		return nil, fmt.Errorf("failed to parse header - %w", err)
	}

	actualPayloadLen := len(b[lenEnd:])
	if int(m.Len) != actualPayloadLen {
		return nil, fmt.Errorf("payload len is set to %d but the actual len is %d bytes",
			m.Len, actualPayloadLen)
	}

	if m.Len != 0 {
		m.Val = b[lenEnd:]
	}

	return m, nil
}

// fromHeader parses only the CSTLV header portion from a []byte.
// The caller must set the CSTLV.Val.
func fromHeader(b []byte) (*CSTLV, error) {
	entireMsgLen := len(b)
	if entireMsgLen < minimumMessageLen {
		return nil, fmt.Errorf("message is less than minimum length of %d bytes - it is %d bytes",
			minimumMessageLen, entireMsgLen)
	}

	return &CSTLV{
		Chan: binary.BigEndian.Uint32(b[chanStart:chanEnd]),
		Seq:  binary.BigEndian.Uint32(b[seqStart:seqEnd]),
		Type: binary.BigEndian.Uint16(b[typeStart:typeEnd]),
		Len:  binary.BigEndian.Uint32(b[lenStart:lenEnd]),
	}, nil
}

// MinimalBytes creates a CSTLV without a val (payload) from the
// specified arguments and serializes it into a []byte.
func MinimalBytes(channel uint32, seq uint32, messageType uint16) []byte {
	m := CSTLV{
		Chan: channel,
		Seq:  seq,
		Type: messageType,
	}

	return m.bytes()
}

func (o *CSTLV) ManualBytes() ([]byte, error) {
	if int(o.Len) != len(o.Val) {
		return nil, fmt.Errorf("message len should be %d - it is %d",
			len(o.Val), o.Len)
	}

	return o.bytes(), nil
}

func (o *CSTLV) AutoBytes() []byte {
	o.Len = uint32(len(o.Val))
	return o.bytes()
}

func (o *CSTLV) bytes() []byte {
	raw := make([]byte, minimumMessageLen+int(o.Len))
	binary.BigEndian.PutUint32(raw, o.Chan)
	binary.BigEndian.PutUint32(raw[seqStart:seqEnd], o.Seq)
	binary.BigEndian.PutUint16(raw[typeStart:typeEnd], o.Type)
	binary.BigEndian.PutUint32(raw[lenStart:lenEnd], o.Len)
	if o.Len > 0 {
		copy(raw[lenEnd:], o.Val)
	}
	return raw
}

// ScannerSplitFn is a simple implementation of a bufio.SplitFunc
// for CSTLV messages. Note that the bufio.Scanner has a relatively
// small buffer by default, which makes it less-than-ideal for use
// with CSTLV messages that contain large payloads.
func ScannerSplitFn(msg []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(msg) == 0 {
		return 0, nil, nil
	}

	msgLen := len(msg)
	if msgLen < minimumMessageLen {
		return 0, nil, nil
	}

	if msgLen > math.MaxUint32 {
		return 0, nil, fmt.Errorf("current token is longer than max uint32")
	}

	msgLenUint32 := uint32(msgLen)
	payloadLen := binary.BigEndian.Uint32(msg[lenStart:lenEnd])
	if payloadLen > msgLenUint32 {
		return 0, nil, nil
	}

	totalExpectedLen := minimumMessageLen + payloadLen
	if totalExpectedLen > msgLenUint32 {
		return 0, nil, nil
	}

	return int(totalExpectedLen), msg[0:totalExpectedLen], nil
}

// NewBufferedParserFrom wraps reader using bufio.NewReader and then
// instantiates a Parser using the new reader.
func NewBufferedParserFrom(reader io.Reader) *Parser {
	return NewParserFrom(bufio.NewReader(reader))
}

// NewParserFrom creates a new Parser from reader.
func NewParserFrom(reader io.Reader) *Parser {
	return &Parser{
		r: reader,
	}
}

// Parser parses CSTLV messages from an io.Reader.
//
// It presents a similar API to that of bufio.Scanner.
type Parser struct {
	r   io.Reader
	m   *CSTLV
	err error
}

// Err returns the last error encountered by the Parser. If no
// error has occurred, then nil is returned.
func (o *Parser) Err() error {
	return o.err
}

// Message is the last-parsed CSTLV.
func (o *Parser) Message() *CSTLV {
	return o.m
}

// Next reads from the underling io.Reader until a CSTLV is found.
// It returns true if a CSTLV is found. False is returned if an
// error is encountered while reading from the underling io.Reader,
// or if the CSTLV cannot be parsed.
//
// This method is typically called in a loop, similar to the
// bufio.Scanner.Scan method. Callers should check the Err
// method if Next returns false.
func (o *Parser) Next() bool {
	if o.err != nil {
		return false
	}

	foundOne := false

	foundOne, o.err = o.next()
	if o.err != nil {
		o.m = nil
	}

	return foundOne
}

func (o *Parser) next() (bool, error) {
	o.m = nil

	headerBytes := make([]byte, minimumMessageLen)
	_, err := io.ReadFull(o.r, headerBytes)
	if err != nil {
		return false, err
	}

	o.m, err = fromHeader(headerBytes)
	if err != nil {
		return false, err
	}

	if o.m.Len == 0 {
		return true, nil
	}

	o.m.Val = make([]byte, o.m.Len)

	_, err = io.ReadFull(o.r, o.m.Val)
	if err != nil {
		return false, err
	}

	return true, nil
}
