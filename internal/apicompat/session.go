package apicompat

import (
	"fmt"
	"io"
	"time"

	"github.com/SeungKang/memshonk/internal/jobsctl"
	"github.com/SeungKang/memshonk/internal/vendored/goterm"
)

// Session represents a client connection and the various parts of
// the application that it has access to.
type Session interface {
	// Close ends the current session.
	Close() error

	// SharedState is the application's shared state that is
	// made available to clients.
	SharedState() SharedState

	// Info returns information describing the session.
	Info() SessionInfo

	// IO returns the session's input-output.
	IO() SessionIO

	// Jobs returns an object that tracks the state of jobs
	// started by this session.
	Jobs() *jobsctl.Ctl

	// Terminal returns a non-nil terminal object and true
	// if the client supports a terminal, otherwise it
	// returns nil and false.
	Terminal() (*goterm.VirtualTerminal, bool)

	// CommandStorage returns information about the session's
	// previously-run commands.
	CommandStorage() *CommandStorage
}

// SessionInfo contains information about the session.
type SessionInfo struct {
	// ID is the session's ID.
	ID string

	// StartedAt is the time that the session was started at.
	StartedAt time.Time
}

func (o SessionInfo) String() string {
	return fmt.Sprintf("Session ID: %s Started at: %s", o.ID, o.StartedAt.Format(time.DateTime))
}

// SessionIO is the session's input-output.
type SessionIO struct {
	// Stdin is the client's standard input.
	Stdin io.Reader

	// Stdout is the client's standard output.
	Stdout io.Writer

	// Stderr is the client's standard error.
	Stderr io.Writer

	// OptTerminal is the client's terminal if
	// it allocated one. This field is nil if
	// no terminal has been allocated.
	OptTerminal *goterm.VirtualTerminal
}
