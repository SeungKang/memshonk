package commands

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/progctl"
)

const (
	AttachCommandName = "attach"
)

func NewAttachCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &AttachCommand{
		progctl: config.Session.SharedState().Progctl,
		stderr:  config.Stderr,
	}

	root := fx.NewCommand(AttachCommandName, "attach to the process", cmd.attach)

	root.FlagSet.IntFlag(&cmd.pid, 0, fx.ArgConfig{
		Name:        "pid",
		Description: "Attach to a process by its PID instead of executable name",
	})

	return root
}

type AttachCommand struct {
	progctl *progctl.Ctl
	stderr  io.Writer
	pid     int
}

func (o *AttachCommand) attach(ctx context.Context) (fx.CommandResult, error) {
	pid, err := o.progctl.Attach(ctx, progctl.AttachConfig{OptPID: o.pid})
	switch {
	case err == nil:
		// Keep going.
	case errors.Is(err, progctl.ErrAlreadyAttached):
		fmt.Fprintf(o.stderr, "note: already attached to process with pid: %d\n",
			pid)
	default:
		return nil, err
	}

	info, err := o.progctl.ExeInfo(ctx)
	if err != nil {
		return nil, err
	}

	return fx.NewSerialCommandResult(AttachCommandResult{
		PID:    pid,
		ExeObj: info.Obj,
	}), nil
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
