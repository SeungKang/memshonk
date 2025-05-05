package app

import (
	"context"
	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/SeungKang/memshonk/internal/progctl"
	"sync"
)

func NewApp(project *Project) *App {
	return &App{
		project: project,
		process: nil, // TODO
	}
}

type App struct {
	rwMu          sync.RWMutex
	project       *Project
	nextSessionId uint64
	sessions      map[uint64]*Session
	process       *progctl.Routine
}

func (o *App) NewSession() *Session {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.sessions == nil {
		o.sessions = make(map[uint64]*Session)
	}

	id := o.nextSessionId
	o.nextSessionId++

	session := newSession(id, o.project, o.process)
	o.sessions[id] = session

	return session
}

type Project struct {
}

func newSession(id uint64, project *Project, process *progctl.Routine) *Session {
	return &Session{
		id:      id,
		project: project,
		process: process,
	}
}

type Session struct {
	id      uint64
	cmdCtx  *CommandContext
	project *Project
	process *progctl.Routine
}

func (o *Session) RunCommand(ctx context.Context, cmd commands.Command) error {
	return cmd.Run(ctx, commands.IO{}, o)
}

func (o *Session) Process() commands.Process {
	return o.process
}

type CommandContext struct {
	seekAddr uint64
}
