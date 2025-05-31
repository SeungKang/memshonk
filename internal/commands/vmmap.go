package commands

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/SeungKang/memshonk/internal/memory"
)

const (
	vmmapCommandName = "vmmap"
)

func VmmapCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      vmmapCommandName,
		Aliases:   []string{"v"},
		ShortHelp: "view the process's memory regions",
		NonFlags: []NonFlagSchema{
			{
				Name:     "search-str",
				Desc:     "address or name to filter regions",
				DefValue: "",
				DataType: "",
			},
		},
		CreateFn: func(c CommandConfig) (Command, error) {
			return VmmapCommand{
				args: VmmapCommandArgs{
					searchStr: c.NonFlags.String("search-str"),
				},
			}, nil
		},
	}
}

type VmmapCommandArgs struct {
	searchStr string
}

type VmmapCommand struct {
	args VmmapCommandArgs
}

func (o VmmapCommand) Name() string {
	return vmmapCommandName
}

func (o VmmapCommand) Run(ctx context.Context, inOut IO, s Session) (CommandResult, error) {
	process := s.Process()

	regions, err := process.Regions(ctx)
	if err != nil {
		return nil, err
	}

	if o.args.searchStr != "" {
		return o.search(ctx, regions, s)
	}

	return o.list(ctx, regions)
}

func (o VmmapCommand) search(ctx context.Context, regions memory.Regions, s Session) (CommandResult, error) {
	if strings.HasPrefix(o.args.searchStr, "0x") {
		ptr, err := memory.CreatePointerFromString(o.args.searchStr)
		if err != nil {
			return nil, err
		}

		resolvedPtr, err := s.Process().ResolvePointer(ctx, ptr)
		if err != nil {
			return nil, err
		}

		region, foundRegion := regions.HasAddr(resolvedPtr)
		if foundRegion {
			return HumanCommandResult(region.String()), nil
		}

		return nil, fmt.Errorf("address not found for %s", o.args.searchStr)
	}

	var out bytes.Buffer

	err := regions.IterObjectsMatching(o.args.searchStr, func(object memory.Object) error {
		if out.Len() > 0 {
			out.WriteByte('\n')
		}

		out.WriteString(object.String())

		return nil
	})
	if err != nil {
		return nil, err
	}

	if out.Len() == 0 {
		return nil, fmt.Errorf("failed to find object matching: %q", o.args.searchStr)
	}

	return HumanCommandResult(out.String()), nil
}

func (o VmmapCommand) list(ctx context.Context, regions memory.Regions) (CommandResult, error) {
	var out bytes.Buffer

	err := regions.IterObjects(func(obj memory.Object) error {
		if out.Len() > 0 {
			out.WriteByte('\n')
		}

		out.WriteString(obj.String())

		return nil
	})
	if err != nil {
		return nil, err
	}

	if regions.NonObjectsLen() == 0 {
		return HumanCommandResult(out.String()), nil
	}

	out.WriteString("\nothers:")

	err = regions.IterNonObjects(func(region *memory.Region) error {
		if region.NoPermissions() {
			// TODO: Implement argument to include these
			// inaccessible regions.
			return nil
		}

		out.WriteByte('\n')

		out.WriteString(region.String())

		return nil
	})
	if err != nil {
		return nil, err
	}

	return HumanCommandResult(out.String()), nil
}
