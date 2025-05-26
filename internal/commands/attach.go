package commands

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/memory"
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

func (o AttachCommand) Run(ctx context.Context, inOut IO, s Session) (CommandResult, error) {
	// TODO: Support AttachCommandArgs
	pid, err := s.Process().Attach(ctx)
	if err != nil {
		return nil, err
	}

	obj, err := s.Process().ExeObject(ctx)
	if err != nil {
		return nil, err
	}

	return AttachCommandResult{
		PID:    pid,
		ExeObj: obj,
	}, nil
}

type AttachCommandResult struct {
	PID    int
	ExeObj memory.MappedObject
}

func (o AttachCommandResult) Serialize() []byte {
	return []byte(fmt.Sprintf("attached to %q, pid: %d, base addr: 0x%x",
		o.ExeObj.Filename,
		o.PID,
		o.ExeObj.BaseAddr))
}
