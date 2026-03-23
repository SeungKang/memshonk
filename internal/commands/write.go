package commands

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"strconv"
	"unicode/utf16"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/progctl"
)

const (
	WriteCommandName = "writem"
)

func NewWriteCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &WriteCommand{
		session: config.Session,
	}

	root := fx.NewCommand(WriteCommandName, "write data to process memory", cmd.run)

	root.FlagSet.StringFlag(&cmd.datatype, rawDataType, fx.ArgConfig{
		Name:        "datatype",
		Description: "Specify the `datatype` of the value to write (refer to \"help datatypes\")",
	})

	root.FlagSet.StringFlag(&cmd.datatype, rawEncoding, fx.ArgConfig{
		Name:        "input-format",
		Description: "Specify the input `format` of the value to write (refer to \"help formats\")",
	})

	root.FlagSet.Uint64Flag(&cmd.numInstances, 1, fx.ArgConfig{
		Name:        "times",
		Description: "The number of times to write the value",
	})

	root.FlagSet.StringFlag(&cmd.dataStr, "", fx.ArgConfig{
		Name:        "value",
		Description: "the `data` to write",
		Required:    true,
	})

	root.FlagSet.StringFlag(&cmd.addrStr, "", fx.ArgConfig{
		Name:        "addr",
		Description: "`address` to write to",
		Required:    true,
	})

	return root
}

type WriteCommand struct {
	session      apicompat.Session
	datatype     string
	inputFormat  string
	dataStr      string
	addrStr      string
	numInstances uint64
}

func (o *WriteCommand) run(ctx context.Context) (fx.CommandResult, error) {
	writer := newProcessWriter(ctx, o.addrStr, o.session.SharedState().Progctl)

	var err error

	switch o.datatype {
	case rawDataType:
		err = o.doRaw(ctx, writer)
	case stringDataType, stringleDataType, stringbeDataType, utf8DataType, utf8leDataType, utf8beDataType:
		err = o.doUtf8String(ctx, writer)
	case utf16DataType, utf16leDataType, utf16beDataType, wstringDataType, wstringleDataType, wstringbeDataType:
		err = o.doUtf16String(ctx, writer)
	case float32DataType, float32leDataType, float32beDataType:
		err = o.doFloat32(ctx, writer)
	case float64DataType, float64leDataType, float64beDataType:
		err = o.doFloat64(ctx, writer)
	case uint16DataType, uint16leDataType, uint16beDataType:
		err = o.doUnit16(ctx, writer)
	case uint32DataType, uint32leDataType, uint32beDataType:
		err = o.doUint32(ctx, writer)
	case uint64DataType, uint64leDataType, uint64beDataType:
		err = o.doUint64(ctx, writer)
	default:
		return nil, fmt.Errorf("unknown datatype: %q", o.datatype)
	}

	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (o *WriteCommand) doRaw(ctx context.Context, writer *processWriter) error {
	data, err := decodeDataStr(o.inputFormat, o.dataStr)
	if err != nil {
		return err
	}

	for i := uint64(0); i < o.numInstances; i++ {
		_, err := writer.Write(data)
		if err != nil {
			return err
		}

		writer.OffsetBy(int64(len(data)))
	}

	return nil
}

func (o *WriteCommand) doUtf8String(ctx context.Context, writer *processWriter) error {
	data, err := decodeDataStr(o.inputFormat, o.dataStr)
	if err != nil {
		return err
	}

	var endian binary.ByteOrder = binary.LittleEndian

	switch o.datatype {
	case stringbeDataType, utf8beDataType:
		endian = binary.BigEndian
	}

	return o.doWrite(data, endian, writer)
}

func (o *WriteCommand) doUtf16String(ctx context.Context, writer *processWriter) error {
	data, err := decodeDataStr(o.inputFormat, o.dataStr)
	if err != nil {
		return err
	}

	runes := []rune(string(data))

	u16ints := utf16.Encode(runes)

	var endian binary.ByteOrder = binary.LittleEndian

	switch o.datatype {
	case utf16beDataType, wstringbeDataType:
		endian = binary.BigEndian
	}

	return o.doWrite(u16ints, endian, writer)
}

func (o *WriteCommand) doUnit16(ctx context.Context, writer *processWriter) error {
	data, err := decodeDataStr(o.inputFormat, o.dataStr)
	if err != nil {
		return err
	}

	str := string(data)

	v, err := stringWithBasePrefixToUint(str, 16)
	if err != nil {
		return fmt.Errorf("failed to parse uint16 string %q - %w",
			str, err)
	}

	var endian binary.ByteOrder = binary.LittleEndian

	if o.datatype == uint16beDataType {
		endian = binary.BigEndian
	}

	return o.doWrite(uint16(v), endian, writer)
}

func (o *WriteCommand) doUint32(ctx context.Context, writer *processWriter) error {
	data, err := decodeDataStr(o.inputFormat, o.dataStr)
	if err != nil {
		return err
	}

	str := string(data)

	v, err := stringWithBasePrefixToUint(str, 32)
	if err != nil {
		return fmt.Errorf("failed to parse uint32 string %q - %w",
			str, err)
	}

	var endian binary.ByteOrder = binary.LittleEndian

	if o.datatype == uint32beDataType {
		endian = binary.BigEndian
	}

	return o.doWrite(uint32(v), endian, writer)
}

func (o *WriteCommand) doUint64(ctx context.Context, writer *processWriter) error {
	data, err := decodeDataStr(o.inputFormat, o.dataStr)
	if err != nil {
		return err
	}

	str := string(data)

	v, err := stringWithBasePrefixToUint(str, 64)
	if err != nil {
		return fmt.Errorf("failed to parse uint64 string %q - %w",
			str, err)
	}

	var endian binary.ByteOrder = binary.LittleEndian

	if o.datatype == uint64beDataType {
		endian = binary.BigEndian
	}

	return o.doWrite(v, endian, writer)
}

func (o *WriteCommand) doFloat32(ctx context.Context, writer *processWriter) error {
	data, err := decodeDataStr(o.inputFormat, o.dataStr)
	if err != nil {
		return err
	}

	str := string(data)

	v, err := strconv.ParseFloat(str, 32)
	if err != nil {
		return fmt.Errorf("failed to parse float32 %q - %w",
			str, err)
	}

	var endian binary.ByteOrder = binary.LittleEndian

	if o.datatype == float32beDataType {
		endian = binary.BigEndian
	}

	return o.doWrite(float32(v), endian, writer)
}

func (o *WriteCommand) doFloat64(ctx context.Context, writer *processWriter) error {
	data, err := decodeDataStr(o.inputFormat, o.dataStr)
	if err != nil {
		return err
	}

	str := string(data)

	v, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return fmt.Errorf("failed to parse float64 %q - %w",
			str, err)
	}

	var endian binary.ByteOrder = binary.LittleEndian

	if o.datatype == float64beDataType {
		endian = binary.BigEndian
	}

	return o.doWrite(v, endian, writer)
}

