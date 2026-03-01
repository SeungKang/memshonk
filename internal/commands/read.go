package commands

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/hex"
	"fmt"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/hexdump"
)

const (
	ReadCommandName = "readm"
)

func NewReadCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &ReadCommand{
		session: config.Session,
	}

	root := fx.NewCommand(ReadCommandName, "read n bytes from addr", cmd.read)

	root.FlagSet.StringFlag(&cmd.encodingFormat, "hexdump", fx.ArgConfig{
		Name:        "encoding",
		Description: "Optional: Specify output encoding format",
	})

	root.FlagSet.Uint64Nf(&cmd.sizeBytes, fx.ArgConfig{
		Name:        "size",
		Description: "number of bytes to read",
		Required:    true,
	})

	root.FlagSet.StringNf(&cmd.addrStr, fx.ArgConfig{
		Name:        "addr",
		Description: "address to read from",
		Required:    true,
	})

	return root
}

type ReadCommand struct {
	session        apicompat.Session
	encodingFormat string
	sizeBytes      uint64
	addrStr        string
}

func (o *ReadCommand) read(ctx context.Context) (fx.CommandResult, error) {
	var fmtFn func([]byte, uintptr) (string, error)

	// TODO: Document encoding formats
	encodingFormat := o.encodingFormat
	switch encodingFormat {
	case "hexdump":
		info, err := o.session.SharedState().Progctl.ExeInfo(ctx)
		if err != nil {
			return nil, err
		}

		fmtFn = func(b []byte, from uintptr) (string, error) {
			var out bytes.Buffer

			err = hexdump.Dump(ctx, hexdump.Config{
				Src:            bytes.NewReader(b),
				Dst:            &out,
				Colors:         hexdump.NewColors(),
				OptStartOffset: uint64(from),
				OptOffsetBits:  info.Bits,
			})
			if err != nil {
				return "", err
			}

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

	data, actualAddr, err := o.session.SharedState().Progctl.ReadFromLookup(ctx, o.addrStr, o.sizeBytes)
	if err != nil {
		return nil, err
	}

	encoded, err := fmtFn(data, actualAddr)
	if err != nil {
		return nil, err
	}

	return fx.NewHumanCommandResult(encoded), nil
}
