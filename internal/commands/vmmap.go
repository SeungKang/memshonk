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

	objects, err := process.MappedObjects(ctx)
	if err != nil {
		return nil, err
	}

	if o.args.searchStr != "" && strings.HasPrefix(o.args.searchStr, "0x") {
		ptr, err := memory.CreatePointerFromString(o.args.searchStr)
		if err != nil {
			return nil, err
		}

		resolvedPtr, err := s.Process().ResolvePointer(ctx, ptr)
		if err != nil {
			return nil, err
		}

		var result HumanCommandResult

		obj, foundObj := objects.HasAddr(resolvedPtr)
		if foundObj {
			result = HumanCommandResult(obj.String())
		}

		region, foundRegion := regions.HasAddr(resolvedPtr)
		if foundRegion {
			if result != "" {
				result += "\n"
			}

			result += HumanCommandResult(region.String())
		}

		if foundObj || foundRegion {
			return result, nil
		}

		return nil, fmt.Errorf("address not found for %s", o.args.searchStr)
	}

	objs := make(map[string]bytes.Buffer, objects.Len())
	const indent = "|-- "

	others := bytes.Buffer{}

	err = regions.Iter(func(_ int, region memory.Region) error {
		if region.Unaccessible() {
			// TODO: Implement argument to include these
			// unaccessible regions.
			return nil
		}

		object, hasMatch := objects.ContainsRegion(region)
		if hasMatch {
			buf := objs[object.Filepath]

			if buf.Len() > 0 {
				buf.WriteByte('\n')
			}

			buf.WriteString(indent)
			buf.WriteString(region.String())

			objs[object.Filepath] = buf

			return nil
		}

		others.WriteByte('\n')
		others.WriteString(indent)
		others.WriteString(region.String())

		return nil
	})

	if err != nil {
		return nil, err
	}

	var out bytes.Buffer

	err = objects.IterObjects(func(obj memory.MappedObject) error {
		if o.args.searchStr != "" {
			if !strings.Contains(obj.Filename, o.args.searchStr) {
				return nil
			}
		}

		if out.Len() > 0 {
			out.WriteByte('\n')
		}

		out.WriteString(obj.String())

		buf, hasIt := objs[obj.Filepath]
		if hasIt {
			out.WriteByte('\n')
			buf.WriteTo(&out)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	if o.args.searchStr != "" {
		return HumanCommandResult(out.String()), nil
	}

	if others.Len() > 0 {
		out.WriteString("\nothers:")
		others.WriteTo(&out)
	}

	return HumanCommandResult(out.String()), nil
}
