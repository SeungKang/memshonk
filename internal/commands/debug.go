package commands

import (
	"context"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
)

const (
	DebugCommandName = "debug"
)

func NewDebugCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &DebugCommand{
		session: config.Session,
	}

	root := fx.NewCommand(DebugCommandName, "control the state of the debugged process", nil)

	root.AddSubcommand("pause", "pause the debugged process", cmd.pause)

	root.AddSubcommand("resume", "resume the debugged process", cmd.resume)

	return root
}

type DebugCommand struct {
	session apicompat.Session
}

func (o *DebugCommand) pause(ctx context.Context) (fx.CommandResult, error) {
	return nil, o.session.SharedState().Progctl.Suspend(ctx)
}

func (o *DebugCommand) resume(ctx context.Context) (fx.CommandResult, error) {
	return nil, o.session.SharedState().Progctl.Resume(ctx)
}
