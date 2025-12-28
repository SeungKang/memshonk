package grsh

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/SeungKang/memshonk/internal/app"
	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/progctl"

	"github.com/desertbit/grumble"
	"github.com/desertbit/readline"
	"github.com/fatih/color"
)

func NewShell(ctx context.Context, session *app.Session) (*Shell, error) {
	// TODO: This is terrible, but grumble makes the assumption
	// that it should parse process arguments and it always tries
	// to parse them via os.Args (we want to do that ourselves
	// using the flag library).
	os.Args = os.Args[0:1]

	grumbleConfig := &grumble.Config{
		Name:        "memshonk",
		PromptColor: color.New(color.FgCyan),

		// The default InterruptHandler calls os.Exit,
		// which is not exactly ideal if a remote client
		// decides to do ctrl+c.
		InterruptHandler: func(*grumble.App, int) {},

		// TODO:
		// CommandPreProc: func(args []string) ([]string, error) {
		// 	err := shvars.Replace(args, session.Variables())
		// 	if err != nil {
		// 		return args, err
		// 	}
		//
		// 	return args, nil
		// },
	}

	wsConfig := session.Project().WorkspaceConfig()

	historyFilePath, historyEnabled := wsConfig.HistoryFilePath(session.ID())
	if historyEnabled {
		grumbleConfig.HistoryFile = historyFilePath
	}

	grumbleApp := grumble.New(grumbleConfig)

	sh := &Shell{
		ga:  grumbleApp,
		ctx: ctx,
	}

	grumbleApp.OnInit(sh.onInit)

	for _, cmdSchema := range commands.BuiltinCommands() {
		grumbleApp.AddCommand(commandSchemaToGrumbleCommand(
			cmdSchema, session))
	}

	attachEvents := events.NewSubscriber[progctl.AttachedEvent](session.Events())
	detachEvents := events.NewSubscriber[progctl.DetachedEvent](session.Events())
	exitedEvents := events.NewSubscriber[progctl.ProcessExitedEvent](session.Events())
	loadedEvents := events.NewSubscriber[plugins.LoadedEvent](session.Events())
	unloadedEvents := events.NewSubscriber[plugins.UnloadedEvent](session.Events())

	go func() {
		defer attachEvents.Unsubscribe()
		defer detachEvents.Unsubscribe()
		defer exitedEvents.Unsubscribe()
		defer loadedEvents.Unsubscribe()
		defer unloadedEvents.Unsubscribe()

		for {
			select {
			case <-ctx.Done():
				return
			case e := <-attachEvents.RecvCh():
				sh.setPrompt(e.Pid)

				close(e.Acked)
			case e := <-detachEvents.RecvCh():
				sh.setPrompt(0)

				close(e.Acked)
			case e := <-exitedEvents.RecvCh():
				sh.setPrompt(0)
				log.Printf("process exited - %v", e.Reason)
			case e := <-loadedEvents.RecvCh():
				grumbleApp.AddCommand(newPluginCommand(e.Plugin, session))
			case e := <-unloadedEvents.RecvCh():
				grumbleApp.Commands().Remove(e.Plugin.Name())
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

func (o *Shell) Run(stdin io.ReadCloser, stdout io.WriteCloser, stderr io.WriteCloser) error {
	config := &readline.Config{Stdin: stdin, Stdout: stdout, Stderr: stderr}
	rl, err := readline.NewEx(config)
	if err != nil {
		return err
	}

	o.ga.SetReadlineDefaults(config)
	return o.ga.RunWithReadline(rl)
}

func (o *Shell) onInit(_ *grumble.App, flags grumble.FlagMap) error {
	o.fm = flags
	o.setPrompt(0)

	return nil
}

func (o *Shell) setPrompt(pid int) {
	if pid == 0 {
		o.ga.SetPrompt("$ ")
	} else {
		o.ga.SetPrompt(fmt.Sprintf("[%d] $ ", pid))
	}
}
