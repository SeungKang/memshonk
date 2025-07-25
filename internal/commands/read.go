package commands

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/SeungKang/memshonk/internal/hexdump"
	"github.com/SeungKang/memshonk/internal/memory"
)

const (
	readCommandName = "read"
)

func ReadCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      readCommandName,
		Aliases:   []string{"r"},
		ShortHelp: "read n bytes from addr",
		Flags: []FlagSchema{
			{
				Short:      "e",
				Long:       "encoding",
				Desc:       "Optional: Specify output encoding format",
				DataType:   "",
				DefaultVal: "hexdump",
			},
		},
		NonFlags: []NonFlagSchema{
			{
				Name:     "size",
				Desc:     "number of bytes to read",
				DataType: uint64(0),
			},
			{
				Name:     "addr",
				Desc:     "address to read from",
				DataType: "",
			},
		},
		CreateFn: func(c CommandConfig) (Command, error) {
			return NewReadCommand(ReadCommandArgs{
				EncodingFormat: c.Flags.String("encoding"),
				SizeBytes:      c.NonFlags.Uint64("size"),
				AddrStr:        c.NonFlags.String("addr"),
			}), nil
		},
	}
}

type ReadCommandArgs struct {
	EncodingFormat string
	SizeBytes      uint64
	AddrStr        string
}

func NewReadCommand(args ReadCommandArgs) ReadCommand {
	return ReadCommand{
		args: args,
	}
}

type ReadCommand struct {
	args ReadCommandArgs
}

func (o ReadCommand) Name() string {
	return readCommandName
}

func (o ReadCommand) Run(ctx context.Context, inOut IO, s Session) (CommandResult, error) {
	var fmtFn func([]byte, uintptr) (string, error)

	// TODO: Document encoding formats
	encodingFormat := o.args.EncodingFormat
	switch encodingFormat {
	case "hexdump":
		info, err := s.Process().ExeInfo(ctx)
		if err != nil {
			return nil, err
		}

		fmtFn = func(b []byte, from uintptr) (string, error) {
			var out bytes.Buffer

			err = hexdump.Dump(ctx, hexdump.Config{
				Src:          bytes.NewReader(b),
				Dst:          &out,
				Colors:       hexdump.NewColors(),
				OptStartOff:  uint64(from),
				OptOffColPad: info.Bits / 4, // 32 == 8, 64 = 16.
			})

			return out.String(), nil
		}
	case "hex":
		fmtFn = func(b []byte, _ uintptr) (string, error) {
			return hex.EncodeToString(b), nil
		}
	case "b64", "base64":
		fmtFn = func(b []byte, _ uintptr) (string, error) {
			return base64.StdEncoding.EncodeToString(b), nil
		}
	default:
		return nil, fmt.Errorf("unknown encoding format: %q", encodingFormat)
	}

	ptr, err := memory.CreatePointerFromString(o.args.AddrStr)
	if err != nil {
		return nil, err
	}

	data, actualAddr, err := s.Process().ReadFromAddr(ctx, ptr, o.args.SizeBytes)
	if err != nil {
		return nil, err
	}

	encoded, err := fmtFn(data, actualAddr)
	if err != nil {
		return nil, err
	}

	return HumanCommandResult(encoded), nil
}
