package app

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/progctl"
	"github.com/SeungKang/memshonk/internal/project"
)

func NewApp(eventGroups *events.Groups, project *project.Project, progCtl progctl.Process, optPluginCtl plugins.Ctl) *App {
	return &App{
		events:    eventGroups,
		project:   project,
		procCtl:   progCtl,
		pluginCtl: optPluginCtl,
	}
}

type App struct {
	events    *events.Groups
	project   *project.Project
	procCtl   progctl.Process
	pluginCtl plugins.Ctl
	rwMu      sync.RWMutex
	randStr   *randomStringer
	sessions  map[string]*Session
}

func (o *App) Events() *events.Groups {
	return o.events
}

func (o *App) ProcCtl() progctl.Process {
	return o.procCtl
}

type SessionConfig struct {
	IO    SessionIO
	OptID string
}

func (o *App) NewSession(config SessionConfig) (*Session, error) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.sessions == nil {
		o.sessions = make(map[string]*Session)
	}

	var id string

	if config.OptID == "" {
		if o.randStr == nil {
			o.randStr = newRandomStringer()
		}

		for i := 0; i < 100; i++ {
			possibleId := o.randStr.String()

			_, hasIt := o.sessions[possibleId]
			if !hasIt {
				id = possibleId

				break
			}
		}

		if id == "" {
			var buf bytes.Buffer

			b := make([]byte, 4)

			_, err := rand.Read(b)
			if err != nil {
				panic(err)
			}

			_, err = hex.NewEncoder(&buf).Write(b)
			if err != nil {
				panic(err)
			}

			id = buf.String()
		}
	} else {
		_, hasIt := o.sessions[config.OptID]
		if hasIt {
			return nil, fmt.Errorf("session id already in use (%q)",
				config.OptID)
		}

		id = config.OptID
	}

	switch {
	case id == "":
		return nil, errors.New("session id string is empty")
	case strings.ContainsAny(id, "/\\"):
		return nil, errors.New("session id contains path separator charcter(s)")
	}

	session := newSession(id, o, config.IO)
	o.sessions[id] = session

	return session, nil
}

func (o *App) Project() *project.Project {
	return o.project
}

type CommandContext struct {
	seekAddr uint64
}
