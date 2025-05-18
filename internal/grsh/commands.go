package grsh

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/app"
	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/desertbit/grumble"
)

func commandSchemaToGrumbleCommand(cmdSchema commands.CommandSchema, session *app.Session) *grumble.Command {
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

func NewAttachCommand(session *app.Session) *grumble.Command {
	return &grumble.Command{
		Name:    "attach",
		Aliases: []string{"a"},
		Help:    "attach to the process",
		Flags: func(f *grumble.Flags) {
			f.Int("p", "pid", 0, "Optional: Specify the process' PID")
			f.String("n", "name", "", "Optional: Specify the process' name")
		},
		Run: func(c *grumble.Context) error {
			err := session.RunCommand(
				context.Background(),
				commands.NewAttachCommand(commands.AttachCommandArgs{
					OptPid:  c.Flags.Int("pid"),
					OptName: c.Flags.String("name"),
				}))
			if err != nil {
				return err
			}

			return nil
		},
	}
}

func NewFindCommand(session *app.Session) *grumble.Command {
	return &grumble.Command{
		Name:    "find",
		Aliases: []string{"f"},
		Help:    "find a pattern in memory",
		Flags: func(f *grumble.Flags) {
		},
		Args: func(a *grumble.Args) {
			a.String("pattern", "byte pattern to search for")
			a.String("start", "the address to start searching from")
		},
		Run: func(c *grumble.Context) error {
			err := session.RunCommand(
				context.Background(),
				commands.NewFindCommand(commands.FindCommandArgs{
					Pattern:   c.Args.String("pattern"),
					StartAddr: c.Args.String("start"),
				}))
			if err != nil {
				return err
			}

			return nil
		},
	}
}

//func NewSeekCommand(session *app.Session) *grumble.Command {
//	return &grumble.Command{
//		Name:    "seek",
//		Aliases: []string{"s"},
//		Help:    "set current address",
//		Args: func(a *grumble.Args) {
//			a.String("addr", "address to seek to")
//		},
//		Run: sh.seek,
//	}
//}

func NewReadCommand(session *app.Session) *grumble.Command {
	return &grumble.Command{
		Name:    "read",
		Aliases: []string{"r"},
		Help:    "read n bytes from addr",
		Flags: func(f *grumble.Flags) {
			f.String("e", "encoding", "hexdump", "Optional: Specify output encoding format")
		},
		Args: func(a *grumble.Args) {
			a.Uint64("size", "number of bytes to read")
			a.String("addr", "address to read from", grumble.Default(""))
		},
		Run: func(c *grumble.Context) error {
			// TODO: Document encoding formats
			err := session.RunCommand(
				context.Background(),
				commands.NewReadCommand(commands.ReadCommandArgs{
					EncodingFormat: c.Flags.String("encoding"),
					AddrStr:        c.Args.String("addr"),
					SizeBytes:      c.Args.Uint64("size"),
				}))
			if err != nil {
				return err
			}

			return nil
		},
	}
}

func NewWriteCommand(session *app.Session) *grumble.Command {
	return &grumble.Command{
		Name:    "write",
		Aliases: []string{"w"},
		Help:    "write value to addr",
		Flags: func(f *grumble.Flags) {
			f.String("e", "encoding", "hex", "Optional: Specify input encoding format")
		},
		Args: func(a *grumble.Args) {
			a.String("data", "data to write")
			a.String("addr", "address to write to", grumble.Default(""))
		},
		Run: func(c *grumble.Context) error {
			// TODO: Document encoding formats
			err := session.RunCommand(
				context.Background(),
				commands.NewWriteCommand(commands.WriteCommandArgs{
					DataStr:        c.Args.String("data"),
					EncodingFormat: c.Flags.String("encoding"),
					AddrStr:        c.Args.String("addr"),
				}))
			if err != nil {
				return err
			}

			return nil
		},
	}
}

func NewObjectsCommand(session *app.Session) *grumble.Command {
	return &grumble.Command{
		Name:    "objects",
		Aliases: []string{"o"},
		Help:    "list the memory mapped objects",
		Flags:   func(f *grumble.Flags) {},
		Args:    func(a *grumble.Args) {},
		Run: func(c *grumble.Context) error {
			err := session.RunCommand(
				context.Background(),
				commands.NewObjectsCommand(commands.ObjectsCommandArgs{}))
			if err != nil {
				return err
			}

			return nil
		},
	}
}
