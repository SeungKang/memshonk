package commands

import (
	"context"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
)

const (
	DissectCommandName = "dissect"
)

func NewDissectCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &DissectCommand{
		config: config,
	}

	root := fx.NewCommand(DissectCommandName, "dissect an executable file", cmd.run)

	root.FlagSet.StringNf(&cmd.exePath, fx.ArgConfig{
		Name:        "exe-path",
		Description: "The executable to analyze",
		Required:    true,
	})

	return root
}

type DissectCommand struct {
	config  apicompat.NewCommandConfig
	exePath string
}

func (o *DissectCommand) run(ctx context.Context) (fx.CommandResult, error) {

}
