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
		Session: config.Session,
		Ctl:     pluginsCtl,
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
	Session              apicompat.Session
	PluginNameOrFilePath string
	Ctl                  plugins.Ctl
}

func (o *PluginsCommand) list(_ context.Context) (fx.CommandResult, error) {
	if o.PluginNameOrFilePath != "" {
		plugin, err := o.Ctl.Plugin(o.PluginNameOrFilePath)
		if err != nil {
			return nil, err
		}

		return fx.NewHumanCommandResult(plugin.PrettyString("")), nil
	}

	return fx.NewHumanCommandResult(o.Ctl.PrettyString("")), nil
}

func (o *PluginsCommand) load(_ context.Context) (fx.CommandResult, error) {
	plugin, err := o.Ctl.Load(plugins.PluginConfig{
		FilePath: o.PluginNameOrFilePath,
	})
	if err != nil {
		return nil, err
	}

	RegisterPlugin(plugin, o.Session.SharedState().Commands)

	return fx.NewHumanCommandResult(plugin.PrettyString("")), nil
}

func (o *PluginsCommand) reload(ctx context.Context) (fx.CommandResult, error) {
	o.Session.SharedState().Commands.Unregister(o.PluginNameOrFilePath)

	err := o.Ctl.Reload(ctx, o.PluginNameOrFilePath)
	if err != nil {
		return nil, err
	}

	plugin, err := o.Ctl.Plugin(o.PluginNameOrFilePath)
	if err != nil {
		return nil, err
	}

	RegisterPlugin(plugin, o.Session.SharedState().Commands)

	return nil, nil
}

func (o *PluginsCommand) unload(_ context.Context) (fx.CommandResult, error) {
	plugToUnload, err := o.Ctl.Plugin(o.PluginNameOrFilePath)
	if err != nil {
		return nil, err
	}

	o.Session.SharedState().Commands.Unregister(plugToUnload.Name())

	err = o.Ctl.Unload(o.PluginNameOrFilePath)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func RegisterPlugin(plugin plugins.Plugin, registry *apicompat.CommandRegistry) {
	description := plugin.Description()
	if description == "" {
		description = plugin.Name() + " plugin"
	}

	newCommandFn := func(config apicompat.NewCommandConfig) *fx.Command {
		root := fx.NewCommand(plugin.Name(), description, nil)

		plugin.IterCommands(func(pluginCmd plugins.Command) error {
			wrapper := &pluginCommandWrapper{
				pluginCmd: pluginCmd,
			}

			sub := root.AddSubcommand(pluginCmd.Name(), "a custom command", nil)

			sub.CustomFn = wrapper.run

			return nil
		})

		parsersSubcommand := fx.NewCommand("parsers", "Run plugin parsers", nil)

		plugin.IterParsers(func(parser plugins.Parser) error {
			wrapper := &pluginParserWrapper{
				pluginParser: parser,
			}

			parserCmd := parsersSubcommand.AddSubcommand(parser.Name(), parser.Name()+" parser", wrapper.run)

			parserCmd.FlagSet.Uint64Nf(&wrapper.addr, fx.ArgConfig{
				Name:        "address",
				Description: "Address of data to parse",
			})

			return nil
		})

		if len(parsersSubcommand.Subcommands) > 0 {
			root.AddSubcommandCustom(parsersSubcommand)
		}

		return root
	}

	registry.Register(plugin.Name(), newCommandFn)
}

type pluginCommandWrapper struct {
	pluginCmd plugins.Command
}

func (o *pluginCommandWrapper) run(ctx context.Context, config fx.RunCommandConfig) (fx.CommandResult, error) {
	output, err := o.pluginCmd.Run(ctx, config.Args)
	if err != nil {
		return nil, err
	}

	if len(output) == 0 {
		return nil, nil
	}

	return fx.NewHumanCommandResult(string(output)), nil
}

type pluginParserWrapper struct {
	pluginParser plugins.Parser
	addr         uint64
}

func (o *pluginParserWrapper) run(ctx context.Context) (fx.CommandResult, error) {
	output, err := o.pluginParser.Run(ctx, uintptr(o.addr))
	if err != nil {
		return nil, err
	}

	return fx.NewHumanCommandResult(string(output)), nil
}
