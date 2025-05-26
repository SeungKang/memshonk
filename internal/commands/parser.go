package commands

import (
	"context"

	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/plugins"
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

func (o ParserCommand) Run(ctx context.Context, inOut IO, s Session) (CommandResult, error) {
	loadedPlugins, enabled := s.Plugins()
	if !enabled {
		return nil, plugins.ErrPluginsDisabled
	}

	plugin, err := loadedPlugins.Plugin(o.args.PluginName)
	if err != nil {
		return nil, err
	}

	addr, err := memory.CreatePointerFromString(o.args.Addr)
	if err != nil {
		return nil, err
	}

	blob, err := plugin.RunParser(o.args.ParserName, addr.Addrs[0])
	if err != nil {
		return nil, err
	}

	return HumanCommandResult(string(blob)), nil
}
