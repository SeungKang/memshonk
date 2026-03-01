package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
)

const (
	SessionsCommandName = "sessions"
)

func NewSessionsCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &SessionsCommand{
		session: config.Session,
	}

	root := fx.NewCommand(SessionsCommandName, "manage sessions", cmd.info)

	info := root.AddSubcommand("info", "show session info", cmd.info)
	root.AddSubcommand("ls", "list sessions", cmd.ls)
	rm := root.AddSubcommand("rm", "remove sessions", cmd.rm)

	// register session-ids on info and rm (not ls)
	for _, c := range []*fx.Command{info, rm} {
		c.FlagSet.StringSliceNf(&cmd.sessionIDs, fx.ArgConfig{
			Name:        "session-ids",
			Description: "one or more session IDs to operate on",
		})
	}

	return root
}

type SessionsCommand struct {
	session    apicompat.Session
	sessionIDs []string
}

func (o *SessionsCommand) info(_ context.Context) (fx.CommandResult, error) {
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

func (o *SessionsCommand) ls(_ context.Context) (fx.CommandResult, error) {
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

func (o *SessionsCommand) rm(_ context.Context) (fx.CommandResult, error) {
	for _, id := range o.sessionIDs {
		o.session.SharedState().Sessions.RemoveSession(id)
	}

	return nil, nil
}
