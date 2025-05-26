package commands

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/plugins"
)

func PluginsCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      "plugins",
		ShortHelp: "manage plugins",
		NonFlags: []NonFlagSchema{
			{
				Name: "command",
				Desc: "the plugin command ('list', 'ls', " +
					"'load', 'unload', 'reload')",
				DefValue: "list",
				DataType: "",
			},
			{
				Name:     "name",
				Desc:     "the plugin name to operate on",
				DataType: "",
				DefValue: "",
			},
		},
		CreateFn: func(c CommandConfig) (Command, error) {
			return NewPluginsCommand(PluginsCommandArgs{
				Mode:                 c.NonFlags.String("command"),
				PluginNameOrFilePath: c.NonFlags.String("name"),
			}), nil
		},
	}
}

type PluginsCommandArgs struct {
	Mode                 string
	PluginNameOrFilePath string
}

func NewPluginsCommand(args PluginsCommandArgs) PluginsCommand {
	return PluginsCommand{
		args: args,
	}
}

type PluginsCommand struct {
	args PluginsCommandArgs
}

func (o PluginsCommand) Run(ctx context.Context, inOut IO, s Session) (CommandResult, error) {
	pluginsCtl, enabled := s.Plugins()
	if !enabled {
		return nil, plugins.ErrPluginsDisabled
	}

	switch o.args.Mode {
	case "list", "ls":
		return o.list(pluginsCtl)
	case "load":
		return o.load(pluginsCtl)
	case "reload":
		return nil, o.reload(ctx, pluginsCtl)
	case "unload":
		return nil, o.unload(pluginsCtl, inOut)
	default:
		return nil, fmt.Errorf("unknown plugins command; %q",
			o.args.Mode)
	}
}

func (o PluginsCommand) list(ctl plugins.Ctl) (CommandResult, error) {
	if o.args.PluginNameOrFilePath != "" {
		plugin, err := ctl.Plugin(o.args.PluginNameOrFilePath)
		if err != nil {
			return nil, err
		}

		return HumanCommandResult(plugin.PrettyString("")), nil
	}

	return HumanCommandResult(ctl.PrettyString("")), nil
}

func (o PluginsCommand) load(ctl plugins.Ctl) (CommandResult, error) {
	plugin, err := ctl.Load(plugins.PluginConfig{
		FilePath: o.args.PluginNameOrFilePath,
	})
	if err != nil {
		return nil, err
	}

	return HumanCommandResult(plugin.PrettyString("")), nil
}

func (o PluginsCommand) reload(ctx context.Context, ctl plugins.Ctl) error {
	err := ctl.Reload(ctx, o.args.PluginNameOrFilePath)
	if err != nil {
		return err
	}

	return nil
}

func (o PluginsCommand) unload(ctl plugins.Ctl, inOut IO) error {
	err := ctl.Unload(o.args.PluginNameOrFilePath)
	if err != nil {
		return err
	}

	return nil
}
