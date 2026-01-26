package app

import (
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/progctl"
	"github.com/SeungKang/memshonk/internal/project"
	"github.com/SeungKang/memshonk/internal/shvars"

	"github.com/SeungKang/memshonk/internal/vendored/goterm"
)

// various signal message types
const (
	unknownSignalType uint8 = iota
	IntSignalType
)

func newSession(id string, app *App, sessionIO SessionIO, isDefault bool) *Session {
	return &Session{
		id:        id,
		isDefault: isDefault,
		app:       app,
		vars: &SessionVariables{
			proj: app.project,
			vars: &shvars.Variables{},
		},
		io: sessionIO,
	}
}

type Session struct {
	id        string
	isDefault bool
	app       *App
	vars      *SessionVariables
	cmdCtx    *CommandContext
	io        SessionIO

	cancelCmdMu  sync.Mutex
	cancelCmdCtx context.CancelFunc
}

func (o *Session) ID() string {
	return o.id
}

type SessionIO struct {
	Stdin  io.ReadCloser
	Stdout io.WriteCloser
	Stderr io.WriteCloser

	OptTerminal goterm.TerminalWithNotifications
}

func (o *Session) IO() SessionIO {
	return o.io
}

func (o *Session) OnSignal(signalType uint8) {
	switch signalType {
	case IntSignalType:
		o.cancelCurrentCommand()
	default:
		// ignore or log unknown signals
	}
}

func (o *Session) Events() *events.Groups {
	return o.app.events
}

func (o *Session) Project() *project.Project {
	return o.app.project
}

func (o *Session) Variables() *SessionVariables {
	return o.vars
}

func (o *Session) Plugins() (plugins.Ctl, bool) {
	if o.app.pluginCtl == nil {
		return nil, false
	}

	return o.app.pluginCtl, true
}

func (o *Session) Terminal() (goterm.TerminalWithNotifications, bool) {
	if o.io.OptTerminal != nil {
		return o.io.OptTerminal, true
	}

	return nil, false
}

func (o *Session) RunCommand(parent context.Context, cmd commands.Command) error {
	ctx, cancelFn := context.WithCancel(parent)
	defer func() {
		o.clearCancel()
		cancelFn()
	}()

	// install the cancel lever for OnSignal
	o.setCancel(func() {
		cancelFn()
	})

	result, err := cmd.Run(ctx, commands.IO{
		Stdout: o.io.Stdout,
		Stderr: o.io.Stderr,
	}, o)
	if err != nil {
		return err
	}

	if result != nil {
		o.io.Stdout.Write(result.Serialize())
		o.io.Stdout.Write([]byte{'\n'})
	}

	return nil
}

func (o *Session) setCancel(fn context.CancelFunc) {
	o.cancelCmdMu.Lock()
	defer o.cancelCmdMu.Unlock()
	o.cancelCmdCtx = fn
}

func (o *Session) clearCancel() {
	o.cancelCmdMu.Lock()
	defer o.cancelCmdMu.Unlock()
	o.cancelCmdCtx = nil
}

func (o *Session) cancelCurrentCommand() {
	o.cancelCmdMu.Lock()
	fn := o.cancelCmdCtx
	o.cancelCmdMu.Unlock()

	if fn != nil {
		fn()
	}
}

func (o *Session) Process() progctl.Process {
	return o.app.ProcCtl()
}

type SessionVariables struct {
	proj *project.Project
	vars *shvars.Variables
}

func (o *SessionVariables) Len() int {
	numProjVars := o.proj.Variables().Len()

	numSessionVars := o.vars.Len()

	return numProjVars + numSessionVars
}

func (o *SessionVariables) Set(name string, value string) error {
	projVars := o.proj.Variables()

	_, alreadyProjectVar := projVars.Get(name)
	if alreadyProjectVar {
		return fmt.Errorf("variable is already set as a project variable (%q)",
			name)
	}

	return o.vars.Set(name, value)
}

func (o *SessionVariables) Get(name string) (string, bool) {
	value, hasProjectVar := o.proj.Variables().Get(name)
	if hasProjectVar {
		return value, true
	}

	value, hasSessionVar := o.vars.Get(name)
	if hasSessionVar {
		return value, true
	}

	return "", false
}
