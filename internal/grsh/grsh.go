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

	grumbleApp.AddCommand(NewFindCommand(session))

	//grumbleApp.AddCommand(NewSeekCommand(session))

	grumbleApp.AddCommand(NewReadCommand(session))

	grumbleApp.AddCommand(NewWriteCommand(session))

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
