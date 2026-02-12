package apicompat

import (
	"context"
)

type Command interface {
	Name() string

	Run(context.Context, Session) (CommandResult, error)
}

type CommandResult interface {
	Serialize() []byte
}
