package app

import (
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

type CommandContext struct {
	seekAddr uint64
}
