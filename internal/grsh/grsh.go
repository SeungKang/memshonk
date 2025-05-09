package grsh

import (
	"context"
	"fmt"
	"github.com/SeungKang/memshonk/internal/app"
	"github.com/desertbit/grumble"
)

func NewShell(ctx context.Context, session *app.Session) (*Shell, error) {
	grumbleApp := grumble.New(&grumble.Config{
		Name:        "xmempg",
		Description: "Wrapper for mempg",
		Flags: func(f *grumble.Flags) {
			f.String(
				"E",
				"mempg-exe",
				"Path to mempg executable",
				"mempg")

			f.Bool(
				"D",
				"insecure-disable-sandbox",
				false,
				"Disable mempg sandbox")
		},
	})

	grumbleApp.SetInterruptHandler(func(a *grumble.App, count int) {
		a.Close()
	})

	sh := &Shell{
		ga:  grumbleApp,
		ctx: ctx,
	}

	grumbleApp.OnInit(sh.onInit)

	grumbleApp.AddCommand(NewAttachCommand(session))

	//grumbleApp.AddCommand(NewSeekCommand(session))

	grumbleApp.AddCommand(NewReadCommand(session))

	//grumbleApp.AddCommand(NewWriteCommand(session))

	return sh, nil
}

type Shell struct {
	ga  *grumble.App
	fm  grumble.FlagMap
	ctx context.Context
}

func (o *Shell) Run() error {
	return o.ga.Run()
}

func (o *Shell) onInit(_ *grumble.App, flags grumble.FlagMap) error {
	o.fm = flags
	o.setPrompt()

	return nil
}

//func (o *Shell) seek(c *grumble.Context) error {
//	addr, err := strconv.ParseUint(strings.TrimPrefix(c.Args.String("addr"), "0x"), 16, 64)
//	if err != nil {
//		return err
//	}
//
//	err = o.pg.Seek(uintptr(addr))
//	if err != nil {
//		return err
//	}
//
//	o.setPrompt()
//
//	return nil
//}

// TODO: implement seek address
func (o *Shell) setPrompt() {
	o.ga.SetPrompt(fmt.Sprintf("[0x%x] $ ", 0))
}

//func (o *Shell) write(c *grumble.Context) error {
//	dataStr := c.Args.String("data")
//	var data []byte
//
//	encodingFormat := c.Flags.String("encoding")
//	switch encodingFormat {
//	case "raw":
//		data = []byte(dataStr)
//	case "hexdump":
//		return errors.New("TODO: someday invert hexdump -C output into bytes")
//	case "hex":
//		var err error
//		data, err = hex.DecodeString(strings.TrimPrefix(dataStr, "0x"))
//		if err != nil {
//			return fmt.Errorf("failed to hex decode string - %w", err)
//		}
//	case "b64", "base64":
//		var err error
//		data, err = base64.StdEncoding.DecodeString(dataStr)
//		if err != nil {
//			return fmt.Errorf("failed to base64 decode string - %w", err)
//		}
//	case "ptr", "pointer":
//		var err error
//		data, err = hex.DecodeString(strings.TrimPrefix(dataStr, "0x"))
//		if err != nil {
//			return fmt.Errorf("failed to hex decode string - %w", err)
//		}
//
//		if len(data) > 8 {
//			return fmt.Errorf("pointer cannot be greater than 8 bytes, got %d", len(data))
//		}
//
//		switch {
//		case len(data) > 8:
//			return fmt.Errorf("pointer cannot be greater than 8 bytes, got %d", len(data))
//		case len(data) < 8:
//			data = append(bytes.Repeat([]byte{0}, 8-len(data)), data...)
//		}
//
//		binary.LittleEndian.PutUint64(data, binary.BigEndian.Uint64(data))
//	default:
//		return fmt.Errorf("unknown encoding format: %q", encodingFormat)
//	}
//
//	addrStr := c.Args.String("addr")
//	var err error
//	if addrStr == "" {
//		err = o.pg.WriteMemoryAtSeekedAddr(data)
//	} else {
//		var addr uint64
//		addr, err = strconv.ParseUint(strings.TrimPrefix(addrStr, "0x"), 16, 64)
//		if err != nil {
//			return err
//		}
//
//		err = o.pg.WriteMemoryAtAddr(data, uintptr(addr))
//	}
//
//	if err != nil {
//		return err
//	}
//
//	return nil
//}
