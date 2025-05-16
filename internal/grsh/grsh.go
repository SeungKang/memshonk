package grsh

import (
	"context"
	"fmt"
	"os"

	"github.com/SeungKang/memshonk/internal/app"
	"github.com/desertbit/grumble"
)

func NewShell(ctx context.Context, session *app.Session) (*Shell, error) {
	// TODO: This is terrible, but grumble makes the assumption
	// that it should parse process arguments and it always tries
	// to parse them via os.Args (we want to do that ourselves
	// using the flag library).
	os.Args = os.Args[0:1]

	grumbleApp := grumble.New(&grumble.Config{
		Name: "memshonk",
		// CommandPreProc: func(args []string) ([]string, error) {
		// 	err := shvars.Replace(args, session.Variables())
		// 	if err != nil {
		// 		return args, err
		// 	}
		//
		// 	return args, nil
		// },
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

	grumbleApp.AddCommand(NewObjectsCommand(session))

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
