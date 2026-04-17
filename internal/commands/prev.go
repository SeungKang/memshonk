package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
)

const (
	PrevCommandName = "prev"
)

func NewPrevCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &PrevCommand{
		storage: config.Session.CommandStorage(),
	}

	root := fx.NewCommand(PrevCommandName, "list and retrieve outputs of previously-run commands", cmd.run)

	root.FlagSet.StringNf(&cmd.target, fx.ArgConfig{
		Name:        "command-name",
		Description: "The command name",
	})

	root.FlagSet.Uint64Nf(&cmd.index, fx.ArgConfig{
		Name:        "index",
		Description: "The output index for the command",
	})

	return root
}

type PrevCommand struct {
	storage *apicompat.CommandStorage
	target  string
	index   uint64
}

func (o *PrevCommand) run(ctx context.Context) (fx.CommandResult, error) {
	if o.target == "" {
		available := o.storage.Available()
		if len(available) == 0 {
			return nil, nil
		}

		return fx.NewHumanCommandResult(strings.Join(available, "\n")), nil
	}

	output, err := o.storage.OutputFor(o.target, o.index)
	if err != nil {
		return nil, fmt.Errorf("failed to get output for %q - %w",
			o.target, err)
	}

	return output.Result, nil
}
