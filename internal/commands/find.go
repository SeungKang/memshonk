package commands

import (
	"context"
	"fmt"
	"github.com/SeungKang/memshonk/internal/memory"
)

var _ Command = (*FindCommand)(nil)

type FindCommandArgs struct {
	Pattern   string
	StartAddr string
}

func NewFindCommand(args FindCommandArgs) FindCommand {
	return FindCommand{
		args: args,
	}
}

type FindCommand struct {
	args FindCommandArgs
}

func (o FindCommand) Run(ctx context.Context, inOut IO, s Session) error {
	addrStr := o.args.StartAddr

	var ptr memory.Pointer
	ptr, err := memory.CreatePointerFromString(addrStr)
	if err != nil {
		return err
	}

	reader, err := memory.NewBufferedReader(s.Process(), ptr, 4096)
	if err != nil {
		return err
	}

	matches, err := memory.FindAllReader(o.args.Pattern, reader)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(inOut.Stdout, "matches:", matches)
	if err != nil {
		return err
	}

	return nil
}
