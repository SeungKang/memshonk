package commands

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

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

func (o ReadCommand) Run(ctx context.Context, inOut IO, s Session) (CommandResult, error) {
	var fmtFn func([]byte) (string, error)

	// TODO: Document encoding formats
	encodingFormat := o.args.EncodingFormat
	switch encodingFormat {
	case "hexdump":
		fmtFn = func(b []byte) (string, error) {
			return strings.TrimSpace(hex.Dump(b)), nil
		}
	case "hex":
		fmtFn = func(b []byte) (string, error) {
			return hex.EncodeToString(b), nil
		}
	case "b64", "base64":
		fmtFn = func(b []byte) (string, error) {
			return base64.StdEncoding.EncodeToString(b), nil
		}
	default:
		return nil, fmt.Errorf("unknown encoding format: %q", encodingFormat)
	}

	ptr, err := memory.CreatePointerFromString(o.args.AddrStr)
	if err != nil {
		return nil, err
	}

	data, err := s.Process().ReadFromAddr(ctx, ptr, o.args.SizeBytes)
	if err != nil {
		return nil, err
	}

	encoded, err := fmtFn(data)
	if err != nil {
		return nil, err
	}

	return HumanCommandResult(encoded), nil
}
