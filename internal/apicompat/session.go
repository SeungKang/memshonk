package apicompat

import (
	"context"
	"io"

	"github.com/SeungKang/memshonk/internal/vendored/goterm"
)

type Session interface {
	SharedState() SharedState

	ID() string

	IO() SessionIO

	Terminal() (goterm.TerminalWithNotifications, bool)

	RunCommand(context.Context, Command) error
}

type SessionIO struct {
	Stdin io.Reader

	Stdout io.Writer

	Stderr io.Writer

	OptTerminal goterm.TerminalWithNotifications
}
