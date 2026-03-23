package commands

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"unicode/utf16"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/hexdump"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/progctl"
)

const (
	ReadCommandName = "readm"
)

func NewReadCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &ReadCommand{
		session: config.Session,
	}

	root := fx.NewCommand(ReadCommandName, "read data from process memory", cmd.run)

	root.FlagSet.StringFlag(&cmd.dataType, rawDataType, fx.ArgConfig{
		Name:        "datatype",
		Description: "The `datatype` to read (refer to \"help datatypes\")",
	})

	root.FlagSet.Uint64Flag(&cmd.sizeBytes, 0, fx.ArgConfig{
		Name:        "size",
		Description: "number of bytes to read (only applies to data types that do not have a fixed size)",
	})

	root.FlagSet.Uint64Flag(&cmd.numInstances, 1, fx.ArgConfig{
		Name:        "times",
		Description: "The number of times to read the data",
	})

	root.FlagSet.StringFlag(&cmd.outputFormat, "", fx.ArgConfig{
		Name:        "output-format",
		Description: "The output `format` of the data (refer to \"help formats\")",
	})

	root.FlagSet.StringFlag(&cmd.addrStr, "", fx.ArgConfig{
		Name:        "addr",
		Description: "`address` to read from",
		Required:    true,
	})

	return root
}

type ReadCommand struct {
	session      apicompat.Session
	dataType     string
	outputFormat string
	sizeBytes    uint64
	numInstances uint64
	addrStr      string
}

func (o *ReadCommand) run(ctx context.Context) (fx.CommandResult, error) {
	info, err := o.session.SharedState().Progctl.ExeInfo(ctx)
	if err != nil {
		return nil, err
	}

	procReader := newProcessReader(ctx, o.addrStr, o.session.SharedState().Progctl)

	var sb strings.Builder

	switch o.dataType {
	case rawDataType:
		err = o.doRaw(ctx, procReader, info, &sb)
	case stringDataType, stringleDataType, stringbeDataType, utf8DataType, utf8leDataType, utf8beDataType:
		err = o.doUtf8String(ctx, procReader, info, &sb)
	case utf16DataType, utf16leDataType, utf16beDataType, wstringDataType, wstringleDataType, wstringbeDataType:
		err = o.doUtf16String(ctx, procReader, info, &sb)
	case float32DataType, float32leDataType, float32beDataType:
		err = o.doFloat32(ctx, procReader, info, &sb)
	case float64DataType, float64leDataType, float64beDataType:
		err = o.doFloat64(ctx, procReader, info, &sb)
	case uint16DataType, uint16leDataType, uint16beDataType:
		err = o.doUnit16(ctx, procReader, info, &sb)
	case uint32DataType, uint32leDataType, uint32beDataType:
		err = o.doUint32(ctx, procReader, info, &sb)
	case uint64DataType, uint64leDataType, uint64beDataType:
		err = o.doUint64(ctx, procReader, info, &sb)
	default:
		return nil, fmt.Errorf("unknown datatype: %q", o.dataType)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to read %q - %w",
			o.dataType, err)
	}

	return fx.NewHumanCommandResult(sb.String()), nil
}

func (o *ReadCommand) doRaw(ctx context.Context, procReader *processReader, info progctl.ExeInfo, sb *strings.Builder) error {
	if o.sizeBytes == 0 {
		return readNumBytesRequiredErr(o.dataType)
	}

	for i := uint64(0); i < o.numInstances; i++ {
		v := make([]byte, o.sizeBytes)

		_, err := procReader.Read(v)
		if err != nil {
			return err
		}

		procReader.OffsetBy(int64(o.sizeBytes))

		delim := byte(' ')

		switch o.outputFormat {
		case rawEncoding:
			sb.Write(v)
		case hexdumpEncoding, "":
			delim = '\n'

			err = hexdump.Dump(ctx, hexdump.Config{
				Src:            bytes.NewReader(v),
				Dst:            sb,
				Colors:         hexdump.NewColors(),
				OptStartOffset: uint64(procReader.LastReadAddr()),
				OptOffsetBits:  info.Bits,
			})
			if err != nil {
				return fmt.Errorf("failed to hexdump data - %w", err)
			}
		case binaryEncoding:
			for i := range v {
				sb.WriteString(fmt.Sprintf(fmt.Sprintf("%08b", v[i])))
			}
		case hexEncoding:
			sb.Write([]byte(hex.EncodeToString(v)))
		default:
			return fmt.Errorf("unsupported output format for raw: %q",
				o.outputFormat)
		}

		if o.numInstances > 1 && i != o.numInstances-1 {
			sb.WriteByte(delim)
		}
	}

	return nil
}

