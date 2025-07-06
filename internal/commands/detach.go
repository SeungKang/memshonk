package commands

import (
	"context"
)

const (
	detachCommandName = "detach"
)

func DetachCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      detachCommandName,
		Aliases:   []string{"d"},
		ShortHelp: "detach from the process",
		CreateFn: func(c CommandConfig) (Command, error) {
			return &DetachCommand{args: DetachCommandArgs{}}, nil
		},
	}
}

type DetachCommandArgs struct{}

type DetachCommand struct {
	args DetachCommandArgs
}

func (o DetachCommand) Name() string {
	return detachCommandName
}

func (o DetachCommand) Run(ctx context.Context, inOut IO, s Session) (CommandResult, error) {
	err := s.Process().Detach(ctx)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
