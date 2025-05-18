package commands

import (
	"context"
	"errors"
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

func (o PluginsCommand) Run(ctx context.Context, inOut IO, s Session) error {
	plugins, enabled := s.Plugins()
	if !enabled {
		return errors.New("plugins are disabled")
	}

	switch o.args.Mode {
	case "list", "ls":
		return o.list(plugins, inOut)
	case "load":
		return o.load(plugins, inOut)
	case "reload":
		return o.reload(plugins, inOut)
	case "unload":
		return o.unload(plugins, inOut)
	default:
		return fmt.Errorf("unknown plugins command; %q",
			o.args.Mode)
	}
}

func (o PluginsCommand) list(ctl plugins.Ctl, inOut IO) error {
	if o.args.PluginNameOrFilePath != "" {
		plugin, hasIt := ctl.Plugin(o.args.PluginNameOrFilePath)
		if !hasIt {
			return errors.New("plugin is not loaded")
		}

		fmt.Fprintln(inOut.Stdout, plugin.PrettyString(""))

		return nil
	}

	fmt.Fprintln(inOut.Stdout, ctl.PrettyString(""))

	return nil
}

func (o PluginsCommand) load(ctl plugins.Ctl, inOut IO) error {
	plugin, err := ctl.Load(o.args.PluginNameOrFilePath)
	if err != nil {
		return fmt.Errorf("failed to load plugin - %w", err)
	}

	fmt.Fprintln(inOut.Stdout, plugin.PrettyString(""))

	return nil
}

func (o PluginsCommand) reload(ctl plugins.Ctl, inOut IO) error {
	return errors.New("TODO: not implemented yet :(")
}

func (o PluginsCommand) unload(ctl plugins.Ctl, inOut IO) error {
	return errors.New("TODO: not implemented yet :(")
}
