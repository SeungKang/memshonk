package apicompat

import (
	"context"
	"io"

	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/progctl"
	"github.com/SeungKang/memshonk/internal/vendored/goterm"
)

type Command interface {
	Name() string

	Run(context.Context, Session) (CommandResult, error)
}

type Session interface {
	IO() SessionIO

	Process() progctl.Process

	Plugins() (plugins.Ctl, bool)

	Events() *events.Groups

	Terminal() (goterm.TerminalWithNotifications, bool)
}

type SessionIO struct {
	Stdin io.Reader

	Stdout io.Writer

	Stderr io.Writer

	OptTerminal goterm.TerminalWithNotifications
}

type CommandResult interface {
	Serialize() []byte
}
