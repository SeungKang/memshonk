package commands

import (
	"context"

	"github.com/SeungKang/memshonk/internal/apicompat"
)

const (
	detachCommandName = "detach"
)

func DetachCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      detachCommandName,
		Aliases:   []string{"d"},
		ShortHelp: "detach from the process",
		CreateFn: func(c CommandConfig) (apicompat.Command, error) {
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

func (o DetachCommand) Run(ctx context.Context, s apicompat.Session) (apicompat.CommandResult, error) {
	err := s.SharedState().Progctl.Detach(ctx)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