func (o *WriteCommand) doWrite(v interface{}, endian binary.ByteOrder, writer *processWriter) error {
	for i := uint64(0); i < o.numInstances; i++ {
		err := binary.Write(writer, endian, v)
		if err != nil {
			return fmt.Errorf("write operation failed - %w", err)
		}

		writer.OffsetBy(int64(writer.LastWriteSize()))
	}

	return nil
}

func newProcessWriter(ctx context.Context, addr string, ctl *progctl.Ctl) *processWriter {
	return &processWriter{
		ctx:     ctx,
		addr:    addr,
		process: ctl,
	}
}

type processWriter struct {
	ctx     context.Context
	addr    string
	process *progctl.Ctl

	useLastWriteAddr bool
	lastWriteAddr    uintptr
	lastWriteSize    uint64
}

func (o *processWriter) Write(b []byte) (int, error) {
	var actualAddr uintptr
	var err error

	if o.useLastWriteAddr {
		actualAddr, err = o.process.WriteToAddr(o.ctx, memory.AbsoluteAddrPointer(o.lastWriteAddr), b)
	} else {
		actualAddr, err = o.process.WriteToLookup(o.ctx, o.addr, b)
	}

	if err != nil {
		return 0, err
	}

	length := len(b)

	o.lastWriteAddr = actualAddr
	o.lastWriteSize = uint64(length)

	return length, nil
}

func (o *processWriter) OffsetBy(i int64) {
	if !o.useLastWriteAddr {
		o.useLastWriteAddr = true
	}

	o.lastWriteAddr += uintptr(i)
}

func (o *processWriter) LastWriteAddr() uintptr {
	return o.lastWriteAddr
}

func (o *processWriter) LastWriteSize() uint64 {
	return o.lastWriteSize
}

func decodeDataStr(inputFormat string, dataStr string) ([]byte, error) {
	switch inputFormat {
	case rawEncoding, "":
		return []byte(dataStr), nil
	case binaryEncoding:
		data, err := binaryStringToBytes(dataStr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse as binary - %w", err)
		}

		return data, nil
	case hexdumpEncoding:
		return nil, fmt.Errorf("hexdump encoding is currently unsupported :(")
	case hexEncoding:
		data, err := hex.DecodeString(dataStr)
		if err != nil {
			return nil, fmt.Errorf("failed to hex-decode string - %w", err)
		}

		return data, nil
	case base64Encoding, b64Encoding:
		data, err := base64.StdEncoding.DecodeString(dataStr)
		if err != nil {
			return nil, fmt.Errorf("failed to base64-decode string - %w", err)
		}

		return data, nil
	default:
		return nil, fmt.Errorf("unsupported output format for raw: %q",
			inputFormat)
	}
}

// binaryStringToBytes was copied from the Google AI thing.
func binaryStringToBytes(s string) ([]byte, error) {
	if len(s)%8 != 0 {
		return nil, fmt.Errorf("binary string length must be a multiple of 8")
	}

	// Calculate the number of bytes needed
	byteLen := len(s) / 8
	bytes := make([]byte, byteLen)

	// Iterate through the string, 8 characters (bits) at a time
	for i := 0; i < byteLen; i++ {
		start := i * 8
		end := start + 8
		byteString := s[start:end]

		// Parse the 8-bit substring as an unsigned integer with base 2
		val, err := strconv.ParseUint(byteString, 2, 8)
		if err != nil {
			return nil, fmt.Errorf("invalid binary sequence: %w", err)
		}

		// Cast the resulting uint64 to a byte and assign it
		bytes[i] = byte(val)
	}

	return bytes, nil
}
