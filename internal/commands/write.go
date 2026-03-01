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

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
)

const (
	WriteCommandName = "writem"
)

func NewWriteCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &WriteCommand{
		session: config.Session,
	}

	root := fx.NewCommand(WriteCommandName, "write value to addr", cmd.write)

	root.FlagSet.StringFlag(&cmd.encodingFormat, "raw", fx.ArgConfig{
		Name:        "encoding",
		Description: "Optional: Specify output encoding format",
	})

	root.FlagSet.StringNf(&cmd.dataStr, fx.ArgConfig{
		Name:        "data",
		Description: "data to write",
		Required:    true,
	})

	root.FlagSet.StringNf(&cmd.addrStr, fx.ArgConfig{
		Name:        "addr",
		Description: "address to write to",
		Required:    true,
	})

	return root
}

type WriteCommand struct {
	session        apicompat.Session
	encodingFormat string
	dataStr        string
	addrStr        string
}

func (o *WriteCommand) write(ctx context.Context) (fx.CommandResult, error) {
	dataStr := o.dataStr
	var data []byte

	// TODO: Document encoding formats
	encodingFormat := o.encodingFormat
	switch encodingFormat {
	case "raw":
		data = []byte(dataStr)
	case "hexdump":
		return nil, errors.New("TODO: someday invert hexdump -C output into bytes")
	case "hex":
		var err error
		data, err = hex.DecodeString(strings.TrimPrefix(dataStr, "0x"))
		if err != nil {
			return nil, fmt.Errorf("failed to hex decode string - %w", err)
		}
	case "b64", "base64":
		var err error
		data, err = base64.StdEncoding.DecodeString(dataStr)
		if err != nil {
			return nil, fmt.Errorf("failed to base64 decode string - %w", err)
		}
	case "ptr", "pointer":
		var err error
		data, err = hex.DecodeString(strings.TrimPrefix(dataStr, "0x"))
		if err != nil {
			return nil, fmt.Errorf("failed to hex decode string - %w", err)
		}

		if len(data) > 8 {
			return nil, fmt.Errorf("pointer cannot be greater than 8 bytes, got %d", len(data))
		}

		switch {
		case len(data) > 8:
			return nil, fmt.Errorf("pointer cannot be greater than 8 bytes, got %d", len(data))
		case len(data) < 8:
			data = append(bytes.Repeat([]byte{0}, 8-len(data)), data...)
		}

		binary.LittleEndian.PutUint64(data, binary.BigEndian.Uint64(data))
	default:
		return nil, fmt.Errorf("unknown encoding format: %q", encodingFormat)
	}

	_, err := o.session.SharedState().Progctl.WriteToLookup(ctx, o.addrStr, data)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
