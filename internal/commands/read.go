package commands

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/SeungKang/memshonk/internal/memory"
)

func ReadCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      "read",
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
				DefValue: uint64(0),
				DataType: uint64(0),
			},
			{
				Name:     "addr",
				Desc:     "address to read from",
				DataType: "",
				DefValue: "",
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

func (o ReadCommand) Run(ctx context.Context, inOut IO, s Session) error {
	var fmtFn func([]byte) error

	// TODO: Document encoding formats
	encodingFormat := o.args.EncodingFormat
	switch encodingFormat {
	case "hexdump":
		fmtFn = func(b []byte) error {
			_, err := fmt.Fprintln(inOut.Stdout, hex.Dump(b))
			return err
		}
	case "hex":
		fmtFn = func(b []byte) error {
			_, err := fmt.Fprintln(inOut.Stdout, hex.EncodeToString(b))
			return err
		}
	case "b64", "base64":
		fmtFn = func(b []byte) error {
			_, err := fmt.Fprintln(inOut.Stdout, base64.StdEncoding.EncodeToString(b))
			return err
		}
	default:
		return fmt.Errorf("unknown encoding format: %q", encodingFormat)
	}

	var ptr memory.Pointer
	addrStr := o.args.AddrStr
	var err error
	if addrStr == "" {
		return errors.New("TODO: implement seek address support")
	} else {
		ptr, err = memory.CreatePointerFromString(addrStr)
		if err != nil {
			return err
		}
	}

	data, err := s.Process().ReadFromAddr(ctx, ptr, o.args.SizeBytes)
	if err != nil {
		return err
	}

	return fmtFn(data)
}
