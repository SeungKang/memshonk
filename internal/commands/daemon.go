package commands

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
)

const (
	DaemonCommandName = "daemon"
)

func NewDaemonCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &DaemonCommand{
		config: config,
	}

	root := fx.NewCommand(DaemonCommandName, "manage the server daemon", nil)
	kill := root.AddSubcommand("kill", "kill the server daemon", cmd.kill)

	kill.FlagSet.BoolFlag(&cmd.force, false, fx.ArgConfig{
		Name:        "force",
		Description: "Force kill the server daemon and other open sessions",
	})

	return root
}

type DaemonCommand struct {
	config apicompat.NewCommandConfig
	force  bool
}

func (o *DaemonCommand) kill(_ context.Context) (fx.CommandResult, error) {
	sessions := o.config.Session.SharedState().Sessions.Sessions()

	if len(sessions) > 1 && !o.force {
		return nil, fmt.Errorf("there are %d other active session(s), use -f to force kill the daemon and all active sessions",
			len(sessions)-1)
	}

	_ = o.config.Session.SharedState().Sessions.Close()

	return nil, nil
}
