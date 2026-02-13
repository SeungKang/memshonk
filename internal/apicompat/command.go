package apicompat

import (
	"context"
)

// Command represents a command that can be run by a client.
type Command interface {
	// Name is the name of the command.
	Name() string

	// Run executes the command.
	Run(context.Context, Session) (CommandResult, error)
}

type CommandResult interface {
	Serialize() []byte
}
