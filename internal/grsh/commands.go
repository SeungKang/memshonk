package grsh

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/SeungKang/memshonk/internal/plugins"

	"github.com/desertbit/grumble"
)

func commandSchemaToGrumbleCommand(cmdSchema commands.CommandSchema, session apicompat.Session) *grumble.Command {
	grumbleCmd := &grumble.Command{
		Name:    cmdSchema.Name,
		Aliases: cmdSchema.Aliases,
		Help:    cmdSchema.ShortHelp,
	}

	if len(cmdSchema.NonFlags) > 0 {
		grumbleCmd.Args = func(args *grumble.Args) {
			for _, nf := range cmdSchema.NonFlags {
				nonFlagToGrumble(nf, args)
			}
		}
	}

	if len(cmdSchema.Flags) > 0 {
		grumbleCmd.Flags = func(flags *grumble.Flags) {
			for _, flag := range cmdSchema.Flags {
				flagToGrumble(flag, flags)
			}
		}
	}

	grumbleCmd.Run = func(grumbleCtx *grumble.Context) error {
		cmd, err := cmdSchema.CreateFn(commands.CommandConfig{
			Flags:    grumbleCtx.Flags,
			NonFlags: grumbleCtx.Args,
		})
		if err != nil {
			return fmt.Errorf("failed to create command - %w", err)
		}

		return session.RunCommand(context.TODO(), cmd)
	}

	return grumbleCmd
}

func nonFlagToGrumble(nonFlag commands.NonFlagSchema, gargs *grumble.Args) {
	var grumbleOptions []grumble.ArgOption

	var grumbleArgFn func(name string, help string, opts ...grumble.ArgOption)

	switch nonFlag.DataType.(type) {
	case string:
		grumbleArgFn = gargs.String
	case []string:
		grumbleArgFn = gargs.StringList
	case bool:
		grumbleArgFn = gargs.Bool
	case int:
		grumbleArgFn = gargs.Int
	case []int:
		grumbleArgFn = gargs.IntList
	case int64:
		grumbleArgFn = gargs.Int64
	case []int64:
		grumbleArgFn = gargs.Int64List
	case uint:
		grumbleArgFn = gargs.Uint
	case []uint:
		grumbleArgFn = gargs.UintList
	case uint64:
		grumbleArgFn = gargs.Uint64
	case []uint64:
		grumbleArgFn = gargs.Uint64List
	default:
		panic(fmt.Sprintf("TODO: unsupported non-flag type: %T",
			nonFlag.DataType))
	}

	if nonFlag.DefValue != nil {
		grumbleOptions = append(grumbleOptions, grumble.Default(nonFlag.DefValue))
	}

	if len(grumbleOptions) > 0 {
		grumbleArgFn(nonFlag.Name, nonFlag.Desc, grumbleOptions...)
	} else {
		grumbleArgFn(nonFlag.Name, nonFlag.Desc)
	}
}

func flagToGrumble(flag commands.FlagSchema, gflags *grumble.Flags) {
	switch flag.DataType.(type) {
	case bool:
		var def bool
		if flag.DefaultVal != nil {
			def = flag.DefaultVal.(bool)
		}

		gflags.Bool(flag.Short, flag.Long, def, flag.Desc)

	case string:
		var def string
		if flag.DefaultVal != nil {
			def = flag.DefaultVal.(string)
		}

		gflags.String(flag.Short, flag.Long, def, flag.Desc)
	case int:
		var def int
		if flag.DefaultVal != nil {
			def = flag.DefaultVal.(int)
		}

		gflags.Int(flag.Short, flag.Long, def, flag.Desc)
	case int64:
		var def int64
		if flag.DefaultVal != nil {
			def = flag.DefaultVal.(int64)
		}

		gflags.Int64(flag.Short, flag.Long, def, flag.Desc)
	case uint:
		var def uint
		if flag.DefaultVal != nil {
			def = flag.DefaultVal.(uint)
		}

		gflags.Uint(flag.Short, flag.Long, def, flag.Desc)
	case uint64:
		var def uint64
		if flag.DefaultVal != nil {
			def = flag.DefaultVal.(uint64)
		}

		gflags.Uint64(flag.Short, flag.Long, def, flag.Desc)
	default:
		panic(fmt.Sprintf("TODO: unsupported flag type: %T",
			flag.DataType))
	}
}

func newPluginCommand(plugin plugins.Plugin, session apicompat.Session) *grumble.Command {
	description := plugin.Description()
	if description == "" {
		description = "a custom plugin"
	}

	grumbleCommand := &grumble.Command{
		Name: plugin.Name(),
		Help: description,
	}

	_ = plugin.IterCommands(func(cmd plugins.Command) error {
		grumbleCommand.AddCommand(&grumble.Command{
			Name:           cmd.Name(),
			Help:           "TODO",
			HelpGroup:      "commands",
			SkipArgParsing: true,
			Run: func(c *grumble.Context) error {
				cmd := commands.NewCommandFromPlugin(cmd, plugin, c.UnparsedArgs)

				return session.RunCommand(context.TODO(), cmd)
			},
		})

		return nil
	})

	_ = plugin.IterParsers(func(parser plugins.Parser) error {
		grumbleCommand.AddCommand(&grumble.Command{
			Name:      parser.Name(),
			Help:      "TODO",
			HelpGroup: "parsers",
			Args: func(args *grumble.Args) {
				args.String("addr", "memory address of data to parse", grumble.Default(""))
			},
			Run: func(c *grumble.Context) error {
				var addr uint64
				addrStr := c.Args.String("addr")

				if addrStr != "" {
					var err error

					addrStr = strings.TrimPrefix(addrStr, "0x")

					addr, err = strconv.ParseUint(addrStr, 16, 64)
					if err != nil {
						return fmt.Errorf("failed to parse address %q - %w",
							addrStr, err)
					}
				}

				cmd := commands.NewParserFromPlugin(parser, plugin, uintptr(addr))

				return session.RunCommand(context.TODO(), cmd)
			},
		})

		return nil
	})

	return grumbleCommand
}
