package commands

import (
	"context"

	"github.com/SeungKang/memshonk/internal/events"
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

	eventPub := events.NewPublisher[DetachEvent](s.Events())
	done := make(chan struct{})
	_ = eventPub.Send(ctx, DetachEvent{Done: done})
	<-done

	return nil, nil
}

type DetachEvent struct {
	Done chan struct{}
}
