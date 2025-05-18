package grsh

import (
	"context"

	"github.com/SeungKang/memshonk/internal/app"
	"github.com/SeungKang/memshonk/internal/commands"
	"github.com/desertbit/grumble"
)

func NewPluginsCommand(session *app.Session) *grumble.Command {
	return &grumble.Command{
		Name: "plugins",
		Help: "manage plugins",
		Args: func(a *grumble.Args) {
			a.String("command", "the plugin command ('list', 'ls', "+
				"'load', 'unload', 'reload')", grumble.Default("list"))
			a.String("name", "the plugin name to operate on", grumble.Default(""))
		},
		Run: func(c *grumble.Context) error {
			err := session.RunCommand(
				context.Background(),
				commands.NewPluginsCommand(commands.PluginsCommandArgs{
					Mode:                 c.Args.String("command"),
					PluginNameOrFilePath: c.Args.String("name"),
				}))
			if err != nil {
				return err
			}

			return nil
		},
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
