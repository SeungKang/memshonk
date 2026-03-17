package apicompat

import (
	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/progctl"
	"github.com/SeungKang/memshonk/internal/project"
	"github.com/SeungKang/memshonk/internal/shvars"
)

// SharedState provides access to various application functionality.
type SharedState struct {
	Sessions SessionManager

	Events *events.Groups

	Vars *shvars.Variables

	Project *project.Project

	Progctl *progctl.Ctl

	Commands *CommandRegistry

	Plugins plugins.Ctl
}

func (o SharedState) HasPlugins() (plugins.Ctl, bool) {
	if o.Plugins == nil {
		return nil, false
	}

	return o.Plugins, true
}

// SessionManager manages sessions.
type SessionManager interface {
	// Sessions returns a slice of the current sessions.
	Sessions() []Session

	// GetSession returns a Session for the specified
	// session ID. If no such session exists, then
	// nil and false are returned.
	GetSession(id string) (Session, bool)

	// RemoveSession removes the specified session b
	// its ID.
	RemoveSession(id string)

	// Close disconnects all clients and exits
	// the SessionManager.
	Close() error
}
