package commands

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
)

const (
	ShonksetCommandName = "shonkset"
)

func NewShonksetCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &ShonksetCommand{
		session: config.Session,
	}

	root := fx.NewCommand(ShonksetCommandName, "set configuration options", cmd.run)

	root.OptLongDesc = `CONFIGURATION ITEMS
  memmode  - The memory reading mode to use when accessing process memory.
             Omit the value to display the current mode.

             Supported values:
               kernel32 - Windows only
               procfs   - Linux only
               ptrace   - Any Unix-like OS

EXAMPLES
  $ shonkset memmode
  procfs

  $ shonkset memmode ptrace
`

	root.FlagSet.StringNf(&cmd.confItem, fx.ArgConfig{
		Name:        "configuration-item",
		Description: "",
		Required:    true,
	})

	root.FlagSet.StringNf(&cmd.confValue, fx.ArgConfig{
		Name:        "configuration-value",
		Description: "",
	})

	return root
}

type ShonksetCommand struct {
	session   apicompat.Session
	confItem  string
	confValue string
}

func (o *ShonksetCommand) run(ctx context.Context) (fx.CommandResult, error) {
	switch o.confItem {
	case "memmode":
		if o.confValue == "" {
			return fx.NewHumanCommandResult(o.session.SharedState().Progctl.MemoryMode()), nil
		}
		err := o.session.SharedState().Progctl.SetMemoryMode(o.confValue)
		return nil, err
	default:
		return nil, fmt.Errorf("invalid configuration item name %q", o.confItem)
	}
}
