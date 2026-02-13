package apicompat

import (
	"context"
	"io"
	"time"

	"github.com/SeungKang/memshonk/internal/vendored/goterm"
)

type Session interface {
	SharedState() SharedState

	Info() SessionInfo

	IO() SessionIO

	Terminal() (goterm.TerminalWithNotifications, bool)

	RunCommand(context.Context, Command) error
}

type SessionInfo struct {
	ID string

	StartedAt time.Time
}

type SessionIO struct {
	Stdin io.Reader

	Stdout io.Writer

	Stderr io.Writer

	OptTerminal goterm.TerminalWithNotifications
}
