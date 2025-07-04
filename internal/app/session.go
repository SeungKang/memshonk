package app

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/progctl"
	"github.com/SeungKang/memshonk/internal/project"
	"github.com/SeungKang/memshonk/internal/shvars"
)

func newSession(id uint64, app *App, cmdIO commands.IO) *Session {
	return &Session{
		id:  id,
		app: app,
		vars: &SessionVariables{
			proj: app.project,
			vars: &shvars.Variables{},
		},
		io: cmdIO,
	}
}

type Session struct {
	id     uint64
	app    *App
	vars   *SessionVariables
	cmdCtx *CommandContext
	io     commands.IO
}

func (o *Session) Events() *events.EventsPubSub {
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

func (o *Session) RunCommand(ctx context.Context, cmd commands.Command) error {
	result, err := cmd.Run(ctx, o.io, o)
	if err != nil {
		return err
	}

	if result != nil {
		o.io.Stdout.Write(result.Serialize())
		o.io.Stdout.Write([]byte{'\n'})
	}

	return nil
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
