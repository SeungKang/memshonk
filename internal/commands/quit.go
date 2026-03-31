package commands

import (
	"context"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
)

const (
	// Note: The "exit" shell builtin prevents us from using "exit"
	// because the interpreter would need to know when "exit" was
	// called from a memshonk shell versus a shell script.
	QuitCommandName = "quit"
)

func NewQuitCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := QuitCommand{
		session: config.Session,
	}

	root := fx.NewCommand(QuitCommandName, "exit the current memshonk session", cmd.run)

	return root
}

type QuitCommand struct {
	session apicompat.Session
}

func (o *QuitCommand) run(context.Context) (fx.CommandResult, error) {
	go o.session.Close()

	return nil, nil
}
