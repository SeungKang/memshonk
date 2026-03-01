package commands

import (
	"context"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/plugins"
)

const (
	PluginsCommandName = "plugins"
)

func NewPluginsCommand(config apicompat.NewCommandConfig) *fx.Command {
	pluginsCtl, pluginsEnabled := config.Session.SharedState().HasPlugins()

	pluginsCmd := PluginsCommand{
		Ctl: pluginsCtl,
	}

	root := fx.NewCommand(PluginsCommandName, "manage plugins", pluginsCmd.list)

	root.OptPreRunFn = func(context.Context) error {
		if pluginsEnabled {
			return nil
		}

		return plugins.ErrPluginsDisabled
	}

	ls := root.AddSubcommand("ls", "list loaded plugins", pluginsCmd.list)

	root.AddSubcommand("load", "load a plugin", pluginsCmd.load)

	root.AddSubcommand("reload", "reload a plugin", pluginsCmd.reload)

	root.AddSubcommand("unload", "unload a plugin", pluginsCmd.unload)

	root.VisitAll(func(c *fx.Command) {
		required := true

		if c.Name() == root.Name() || c.Name() == ls.Name() {
			required = false
		}

		c.FlagSet.StringNf(&pluginsCmd.PluginNameOrFilePath, fx.ArgConfig{
			Name:        "plugin-name-or-path",
			Description: "name of a plugin or its file path",
			Required:    required,
		})
	})

	return root
}

type PluginsCommand struct {
	PluginNameOrFilePath string
	Ctl                  plugins.Ctl
}

func (o PluginsCommand) list(_ context.Context) (fx.CommandResult, error) {
	if o.PluginNameOrFilePath != "" {
		plugin, err := o.Ctl.Plugin(o.PluginNameOrFilePath)
		if err != nil {
			return nil, err
		}

		return fx.NewHumanCommandResult(plugin.PrettyString("")), nil
	}

	return fx.NewHumanCommandResult(o.Ctl.PrettyString("")), nil
}

func (o PluginsCommand) load(_ context.Context) (fx.CommandResult, error) {
	plugin, err := o.Ctl.Load(plugins.PluginConfig{
		FilePath: o.PluginNameOrFilePath,
	})
	if err != nil {
		return nil, err
	}

	return fx.NewHumanCommandResult(plugin.PrettyString("")), nil
}

func (o PluginsCommand) reload(ctx context.Context) (fx.CommandResult, error) {
	err := o.Ctl.Reload(ctx, o.PluginNameOrFilePath)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (o PluginsCommand) unload(_ context.Context) (fx.CommandResult, error) {
	err := o.Ctl.Unload(o.PluginNameOrFilePath)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func NewCommandFromPlugin(cmd plugins.Command, plugin plugins.Plugin, args []string) *CommandFromPlugin {
	return &CommandFromPlugin{
		cmd:        cmd,
		pluginName: plugin.Name(),
		cmdName:    cmd.Name(),
		args:       args,
	}
}

type CommandFromPlugin struct {
	cmd        plugins.Command
	pluginName string
	cmdName    string
	args       []string
}

func (o CommandFromPlugin) Name() string {
	return o.cmd.Name() + "::" + o.cmdName
}

func (o CommandFromPlugin) Run(ctx context.Context, s apicompat.Session) (apicompat.CommandResult, error) {
	output, err := o.cmd.Run(ctx, o.args)
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return nil, nil
	}

	return HumanCommandResult(output), nil
}

func NewParserFromPlugin(parser plugins.Parser, plugin plugins.Plugin, arg uintptr) *ParserFromPlugin {
	return &ParserFromPlugin{
		parser:     parser,
		pluginName: plugin.Name(),
		parserName: parser.Name(),
		arg:        arg,
	}
}

type ParserFromPlugin struct {
	parser     plugins.Parser
	pluginName string
	parserName string
	arg        uintptr
}

func (o ParserFromPlugin) Name() string {
	return o.parser.Name() + "::" + o.parserName
}

func (o ParserFromPlugin) Run(ctx context.Context, s apicompat.Session) (apicompat.CommandResult, error) {
	output, err := o.parser.Run(ctx, o.arg)
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return nil, nil
	}

	return HumanCommandResult(output), nil
}
