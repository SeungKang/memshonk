package app

import (
	"context"
	"sync"

	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/SeungKang/memshonk/internal/progctl"
	"github.com/SeungKang/memshonk/internal/project"
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
		io:  cmdIO,
	}
}

type Session struct {
	id     uint64
	app    *App
	cmdCtx *CommandContext
	io     commands.IO
}

func (o *Session) RunCommand(ctx context.Context, cmd commands.Command) error {
	// TODO: Implement a RunCommandWithIO method to customize IO.
	return cmd.Run(ctx, o.io, o)
}

func (o *Session) Process() progctl.Process {
	return o.app.ProcCtl()
}

type CommandContext struct {
	seekAddr uint64
}
