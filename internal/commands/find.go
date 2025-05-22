package commands

import (
	"context"
	"fmt"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/progctl"
)

func FindCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      "find",
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

func (o FindCommand) Run(ctx context.Context, inOut IO, s Session) error {
	regions, err := s.Process().Regions(ctx)
	if err != nil {
		return err
	}

	process := s.Process()

	return regions.Iter(func(i int, region memory.Region) error {
		return o.searchRegion(region, inOut, process)
	})
}

func (o FindCommand) searchRegion(region memory.Region, inOut IO, process progctl.Process) error {
	if !region.Readable {
		return nil
	}

	reader, err := memory.NewBufferedReader(
		process,
		memory.AbsoluteAddrPointer(region.BaseAddress),
		region.Size)
	if err != nil {
		return err
	}

	matches, err := memory.FindAllReader(o.args.Pattern, reader)
	if err != nil {
		// TODO ignoring error
		return nil
	}

	if len(matches) > 0 {
		_, err = fmt.Fprintln(inOut.Stdout, "matches:", matches)
		if err != nil {
			return err
		}
	}

	return nil
}
