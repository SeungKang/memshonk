package commands

import (
	"context"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/progctl"
)

const (
	DetachCommandName = "detach"
)

func NewDetachCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &DetachCommand{
		progctl: config.Session.SharedState().Progctl,
	}

	root := fx.NewCommand(DetachCommandName, "detach from the process", cmd.detach)

	return root
}

type DetachCommand struct {
	progctl *progctl.Ctl
}

func (o *DetachCommand) detach(ctx context.Context) (fx.CommandResult, error) {
	err := o.progctl.Detach(ctx)
	if err != nil {
		return nil, err
	}

	return nil, nil
}
