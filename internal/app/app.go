package app

import (
	"context"
	"fmt"
	"sync"

	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/SeungKang/memshonk/internal/progctl"
	"github.com/SeungKang/memshonk/internal/project"
	"github.com/SeungKang/memshonk/internal/shvars"
)

func NewApp(project *project.Project) *App {
	return &App{
		project: project,
		procCtl: progctl.NewCtl(project.General().ExeName),
	}
}

type App struct {
	project       *project.Project
	rwMu          sync.RWMutex
	nextSessionId uint64
	sessions      map[uint64]*Session
	procCtl       *progctl.Ctl
}

func (o *App) ProcCtl() *progctl.Ctl {
	return o.procCtl
}

func (o *App) NewSession(cmdIO commands.IO) *Session {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.sessions == nil {
		o.sessions = make(map[uint64]*Session)
	}

	id := o.nextSessionId
	o.nextSessionId++

	session := newSession(id, o, cmdIO)
	o.sessions[id] = session

	return session
}

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

func (o *Session) Project() *project.Project {
	return o.app.project
}

func (o *Session) Variables() *SessionVariables {
	return o.vars
}

func (o *Session) RunCommand(ctx context.Context, cmd commands.Command) error {
	// TODO: Implement a RunCommandWithIO method to customize IO.
	return cmd.Run(ctx, o.io, o)
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

type CommandContext struct {
	seekAddr uint64
}
