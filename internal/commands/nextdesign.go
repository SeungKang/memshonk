package commands

import (
	"context"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/plugins"
)

func NewPluginsCommandX(config apicompat.NewCommandConfig) *fx.Command {
	pluginsCtl, pluginsEnabled := config.Session.SharedState().HasPlugins()

	pluginsCmd := PluginsCommandX{
		Ctl: pluginsCtl,
	}

	root := fx.NewCommand(pluginsCommandName, "manage plugins", pluginsCmd.list)

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

type PluginsCommandX struct {
	PluginNameOrFilePath string
	Ctl                  plugins.Ctl
}

func (o PluginsCommandX) list(_ context.Context) (fx.CommandResult, error) {
	if o.PluginNameOrFilePath != "" {
		plugin, err := o.Ctl.Plugin(o.PluginNameOrFilePath)
		if err != nil {
			return nil, err
		}

		return fx.NewHumanCommandResult(plugin.PrettyString("")), nil
	}

	return fx.NewHumanCommandResult(o.Ctl.PrettyString("")), nil
}

func (o PluginsCommandX) load(_ context.Context) (fx.CommandResult, error) {
	plugin, err := o.Ctl.Load(plugins.PluginConfig{
		FilePath: o.PluginNameOrFilePath,
	})
	if err != nil {
		return nil, err
	}

	return fx.NewHumanCommandResult(plugin.PrettyString("")), nil
}

func (o PluginsCommandX) reload(ctx context.Context) (fx.CommandResult, error) {
	err := o.Ctl.Reload(ctx, o.PluginNameOrFilePath)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func (o PluginsCommandX) unload(_ context.Context) (fx.CommandResult, error) {
	err := o.Ctl.Unload(o.PluginNameOrFilePath)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func NewCommandFromPluginX(cmd plugins.Command, plugin plugins.Plugin, args []string) *CommandFromPlugin {
	return &CommandFromPlugin{
		cmd:        cmd,
		pluginName: plugin.Name(),
		cmdName:    cmd.Name(),
		args:       args,
	}
}

type CommandFromPluginX struct {
	cmd        plugins.Command
	pluginName string
	cmdName    string
	args       []string
}

func (o CommandFromPluginX) Name() string {
	return o.cmd.Name() + "::" + o.cmdName
}

func (o CommandFromPluginX) Run(ctx context.Context, s apicompat.Session) (apicompat.CommandResult, error) {
	output, err := o.cmd.Run(ctx, o.args)
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return nil, nil
	}

	return HumanCommandResult(output), nil
}

func NewParserFromPluginX(parser plugins.Parser, plugin plugins.Plugin, arg uintptr) *ParserFromPlugin {
	return &ParserFromPlugin{
		parser:     parser,
		pluginName: plugin.Name(),
		parserName: parser.Name(),
		arg:        arg,
	}
}

type ParserFromPluginX struct {
	parser     plugins.Parser
	pluginName string
	parserName string
	arg        uintptr
}

func (o ParserFromPluginX) Name() string {
	return o.parser.Name() + "::" + o.parserName
}

func (o ParserFromPluginX) Run(ctx context.Context, s apicompat.Session) (apicompat.CommandResult, error) {
	output, err := o.parser.Run(ctx, o.arg)
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return nil, nil
	}

	return HumanCommandResult(output), nil
}
