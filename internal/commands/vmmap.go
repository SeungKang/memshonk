package commands

import (
	"bytes"
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/memory"
)

func VmmapCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      "vmmap",
		Aliases:   []string{"v"},
		ShortHelp: "view the process's memory regions",
		NonFlags: []NonFlagSchema{
			{
				Name:     "addr",
				Desc:     "address to search for which region it is in",
				DefValue: "",
				DataType: "",
			},
		},
		CreateFn: func(c CommandConfig) (Command, error) {
			return VmmapCommand{
				args: VmmapCommandArgs{
					AddrStr: c.NonFlags.String("addr"),
				},
			}, nil
		},
	}
}

type VmmapCommandArgs struct {
	AddrStr string
}

type VmmapCommand struct {
	args VmmapCommandArgs
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

	if o.args.AddrStr != "" {
		ptr, err := memory.CreatePointerFromString(o.args.AddrStr)
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

		return nil, fmt.Errorf("address not found for %s", o.args.AddrStr)
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

		if others.Len() == 0 {
			others.WriteString("others:")
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

	if others.Len() > 0 {
		out.WriteByte('\n')
		others.WriteTo(&out)
	}

	return HumanCommandResult(out.String()), nil
}
