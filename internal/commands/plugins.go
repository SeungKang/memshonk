package commands

import (
	"context"
	"errors"
	"fmt"

	"github.com/SeungKang/memshonk/internal/plugins"
)

var _ Command = (*AttachCommand)(nil)

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

func (o PluginsCommand) unload(ctl plugins.Ctl, inOut IO) error {
	return errors.New("TODO: not implemented yet :(")
}

func (o PluginsCommand) reload(ctl plugins.Ctl, inOut IO) error {
	return errors.New("TODO: not implemented yet :(")
}
