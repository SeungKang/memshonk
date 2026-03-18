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

	root := fx.NewCommand(HelpCommandName, "list available commands and help topics", cmd.run)

	root.FlagSet.StringNf(&cmd.optTopic, fx.ArgConfig{
		Name:        "command-or-topic",
		Description: "Optionally display help for a specific command or topic",
	})

	return root
}

type HelpCommand struct {
	session  apicompat.Session
	optTopic string
}

func (o *HelpCommand) run(_ context.Context) (fx.CommandResult, error) {
	var sb strings.Builder

	if o.optTopic == "" {
		// Note: It is important that we render the
		// command list in this method instead of
		// making it part of the command's usage
		// because the PrintUsage will trigger
		// infinite recursive calls to the PrintUsage
		// method... and yeah, no bueno.
		sb.WriteString(`OVERVIEW
  memshonk is like Wireshark, but for process memory. It provides
  an interactive shell that supports POSIX shell syntax, pipes,
  job control, and execution of external programs and internal
  memshonk commands.

TOPICS
  pattern - Pattern string format used in the "find" command and
            potentially other commands

COMMANDS
`)

		cmds := o.session.SharedState().Commands.AsSlice(o.session)

		longest := 0

		for _, cmd := range cmds {
			cmdNameLen := len(cmd.Name())

			if cmdNameLen > longest {
				longest = cmdNameLen
			}
		}

		for i, cmd := range cmds {
			name := cmd.Name()
			nameLen := len(name)

			var sep string

			switch {
			case nameLen == longest:
				sep = " - "
			case nameLen > longest:
				sep = strings.Repeat(" ", nameLen-longest) + " - "
			case longest > nameLen:
				sep = strings.Repeat(" ", longest-nameLen) + " - "
			}

			sb.WriteString("  " + name + sep + cmd.Description)

			if len(cmds) > 1 && i != len(cmds)-1 {
				sb.WriteByte('\n')
			}
		}

		return fx.NewHumanCommandResult(sb.String()), nil
	}

	switch o.optTopic {
	case "pattern":
		return fx.NewHumanCommandResult(`PATTERN STRINGS
  memshonk supports a pattern string format for searching for byte sequences.
  This format is heavily inspired by video game modding tools which employ
  a similar pattern format.

  Users can specify a pattern as a hexadecimal string optionall separated
  by space charcaters. For example, to match four "A" characters, the
  string would be:
    41 41 41 41

  ... or:
    41414141

  Wildcard values can be specified using "??". For example, to match one
  "A" followed by two bytes of any value and then a "B" the string would
  look like this:
    41 ?? ?? 42`), nil
	default:
		cmdFn, found := o.session.SharedState().Commands.Lookup(o.optTopic)
		if !found {
			return nil, fmt.Errorf("unknown help topic or command: %q", o.optTopic)
		}

		cmd := cmdFn(apicompat.NewCommandConfig{Session: o.session})
		cmd.FlagSet.Actual().SetOutput(&sb)
		cmd.PrintUsage()

		return fx.NewHumanCommandResult(strings.TrimRight(sb.String(), "\n")), nil
	}
}
