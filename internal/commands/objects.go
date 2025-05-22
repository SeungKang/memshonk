package commands

import (
	"context"
	"fmt"
	"github.com/SeungKang/memshonk/internal/memory"
)

func ObjectsCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      "objects",
		Aliases:   []string{"o"},
		ShortHelp: "list the memory mapped objects",
		CreateFn: func(c CommandConfig) (Command, error) {
			return NewObjectsCommand(ObjectsCommandArgs{}), nil
		},
	}
}

type ObjectsCommandArgs struct {
}

func NewObjectsCommand(args ObjectsCommandArgs) ObjectsCommand {
	return ObjectsCommand{
		args: args,
	}
}

type ObjectsCommand struct {
	args ObjectsCommandArgs
}

func (o ObjectsCommand) Run(ctx context.Context, inOut IO, s Session) error {
	objects, err := s.Process().MappedObjects(ctx)
	if err != nil {
		return err
	}

	err = objects.IterObjects(func(object memory.MappedObject) error {
		fmt.Fprintln(inOut.Stdout, object.String())
		return nil
	})
	if err != nil {
		return err
	}

	return nil
}
