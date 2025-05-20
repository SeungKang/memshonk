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
				Name:     "plugin-name",
				Desc:     "the plugin name",
				DataType: "",
			},
			{
				Name:     "parser-name",
				Desc:     "the parser name",
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
					PluginName: c.NonFlags.String("plugin-name"),
					ParserName: c.NonFlags.String("parser-name"),
					Addr:       c.NonFlags.String("addr"),
				},
			}, nil
		},
	}
}

type ParserCommandArgs struct {
	PluginName string
	ParserName string
	Addr       string
}

type ParserCommand struct {
	args ParserCommandArgs
}

func (o ParserCommand) Run(ctx context.Context, inOut IO, s Session) error {
	loadedPlugins, enabled := s.Plugins()
	if !enabled {
		return errors.New("plugins are disabled")
	}

	plugin, err := loadedPlugins.Plugin(o.args.PluginName)
	if err != nil {
		return err
	}

	addr, err := memory.CreatePointerFromString(o.args.Addr)
	if err != nil {
		return err
	}

	blob, err := plugin.RunParser(o.args.ParserName, addr.Addrs[0])
	if err != nil {
		return err
	}

	fmt.Fprintln(inOut.Stdout, string(blob))

	return nil
}