func (o *ReadCommand) doUtf8String(ctx context.Context, procReader *processReader, info progctl.ExeInfo, sb *strings.Builder) error {
	if o.sizeBytes == 0 {
		return readNumBytesRequiredErr(o.dataType)
	}

	var endian binary.ByteOrder = binary.LittleEndian

	switch o.dataType {
	case stringbeDataType, utf8beDataType:
		endian = binary.BigEndian
	}

	for i := uint64(0); i < o.numInstances; i++ {
		v := make([]byte, o.sizeBytes)

		err := binary.Read(procReader, endian, &v)
		if err != nil {
			return err
		}

		procReader.OffsetBy(int64(o.sizeBytes))

		delim := byte(' ')

		switch o.outputFormat {
		case rawEncoding:
			sb.Write(v)
		case hexdumpEncoding, "":
			delim = '\n'

			err = hexdump.Dump(ctx, hexdump.Config{
				Src:            bytes.NewReader(v),
				Dst:            sb,
				Colors:         hexdump.NewColors(),
				OptStartOffset: uint64(procReader.LastReadAddr()),
				OptOffsetBits:  info.Bits,
			})
			if err != nil {
				return fmt.Errorf("failed to hexdump data - %w", err)
			}
		case binaryEncoding:
			for i := range v {
				sb.WriteString(fmt.Sprintf(fmt.Sprintf("%08b", v[i])))
			}
		case hexEncoding:
			sb.Write([]byte(hex.EncodeToString(v)))
		default:
			return fmt.Errorf("unsupported output format for uint8: %q",
				o.outputFormat)
		}

		if o.numInstances > 1 && i != o.numInstances-1 {
			sb.WriteByte(delim)
		}
	}

	return nil
}

func (o *ReadCommand) doUtf16String(ctx context.Context, procReader *processReader, info progctl.ExeInfo, sb *strings.Builder) error {
	if o.sizeBytes == 0 {
		return readNumBytesRequiredErr(o.dataType)
	}

	var endian binary.ByteOrder = binary.LittleEndian

	switch o.dataType {
	case utf16beDataType, wstringbeDataType:
		endian = binary.BigEndian
	}

	var buf bytes.Buffer

	procReader.SaveReadsTo(&buf)

	for i := uint64(0); i < o.numInstances; i++ {
		v := make([]uint16, o.sizeBytes)

		err := binary.Read(procReader, endian, &v)
		if err != nil {
			return err
		}

		str := utf16.Decode(v)

		procReader.OffsetBy(int64(o.sizeBytes))

		delim := byte(' ')

		switch o.outputFormat {
		case rawEncoding:
			sb.WriteString(string(str))
		case hexdumpEncoding, "":
			delim = '\n'

			err = hexdump.Dump(ctx, hexdump.Config{
				Src:            &buf,
				Dst:            sb,
				Colors:         hexdump.NewColors(),
				OptStartOffset: uint64(procReader.LastReadAddr()),
				OptOffsetBits:  info.Bits,
			})
			if err != nil {
				return fmt.Errorf("failed to hexdump data - %w", err)
			}
		case binaryEncoding:
			for i := range v {
				sb.WriteString(fmt.Sprintf(fmt.Sprintf("%08b", v[i])))
			}
		case hexEncoding:
			sb.Write([]byte(fmt.Sprintf("%x", str)))
		default:
			return fmt.Errorf("unsupported output format for utf16: %q",
				o.outputFormat)
		}

		if o.numInstances > 1 && i != o.numInstances-1 {
			sb.WriteByte(delim)
		}
	}

	return nil
}

