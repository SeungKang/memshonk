package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
)

const (
	HelpCommandName = "help"
)

func NewHelpCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &HelpCommand{
		session: config.Session,
	}

	root := fx.NewCommand(HelpCommandName, "list available commands", cmd.help)

	root.FlagSet.StringNf(&cmd.optCommand, fx.ArgConfig{
		Name:        "command",
		Description: "optionally display help for a specific command",
	})

	return root
}

type HelpCommand struct {
	session    apicompat.Session
	optCommand string
}

func (o *HelpCommand) help(_ context.Context) (fx.CommandResult, error) {
	var sb strings.Builder

	if o.optCommand == "" {
		cmds := o.session.SharedState().Commands.AsSlice(o.session)

		sb.WriteString("memshonk commands:\n")

		for _, cmd := range cmds {
			fmt.Fprintf(&sb, "  %-15s %s\n", cmd.Name(), cmd.Description)
		}

		sb.WriteString("\nshell commands and external programs are supported as well")

		return fx.NewHumanCommandResult(sb.String()), nil
	} else {
		cmdFn, found := o.session.SharedState().Commands.Lookup(o.optCommand)
		if !found {
			return nil, fmt.Errorf("command %q not found", o.optCommand)
		}

		cmd := cmdFn(apicompat.NewCommandConfig{Session: o.session})
		cmd.FlagSet.Actual().SetOutput(&sb)
		cmd.PrintUsage()

		return fx.NewHumanCommandResult(strings.TrimRight(sb.String(), "\n")), nil
	}
}
