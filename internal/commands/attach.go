package commands

import (
	"context"
	"fmt"
)

var _ Command = (*AttachCommand)(nil)

type AttachCommandArgs struct {
	OptPid  int
	OptName string
}

func NewAttachCommand(args AttachCommandArgs) AttachCommand {
	return AttachCommand{
		args: args,
	}
}

type AttachCommand struct {
	args AttachCommandArgs
}

func (o AttachCommand) Run(ctx context.Context, inOut IO, s Session) error {
	// TODO: Support AttachCommandArgs
	pid, err := s.Process().Attach(ctx)
	if err != nil {
		return err
	}

	fmt.Fprintf(inOut.Stdout, "attached to pid: %d", pid)

	return nil
}
