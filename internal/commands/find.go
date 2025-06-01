package commands

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/progctl"
)

const (
	findCommandName = "find"
)

func FindCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      findCommandName,
		Aliases:   []string{"f"},
		ShortHelp: "find a pattern in memory",
		NonFlags: []NonFlagSchema{
			{
				Name:     "pattern",
				Desc:     "byte pattern to search for",
				DefValue: "",
				DataType: "",
			},
			{
				Name:     "start",
				Desc:     "the address to start searching from",
				DataType: "",
				DefValue: "",
			},
			{
				Name:     "end",
				Desc:     "the address to stop searching at",
				DataType: "",
				DefValue: "",
			},
		},
		CreateFn: func(c CommandConfig) (Command, error) {
			return NewFindCommand(FindCommandArgs{
				Pattern:   c.NonFlags.String("pattern"),
				StartAddr: c.NonFlags.String("start"),
				EndAddr:   c.NonFlags.String("end"),
			}), nil
		},
	}
}

type FindCommandArgs struct {
	Pattern   string
	StartAddr string
	EndAddr   string
}

func NewFindCommand(args FindCommandArgs) FindCommand {
	return FindCommand{
		args: args,
	}
}

type FindCommand struct {
	args FindCommandArgs
}

func (o FindCommand) Name() string {
	return findCommandName
}

func (o FindCommand) Run(ctx context.Context, inOut IO, s Session) (CommandResult, error) {
	regions, err := s.Process().Regions(ctx)
	if err != nil {
		return nil, err
	}

	process := s.Process()

	var matches MemoryPointerListCommandResult

	fmt.Fprint(inOut.Stderr, "searching")

	err = regions.Iter(func(i int, region memory.Region) error {
		matchedAddrs, err := o.searchRegion(region, inOut, process)
		if err != nil {
			return err
		}

		// print 70 "." to show search progress
		step := regions.Len() / 70
		if step == 0 {
			step = 1
		}

		if i%step == 0 {
			_, err = fmt.Fprint(inOut.Stderr, ".")
			if err != nil {
				return err
			}
		}

		matches = append(matches, matchedAddrs...)

		return nil
	})

	fmt.Fprintln(inOut.Stderr, "")

	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (o FindCommand) searchRegion(region memory.Region, inOut IO, process progctl.Process) ([]memory.Pointer, error) {
	if !region.Readable {
		return nil, nil
	}

	reader, err := memory.NewBufferedReader(
		process,
		memory.AbsoluteAddrPointer(region.BaseAddr),
		region.Size)
	if err != nil {
		return nil, err
	}

	matches, err := memory.FindAllReader(o.args.Pattern, reader)
	if err != nil {
		// TODO ignoring error
		return nil, nil
	}

	if len(matches) > 0 {
		return matches, nil
	}

	return nil, nil
}