func (o *ReadCommand) doUnit16(ctx context.Context, procReader *processReader, info progctl.ExeInfo, sb *strings.Builder) error {
	var endian binary.ByteOrder = binary.LittleEndian

	if o.dataType == uint16beDataType {
		endian = binary.BigEndian
	}

	var buf bytes.Buffer

	procReader.SaveReadsTo(&buf)

	for i := uint64(0); i < o.numInstances; i++ {
		var v uint16

		err := binary.Read(procReader, endian, &v)
		if err != nil {
			return err
		}

		procReader.OffsetBy(2)

		delim := byte(' ')

		switch o.outputFormat {
		case hexdumpEncoding:
			delim = '\n'

			err = hexdump.Dump(ctx, hexdump.Config{
				Src:            &buf,
				Dst:            sb,
				Colors:         hexdump.NewColors(),
				OptStartOffset: uint64(procReader.LastReadAddr()),
				OptOffsetBits:  info.Bits,
			})
			if err != nil {
				return fmt.Errorf("failed to hexdump data - %w", err)
			}
		case binaryEncoding:
			sb.WriteString(strconv.FormatUint(uint64(v), 2))
		case hexEncoding:
			sb.WriteString(strconv.FormatUint(uint64(v), 16))
		case "":
			sb.WriteString(strconv.FormatUint(uint64(v), 10))
		default:
			return fmt.Errorf("unsupported output format for uint16: %q",
				o.outputFormat)
		}

		buf.Reset()

		if o.numInstances > 1 && i != o.numInstances-1 {
			sb.WriteByte(delim)
		}
	}

	return nil
}

func (o *ReadCommand) doUint32(ctx context.Context, procReader *processReader, info progctl.ExeInfo, sb *strings.Builder) error {
	var endian binary.ByteOrder = binary.LittleEndian

	if o.dataType == uint16beDataType {
		endian = binary.BigEndian
	}

	var buf bytes.Buffer

	procReader.SaveReadsTo(&buf)

	for i := uint64(0); i < o.numInstances; i++ {
		var v uint32

		err := binary.Read(procReader, endian, &v)
		if err != nil {
			return err
		}

		procReader.OffsetBy(4)

		delim := byte(' ')

		switch o.outputFormat {
		case hexdumpEncoding:
			delim = '\n'

			err = hexdump.Dump(ctx, hexdump.Config{
				Src:            &buf,
				Dst:            sb,
				Colors:         hexdump.NewColors(),
				OptStartOffset: uint64(procReader.LastReadAddr()),
				OptOffsetBits:  info.Bits,
			})
			if err != nil {
				return fmt.Errorf("failed to hexdump data - %w", err)
			}
		case binaryEncoding:
			sb.WriteString(strconv.FormatUint(uint64(v), 2))
		case hexEncoding:
			sb.WriteString(strconv.FormatUint(uint64(v), 16))
		case "":
			sb.WriteString(strconv.FormatUint(uint64(v), 10))
		default:
			return fmt.Errorf("unsupported output format for uint32: %q",
				o.outputFormat)
		}

		buf.Reset()

		if o.numInstances > 1 && i != o.numInstances-1 {
			sb.WriteByte(delim)
		}
	}

	return nil
}

func (o *ReadCommand) doUint64(ctx context.Context, procReader *processReader, info progctl.ExeInfo, sb *strings.Builder) error {
	var endian binary.ByteOrder = binary.LittleEndian

	if o.dataType == uint16beDataType {
		endian = binary.BigEndian
	}

	var buf bytes.Buffer

	procReader.SaveReadsTo(&buf)

	for i := uint64(0); i < o.numInstances; i++ {
		var v uint64

		err := binary.Read(procReader, endian, &v)
		if err != nil {
			return err
		}

		procReader.OffsetBy(8)

		delim := byte(' ')

		switch o.outputFormat {
		case hexdumpEncoding:
			delim = '\n'

			err = hexdump.Dump(ctx, hexdump.Config{
				Src:            &buf,
				Dst:            sb,
				Colors:         hexdump.NewColors(),
				OptStartOffset: uint64(procReader.LastReadAddr()),
				OptOffsetBits:  info.Bits,
			})
			if err != nil {
				return fmt.Errorf("failed to hexdump data - %w", err)
			}
		case binaryEncoding:
			sb.WriteString(strconv.FormatUint(v, 2))
		case hexEncoding:
			sb.WriteString(strconv.FormatUint(v, 16))
		case "":
			sb.WriteString(strconv.FormatUint(v, 10))
		default:
			return fmt.Errorf("unsupported output format for uint64: %q",
				o.outputFormat)
		}

		buf.Reset()

		if o.numInstances > 1 && i != o.numInstances-1 {
			sb.WriteByte(delim)
		}
	}

	return nil
}

