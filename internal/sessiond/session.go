package sessiond

import (
	"context"
	"fmt"
	"sync"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/project"
	"github.com/SeungKang/memshonk/internal/shvars"

	"github.com/SeungKang/memshonk/internal/vendored/goterm"
)

// various signal message types
const (
	unknownSignalType uint8 = iota
	IntSignalType
)

type Session struct {
	shared    apicompat.SharedState
	id        string
	isDefault bool
	vars      *SessionVariables
	io        apicompat.SessionIO
	stopper   *sessionStopper

	cancelCmdMu  sync.Mutex
	cancelCmdCtx context.CancelFunc
}

func (o *Session) Ctx() context.Context {
	return o.stopper.ctx
}

func (o *Session) Done() <-chan struct{} {
	return o.stopper.ctx.Done()
}

func (o *Session) Close() error {
	return o.stopper.Close()
}

func (o *Session) SharedState() apicompat.SharedState {
	return o.shared
}

func (o *Session) ID() string {
	return o.id
}

func (o *Session) IO() apicompat.SessionIO {
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

func (o *Session) Variables() *SessionVariables {
	return o.vars
}

func (o *Session) Terminal() (goterm.TerminalWithNotifications, bool) {
	if o.io.OptTerminal != nil {
		return o.io.OptTerminal, true
	}

	return nil, false
}

func (o *Session) RunCommand(parent context.Context, cmd apicompat.Command) error {
	ctx, cancelFn := context.WithCancel(parent)
	defer func() {
		o.clearCancel()
		cancelFn()
	}()

	go func() {
		select {
		case <-ctx.Done():
		case <-o.stopper.ctx.Done():
			cancelFn()
		}
	}()

	// install the cancel lever for OnSignal
	o.setCancel(func() {
		cancelFn()
	})

	result, err := cmd.Run(ctx, o)
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
