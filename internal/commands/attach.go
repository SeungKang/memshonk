package commands

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/memory"
)

const (
	attachCommandName = "attach"
)

func AttachCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      attachCommandName,
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

func (o AttachCommand) Name() string {
	return attachCommandName
}

func (o AttachCommand) Run(ctx context.Context, inOut IO, s Session) (CommandResult, error) {
	// TODO: Support AttachCommandArgs
	pid, err := s.Process().Attach(ctx)
	if err != nil {
		return nil, err
	}

	eventPub := events.NewPublisher[AttachEvent](s.Events())
	done := make(chan struct{})
	_ = eventPub.Send(ctx, AttachEvent{Pid: pid, Done: done})
	<-done

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
	ExeObj memory.Object
}

func (o AttachCommandResult) Serialize() []byte {
	return []byte(fmt.Sprintf("attached to %q, pid: %d, base addr: 0x%x",
		o.ExeObj.Name,
		o.PID,
		o.ExeObj.BaseAddr))
}

type AttachEvent struct {
	Pid  int
	Done chan struct{}
}
