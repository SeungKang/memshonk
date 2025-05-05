package commands

import "context"

type Command interface {
	Run(ctx context.Context, inputOutput IO, s Session) error
}

type Session interface {
	Process() Process
}

type Process interface {
	Attach(ctx context.Context) (int, error)
}

type IO struct {
}
