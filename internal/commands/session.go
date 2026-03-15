package commands

import (
	"context"
	"flag"
	"fmt"
	"strings"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
)

const (
	SessionCommandName = "session"
)

func NewSessionCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &SessionCommand{
		session: config.Session,
	}

	root := fx.NewCommand(SessionCommandName, "manage session", nil)
	root.Fn = func(_ context.Context) (fx.CommandResult, error) {
		root.PrintUsage()
		return nil, flag.ErrHelp
	}

	info := root.AddSubcommand("info", "show session info", cmd.info)
	root.AddSubcommand("ls", "list sessions", cmd.ls)
	rm := root.AddSubcommand("rm", "remove session", cmd.rm)

	info.FlagSet.StringSliceNf(&cmd.sessionIDs, fx.ArgConfig{
		Name:        "session-ids",
		Description: "one or more session IDs to operate on",
	})

	rm.FlagSet.StringSliceNf(&cmd.sessionIDs, fx.ArgConfig{
		Name:        "session-ids",
		Description: "one or more session IDs to remove",
		Required:    true,
	})

	return root
}

type SessionCommand struct {
	session    apicompat.Session
	sessionIDs []string
}

func (o *SessionCommand) info(_ context.Context) (fx.CommandResult, error) {
	if len(o.sessionIDs) == 0 {
		return fx.NewHumanCommandResult(o.session.Info().String()), nil
	}

	var b strings.Builder

	for i, id := range o.sessionIDs {
		session, ok := o.session.SharedState().Sessions.GetSession(id)
		if !ok {
			return nil, fmt.Errorf("session not found: %q", id)
		}

		if i > 0 {
			b.WriteString("\n")
		}

		b.WriteString(session.Info().String())
	}

	return fx.NewHumanCommandResult(b.String()), nil
}

func (o *SessionCommand) ls(_ context.Context) (fx.CommandResult, error) {
	sessions := o.session.SharedState().Sessions.Sessions()

	var b strings.Builder

	for i, session := range sessions {
		if i > 0 {
			b.WriteString("\n")
		}

		b.WriteString(session.Info().String())
	}

	return fx.NewHumanCommandResult(b.String()), nil
}

func (o *SessionCommand) rm(_ context.Context) (fx.CommandResult, error) {
	for _, id := range o.sessionIDs {
		if _, ok := o.session.SharedState().Sessions.GetSession(id); !ok {
			return nil, fmt.Errorf("session not found: %q", id)
		}

		o.session.SharedState().Sessions.RemoveSession(id)
	}

	return nil, nil
}
