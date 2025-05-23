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
		NonFlags:  []NonFlagSchema{},
		CreateFn: func(c CommandConfig) (Command, error) {
			return &VmmapCommand{
				args: VmmapCommandArgs{},
			}, nil
		},
	}
}

type VmmapCommandArgs struct {
}

type VmmapCommand struct {
	args VmmapCommandArgs
}

func (o VmmapCommand) Run(ctx context.Context, inOut IO, s Session) error {
	process := s.Process()

	regions, err := process.Regions(context.Background())
	if err != nil {
		return err
	}

	objects, err := process.MappedObjects(ctx)
	if err != nil {
		return err
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

		var hasMatch bool

		err := objects.IterObjects(func(obj memory.MappedObject) error {
			if obj.ContainsAddr(region.BaseAddress) {
				hasMatch = true

				buf := objs[obj.Filepath]

				buf.WriteString(indent)
				buf.WriteString(region.String() + "\n")

				objs[obj.Filepath] = buf

				return memory.ErrStopIterating
			}

			return nil
		})
		if err != nil {
			return err
		}

		if !hasMatch {
			if others.Len() == 0 {
				others.WriteString("others:\n")
			}

			others.WriteString(indent)
			others.WriteString(region.String() + "\n")
		}

		return nil
	})
	if err != nil {
		return err
	}

	err = objects.IterObjects(func(obj memory.MappedObject) error {
		buf, hasIt := objs[obj.Filepath]
		if hasIt {
			fmt.Fprintln(inOut.Stdout, obj.String())
			return nil
		}

		_, err = io.Copy(inOut.Stdout, &buf)
		return err
	})
	if err != nil {
		return err
	}

	_, err = io.Copy(inOut.Stdout, &others)
	return err
}
