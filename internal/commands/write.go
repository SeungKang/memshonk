package commands

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/SeungKang/memshonk/internal/memory"
)

func WriteCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      "write",
		Aliases:   []string{"w"},
		ShortHelp: "write value to addr",
		Flags: []FlagSchema{
			{
				Short:      "e",
				Long:       "encoding",
				Desc:       "Optional: Specify output encoding format",
				DataType:   "",
				DefaultVal: "hex",
			},
		},
		NonFlags: []NonFlagSchema{
			{
				Name:     "data",
				Desc:     "data to write",
				DefValue: "",
				DataType: "",
			},
			{
				Name:     "addr",
				Desc:     "address to write to",
				DataType: "",
				DefValue: "",
			},
		},
		CreateFn: func(c CommandConfig) (Command, error) {
			return NewWriteCommand(WriteCommandArgs{
				EncodingFormat: c.Flags.String("encoding"),
				DataStr:        c.NonFlags.String("data"),
				AddrStr:        c.NonFlags.String("addr"),
			}), nil
		},
	}
}

type WriteCommandArgs struct {
	EncodingFormat string
	DataStr        string
	AddrStr        string
}

func NewWriteCommand(args WriteCommandArgs) WriteCommand {
	return WriteCommand{
		args: args,
	}
}

type WriteCommand struct {
	args WriteCommandArgs
}

func (o WriteCommand) Run(ctx context.Context, _ IO, s Session) error {
	dataStr := o.args.DataStr
	var data []byte

	// TODO: Document encoding formats
	encodingFormat := o.args.EncodingFormat
	switch encodingFormat {
	case "raw":
		data = []byte(dataStr)
	case "hexdump":
		return errors.New("TODO: someday invert hexdump -C output into bytes")
	case "hex":
		var err error
		data, err = hex.DecodeString(strings.TrimPrefix(dataStr, "0x"))
		if err != nil {
			return fmt.Errorf("failed to hex decode string - %w", err)
		}
	case "b64", "base64":
		var err error
		data, err = base64.StdEncoding.DecodeString(dataStr)
		if err != nil {
			return fmt.Errorf("failed to base64 decode string - %w", err)
		}
	case "ptr", "pointer":
		var err error
		data, err = hex.DecodeString(strings.TrimPrefix(dataStr, "0x"))
		if err != nil {
			return fmt.Errorf("failed to hex decode string - %w", err)
		}

		if len(data) > 8 {
			return fmt.Errorf("pointer cannot be greater than 8 bytes, got %d", len(data))
		}

		switch {
		case len(data) > 8:
			return fmt.Errorf("pointer cannot be greater than 8 bytes, got %d", len(data))
		case len(data) < 8:
			data = append(bytes.Repeat([]byte{0}, 8-len(data)), data...)
		}

		binary.LittleEndian.PutUint64(data, binary.BigEndian.Uint64(data))
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

	err = s.Process().WriteToAddr(ctx, data, ptr)
	if err != nil {
		return err
	}

	return nil
}
