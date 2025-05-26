package commands

import (
	"bytes"
	"context"
	"fmt"
	"io"
	
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

func (o VmmapCommand) Run(ctx context.Context, inOut IO, s Session) error {
	process := s.Process()

	regions, err := process.Regions(ctx)
	if err != nil {
		return err
	}

	objects, err := process.MappedObjects(ctx)
	if err != nil {
		return err
	}

	if o.args.AddrStr != "" {
		ptr, err := memory.CreatePointerFromString(o.args.AddrStr)
		if err != nil {
			return err
		}

		resolvedPtr, err := s.Process().ResolvePointer(ctx, ptr)
		if err != nil {
			return err
		}

		obj, foundObj := objects.HasAddr(resolvedPtr)
		if foundObj {
			fmt.Fprintln(inOut.Stdout, obj.String())
		}

		region, foundRegion := regions.HasAddr(resolvedPtr)
		if foundRegion {
			fmt.Fprintln(inOut.Stdout, region.String())
		}

		if foundObj || foundRegion {
			return nil
		}

		return fmt.Errorf("address not found for %s", o.args.AddrStr)
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

			buf.WriteString(indent)
			buf.WriteString(region.String() + "\n")

			objs[object.Filepath] = buf

			return nil
		}

		if others.Len() == 0 {
			others.WriteString("others:\n")
		}

		others.WriteString(indent)
		others.WriteString(region.String() + "\n")

		return nil
	})

	if err != nil {
		return err
	}

	err = objects.IterObjects(func(obj memory.MappedObject) error {
		fmt.Fprintln(inOut.Stdout, obj.String())

		buf, hasIt := objs[obj.Filepath]
		if hasIt {
			_, err = io.Copy(inOut.Stdout, &buf)
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	_, err = io.Copy(inOut.Stdout, &others)
	return err
}