func (o *ReadCommand) doFloat32(ctx context.Context, procReader *processReader, info progctl.ExeInfo, sb *strings.Builder) error {
	var endian binary.ByteOrder = binary.LittleEndian

	if o.dataType == uint16beDataType {
		endian = binary.BigEndian
	}

	var buf bytes.Buffer

	procReader.SaveReadsTo(&buf)

	for i := uint64(0); i < o.numInstances; i++ {
		var v float32

		err := binary.Read(procReader, endian, &v)
		if err != nil {
			return err
		}

		procReader.OffsetBy(4)

		delim := byte(' ')

		switch o.outputFormat {
		case hexdumpEncoding:
			delim = '\n'

			err = hexdump.Dump(ctx, hexdump.Config{
				Src:            &buf,
				Dst:            sb,
				Colors:         hexdump.NewColors(),
				OptStartOffset: uint64(procReader.LastReadAddr()),
				OptOffsetBits:  info.Bits,
			})
			if err != nil {
				return fmt.Errorf("failed to hexdump data - %w", err)
			}
		case binaryEncoding:
			sb.WriteString(strconv.FormatFloat(float64(v), 'b', 4, 32))
		case hexEncoding:
			sb.WriteString(strconv.FormatFloat(float64(v), 'x', 4, 32))
		case "":
			sb.WriteString(strconv.FormatFloat(float64(v), 'f', 4, 32))
		default:
			return fmt.Errorf("unsupported output format for float32: %q",
				o.outputFormat)
		}

		buf.Reset()

		if o.numInstances > 1 && i != o.numInstances-1 {
			sb.WriteByte(delim)
		}
	}

	return nil
}

func (o *ReadCommand) doFloat64(ctx context.Context, procReader *processReader, info progctl.ExeInfo, sb *strings.Builder) error {
	var endian binary.ByteOrder = binary.LittleEndian

	if o.dataType == uint16beDataType {
		endian = binary.BigEndian
	}

	var buf bytes.Buffer

	procReader.SaveReadsTo(&buf)

	for i := uint64(0); i < o.numInstances; i++ {
		var v float64

		err := binary.Read(procReader, endian, &v)
		if err != nil {
			return err
		}

		procReader.OffsetBy(8)

		delim := byte(' ')

		switch o.outputFormat {
		case hexdumpEncoding:
			delim = '\n'

			err = hexdump.Dump(ctx, hexdump.Config{
				Src:            &buf,
				Dst:            sb,
				Colors:         hexdump.NewColors(),
				OptStartOffset: uint64(procReader.LastReadAddr()),
				OptOffsetBits:  info.Bits,
			})
			if err != nil {
				return fmt.Errorf("failed to hexdump data - %w", err)
			}
		case binaryEncoding:
			sb.WriteString(strconv.FormatFloat(v, 'b', 4, 64))
		case hexEncoding:
			sb.WriteString(strconv.FormatFloat(v, 'x', 4, 64))
		case "":
			sb.WriteString(strconv.FormatFloat(v, 'f', 4, 64))
		default:
			return fmt.Errorf("unsupported output format for float64: %q",
				o.outputFormat)
		}

		buf.Reset()

		if o.numInstances > 1 && i != o.numInstances-1 {
			sb.WriteByte(delim)
		}
	}

	return nil
}

func readNumBytesRequiredErr(format string) error {
	return fmt.Errorf("please specify the number of bytes to read when using the %q type",
		format)
}

func newProcessReader(ctx context.Context, addr string, ctl *progctl.Ctl) *processReader {
	return &processReader{
		ctx:     ctx,
		addr:    addr,
		process: ctl,
	}
}

type processReader struct {
	ctx     context.Context
	addr    string
	process *progctl.Ctl

	saveReadsTo io.Writer

	useLastReadAddr bool
	lastReadAddr    uintptr
}

func (o *processReader) Read(b []byte) (int, error) {
	var data []byte
	var actualAddr uintptr
	var err error
	size := uint64(len(b))

	if o.useLastReadAddr {
		data, actualAddr, err = o.process.ReadFromAddr(o.ctx, memory.AbsoluteAddrPointer(o.lastReadAddr), size)
	} else {
		data, actualAddr, err = o.process.ReadFromLookup(o.ctx, o.addr, size)
	}

	if err != nil {
		return 0, err
	}

	o.lastReadAddr = actualAddr

	i := copy(b, data)

	if o.saveReadsTo != nil {
		_, err := o.saveReadsTo.Write(b[0:i])
		if err != nil {
			return 0, fmt.Errorf("failed to write data to save-reads-to writer - %w", err)
		}
	}

	return i, nil
}

func (o *processReader) SaveReadsTo(w io.Writer) {
	o.saveReadsTo = w
}

func (o *processReader) OffsetBy(i int64) {
	if !o.useLastReadAddr {
		o.useLastReadAddr = true
	}

	o.lastReadAddr += uintptr(i)
}

func (o *processReader) LastReadAddr() uintptr {
	return o.lastReadAddr
}
