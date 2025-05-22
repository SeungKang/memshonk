package commands

import (
	"context"
	"fmt"
)

func AttachCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      "attach",
		Aliases:   []string{"a"},
		ShortHelp: "attach to the process",
		Flags: []FlagSchema{
			{
				Short:      "p",
				Long:       "pid",
				Desc:       "Optional: Specify the process' PID",
				DataType:   0,
				DefaultVal: 0,
			},
			{
				Short:      "n",
				Long:       "name",
				Desc:       "Optional: Specify the process' name",
				DataType:   "",
				DefaultVal: "",
			},
		},
		CreateFn: func(c CommandConfig) (Command, error) {
			return NewAttachCommand(AttachCommandArgs{
				OptPid:  c.Flags.Int("pid"),
				OptName: c.Flags.String("name"),
			}), nil
		},
	}
}

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

	fmt.Fprintf(inOut.Stdout, "attached to pid: %d\n", pid)

	return nil
}
