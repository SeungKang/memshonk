package apicompat

import (
	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/progctl"
	"github.com/SeungKang/memshonk/internal/project"
)

type SharedState struct {
	Sessions SessionManager

	Events *events.Groups

	Project *project.Project

	Progctl *progctl.Ctl

	Plugins plugins.Ctl
}

func (o SharedState) HasPlugins() (plugins.Ctl, bool) {
	if o.Plugins == nil {
		return nil, false
	}

	return o.Plugins, true
}

type SessionManager interface {
	Sessions() []Session

	GetSession(id string) (Session, bool)

	RemoveSession(id string)
}
