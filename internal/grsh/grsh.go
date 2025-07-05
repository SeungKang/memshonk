package grsh

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/SeungKang/memshonk/internal/app"
	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/SeungKang/memshonk/internal/events"
	"github.com/desertbit/grumble"
	"github.com/fatih/color"
)

func NewShell(ctx context.Context, session *app.Session) (*Shell, error) {
	// TODO: This is terrible, but grumble makes the assumption
	// that it should parse process arguments and it always tries
	// to parse them via os.Args (we want to do that ourselves
	// using the flag library).
	os.Args = os.Args[0:1]

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get user home dir - %w", err)
	}

	configDir := filepath.Join(homeDir, ".memshonk")
	err = os.MkdirAll(configDir, 0o700)
	if err != nil {
		return nil, fmt.Errorf("failed to make config directory at '%s' - %w", configDir, err)
	}

	grumbleApp := grumble.New(&grumble.Config{
		Name:        "memshonk",
		HistoryFile: filepath.Join(configDir, "history"),
		PromptColor: color.New(color.FgCyan),
		// CommandPreProc: func(args []string) ([]string, error) {
		// 	err := shvars.Replace(args, session.Variables())
		// 	if err != nil {
		// 		return args, err
		// 	}
		//
		// 	return args, nil
		// },
	})

	sh := &Shell{
		ga:  grumbleApp,
		ctx: ctx,
	}

	grumbleApp.OnInit(sh.onInit)

	for _, cmdSchema := range commands.BuiltinCommands() {
		grumbleApp.AddCommand(commandSchemaToGrumbleCommand(
			cmdSchema, session))
	}

	attachEvents := events.NewSubscriber[commands.AttachEvent](session.Events())
	detachEvents := events.NewSubscriber[commands.DetachEvent](session.Events())

	go func() {
		defer attachEvents.Unsubscribe()
		defer detachEvents.Unsubscribe()

		for {
			select {
			case <-ctx.Done():
				return
			case e := <-attachEvents.RecvCh():
				sh.setPrompt(e.Pid)

				close(e.Done)
			case e := <-detachEvents.RecvCh():
				sh.setPrompt(0)

				close(e.Done)
			}
		}
	}()

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
	o.setPrompt(0)

	return nil
}

// TODO: implement seek address
func (o *Shell) setPrompt(pid int) {
	if pid == 0 {
		o.ga.SetPrompt("$ ")
	} else {
		o.ga.SetPrompt(fmt.Sprintf("[%d] $ ", pid))
	}
}
