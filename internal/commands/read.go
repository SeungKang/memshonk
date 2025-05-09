package commands

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/SeungKang/memshonk/internal/memory"
)

var _ Command = (*ReadCommand)(nil)

type ReadCommandArgs struct {
	EncodingFormat string
	AddrStr        string
	Size           uint
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

	addrStr := o.args.AddrStr
	var data []byte
	var err error
	if addrStr == "" {
		return errors.New("TODO: implement seek address support")
	} else {
		var addr uint64
		addr, err = strconv.ParseUint(strings.TrimPrefix(addrStr, "0x"), 16, 64)
		if err != nil {
			return err
		}

		// TODO: support specifying OptModule
		data, err = s.Process().ReadFromAddr(ctx, memory.Pointer{
			Name:      "",
			Addrs:     []uintptr{uintptr(addr)},
			OptModule: "",
		}, o.args.Size)
	}

	if err != nil {
		return err
	}

	return fmtFn(data)
}
