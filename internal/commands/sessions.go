package commands

import (
	"context"
	"fmt"
	"strings"

	"github.com/SeungKang/memshonk/internal/apicompat"
)

const (
	sessionsCommandName = "sessions"
)

func SessionsCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      sessionsCommandName,
		ShortHelp: "manage sessions",
		NonFlags: []NonFlagSchema{
			{
				Name:     "command",
				Desc:     "the sessions command ('info', 'ls', 'rm')",
				DefValue: "info",
				DataType: "",
			},
			{
				Name:     "session-ids",
				Desc:     "one or more session IDs to operate on",
				DataType: []string{},
				DefValue: nil,
			},
		},
		CreateFn: func(c CommandConfig) (apicompat.Command, error) {
			return NewSessionsCommand(SessionsCommandArgs{
				Mode:       c.NonFlags.String("command"),
				SessionIDs: c.NonFlags.StringList("session-ids"),
			}), nil
		},
	}
}

type SessionsCommandArgs struct {
	Mode       string
	SessionIDs []string
}

func NewSessionsCommand(args SessionsCommandArgs) SessionsCommand {
	return SessionsCommand{
		args: args,
	}
}

type SessionsCommand struct {
	args SessionsCommandArgs
}

func (o SessionsCommand) Name() string {
	return sessionsCommandName
}

func (o SessionsCommand) Run(ctx context.Context, s apicompat.Session) (apicompat.CommandResult, error) {
	switch o.args.Mode {
	case "info":
		return o.info(s)
	case "ls":
		return o.ls(s)
	case "rm":
		return nil, o.rm(s)
	default:
		return nil, fmt.Errorf("unknown sessions command; %q", o.args.Mode)
	}
}

func (o SessionsCommand) info(s apicompat.Session) (apicompat.CommandResult, error) {
	if len(o.args.SessionIDs) == 0 {
		return HumanCommandResult(s.Info().String()), nil
	}

	var b strings.Builder

	for i, id := range o.args.SessionIDs {
		session, ok := s.SharedState().Sessions.GetSession(id)
		if !ok {
			return nil, fmt.Errorf("session not found: %q", id)
		}

		if i > 0 {
			b.WriteString("\n")
		}

		b.WriteString(session.Info().String())
	}

	return HumanCommandResult(b.String()), nil
}

func (o SessionsCommand) ls(s apicompat.Session) (apicompat.CommandResult, error) {
	sessions := s.SharedState().Sessions.Sessions()

	var b strings.Builder

	for i, session := range sessions {
		if i > 0 {
			b.WriteString("\n")
		}

		b.WriteString(session.Info().String())
	}

	return HumanCommandResult(b.String()), nil
}

func (o SessionsCommand) rm(s apicompat.Session) error {
	for _, id := range o.args.SessionIDs {
		s.SharedState().Sessions.RemoveSession(id)
	}

	return nil
}
