package fx

import (
	"context"
	"flag"
	"fmt"
	"io"
)

func NewCommand(name string, description string, usageOut io.Writer, fn func(context.Context) (CommandResult, error)) *Command {
	set := NewFlagSet(name)

	set.Actual().SetOutput(usageOut)

	set.internal.Usage = func() {}

	cmd := &Command{
		FlagSet:     set,
		Description: description,
		Fn:          fn,
	}

	// I tried using BoolFuncFlag and propogating flag.ErrHelp,
	// but that creates a new error object using FlagSet.failf
	// which makes it impossible to check if flag.ErrHelp is
	// present using errors.Is.
	set.BoolFlag(&cmd.help, false, ArgConfig{
		Name:        "help",
		Description: "Display this information",
	})

	return cmd
}

type Command struct {
	FlagSet     *FlagSet
	Description string
	Subcommands []*Command
	Fn          func(context.Context) (CommandResult, error)
	OptParent   *Command
	OptPreRunFn func(context.Context) error

	help bool
}

func (o *Command) Name() string {
	return o.FlagSet.Actual().Name()
}

func (o *Command) PrintUsage() {
	o.FlagSet.internal.Output().Write([]byte(`SYNOPSIS
` + o.synopsis("  ") + `

DESCRIPTION
  ` + o.Description + `

OPTIONS
`))

	_ = LongArgsUsage(o.FlagSet, 80)
}

func (o *Command) synopsis(indent string) string {
	var names string
	current := o

	for current != nil {
		if names == "" {
			names = current.Name()
		} else {
			names = current.Name() + " " + names
		}

		current = current.OptParent
	}

	names = indent + names

	str := names + " -h"

	var hasOptions bool

	var flags string

	var nonFlags string

	o.FlagSet.VisitAll(func(info ArgInfo) {
		if info.IsFlag {
			hasOptions = true

			if info.Config.Required && len(info.Config.Name) == 1 {
				if flags != "" {
					flags += " "
				}

				flags += "-" + info.Config.Name + " " + flagDataType(info.OptFlag)
			}
		} else {
			if nonFlags != "" {
				nonFlags += " "
			}

			if info.Config.Required {
				nonFlags += info.Config.Name
			} else {
				nonFlags += "[" + info.Config.Name + "]"
			}
		}
	})

	if hasOptions || flags != "" || nonFlags != "" {
		str += "\n" + names
	}

	if hasOptions {
		str += " [options]"
	}

	if flags != "" {
		str += " " + flags
	}

	if nonFlags != "" {
		str += " " + nonFlags
	}

	for _, sub := range o.Subcommands {
		str += "\n\n" + sub.synopsis(indent)
	}

	return str
}

func (o *Command) AddSubcommand(name string, description string, fn func(context.Context) (CommandResult, error)) *Command {
	return o.AddSubcommandCustom(NewCommand(name, description, o.FlagSet.Actual().Output(), fn))
}

func (o *Command) AddSubcommandCustom(cmd *Command) *Command {
	cmd.OptParent = o

	if cmd.FlagSet.Actual().Output() != o.FlagSet.Actual().Output() {
		cmd.FlagSet.Actual().SetOutput(o.FlagSet.Actual().Output())
	}

	o.Subcommands = append(o.Subcommands, cmd)

	return cmd
}

func (o *Command) VisitAll(fn func(c *Command)) {
	fn(o)

	for _, sub := range o.Subcommands {
		sub.VisitAll(fn)
	}
}

func (o *Command) Run(ctx context.Context, args []string) (CommandResultWrapper, error) {
	var result CommandResultWrapper

	err := o.run(ctx, args, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (o *Command) run(ctx context.Context, args []string, r *CommandResultWrapper) error {
	err := o.FlagSet.Parse(args)
	if o.help {
		o.PrintUsage()

		return flag.ErrHelp
	}
	if err != nil {
		return err
	}

	if o.OptPreRunFn != nil {
		err := o.OptPreRunFn(ctx)
		if err != nil {
			return err
		}
	}

	r.Commands = append(r.Commands, o.Name())

	if len(o.Subcommands) > 0 && o.FlagSet.Actual().NArg() > 0 {
		requestedSubCmd := o.FlagSet.Actual().Arg(0)

		for _, possible := range o.Subcommands {
			if possible.Name() == requestedSubCmd {
				err := possible.run(ctx, o.FlagSet.Actual().Args()[1:], r)
				if err != nil {
					return err
				}

				return nil
			}
		}

		if len(o.FlagSet.nonflags) == 0 {
			return fmt.Errorf("unknown subcommand: %q", requestedSubCmd)
		}
	}

	res, err := o.Fn(ctx)
	if err != nil {
		return err
	}

	r.Result = res

	return nil
}

type CommandResultWrapper struct {
	Commands []string

	Result CommandResult
}

type CommandResult interface {
	Human() string

	ExitStatus() uint64
}

type Serializable interface {
	Serialize() []byte
}

func NewSerialCommandResult(serial Serializable) CommandResult {
	return basicCommandResult{
		human: string(serial.Serialize()),
	}
}

func NewHumanCommandResult(output string) CommandResult {
	return basicCommandResult{
		human: output,
	}
}

type basicCommandResult struct {
	human string
}

func (o basicCommandResult) Human() string {
	return o.human
}

func (o basicCommandResult) ExitStatus() uint64 {
	return 0
}
