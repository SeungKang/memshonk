package fx

import (
	"context"
	"flag"
	"fmt"
)

func NewCommand(name string, description string, fn func(context.Context) (CommandResult, error)) *Command {
	set := NewFlagSet(name)

	set.internal.Usage = func() {}

	cmd := &Command{
		FlagSet:     set,
		Description: description,
		Fn:          fn,
	}

	// I tried using BoolFuncFlag to return flag.ErrHelp,
	// but that calls FlagSet.failf which uses errors.New,
	// which makes it impossible to check if flag.ErrHelp
	// is present using errors.Is.
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
	CustomFn    func(context.Context, RunCommandConfig) (CommandResult, error)

	OptLongDesc string
	OptParent   *Command
	OptPreRunFn func(context.Context) error

	help bool
}

type RunCommandConfig struct {
	Args []string
}

func (o *Command) Name() string {
	return o.FlagSet.Actual().Name()
}

func (o *Command) PrintUsage() {
	usage := `SYNOPSIS
` + o.synopsis("  ") + `

DESCRIPTION
  ` + o.Description + `

`

	if o.OptLongDesc != "" {
		usage += o.OptLongDesc + "\n"
	}

	usage += "OPTIONS\n"

	o.FlagSet.internal.Output().Write([]byte(usage))

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

			if info.Config.Required {
				if flags != "" {
					flags += " "
				}

				dashes := "--"
				name := info.Config.Name

				if info.Config.OptShortName != "" {
					dashes = "-"
					name = info.Config.OptShortName
				}

				usageInfo := getFlagUsageInfo(info.OptFlag)

				flags += dashes + name + " " + usageInfo.DatatypeStr
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
	return o.AddSubcommandCustom(NewCommand(name, description, fn))
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

	err := o.runRecurse(ctx, args, &result)
	if err != nil {
		return result, err
	}

	return result, nil
}

func (o *Command) runRecurse(ctx context.Context, args []string, r *CommandResultWrapper) error {
	r.Commands = append(r.Commands, o.Name())

	if o.CustomFn != nil {
		if o.Fn != nil {
			return fmt.Errorf("fn and custom fn fields cannot both be non-nil")
		}

		res, err := o.CustomFn(ctx, RunCommandConfig{
			Args: args,
		})
		if err != nil {
			return err
		}

		r.Result = res

		return nil
	}

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

	if len(o.Subcommands) > 0 && o.FlagSet.Actual().NArg() > 0 {
		requestedSubCmd := o.FlagSet.Actual().Arg(0)

		for _, possible := range o.Subcommands {
			if possible.Name() == requestedSubCmd {
				err := possible.runRecurse(ctx, o.FlagSet.Actual().Args()[1:], r)
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

	if o.Fn != nil {
		res, err := o.Fn(ctx)
		if err != nil {
			return err
		}

		r.Result = res
	}

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
