package commands

import (
	"context"
	"errors"
	"fmt"
	"github.com/SeungKang/memshonk/internal/memory"
)

func ParserCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      "parser",
		ShortHelp: "run parser plugins",
		NonFlags: []NonFlagSchema{
			{
				Name:     "parser-id",
				Desc:     "the parser id",
				DataType: "",
			},
			{
				Name:     "addr",
				Desc:     "the addr to parse",
				DataType: "",
			},
		},
		CreateFn: func(c CommandConfig) (Command, error) {
			return &ParserCommand{
				args: ParserCommandArgs{
					ParserID: c.NonFlags.String("parser-id"),
					Addr:     c.NonFlags.String("addr"),
				},
			}, nil
		},
	}
}

type ParserCommandArgs struct {
	ParserID string
	Addr     string
}

type ParserCommand struct {
	args ParserCommandArgs
}

func (o ParserCommand) Run(ctx context.Context, inOut IO, s Session) error {
	loadedPlugins, enabled := s.Plugins()
	if !enabled {
		return errors.New("plugins are disabled")
	}

	parser, err := loadedPlugins.Parser(o.args.ParserID)
	if err != nil {
		return err
	}

	addr, err := memory.CreatePointerFromString(o.args.Addr)
	if err != nil {
		return err
	}

	blob, err := parser.Run(addr.Addrs[0])
	if err != nil {
		return err
	}

	fmt.Fprintln(inOut.Stdout, string(blob))

	return nil
}
