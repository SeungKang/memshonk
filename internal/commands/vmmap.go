package commands

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/memory"
)

const (
	VmmapCommandName = "vmmap"
)

func NewVmmapCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &VmmapCommand{
		session: config.Session,
	}

	root := fx.NewCommand(VmmapCommandName, "view the process's memory regions", cmd.run)

	root.FlagSet.StringNf(&cmd.searchStr, fx.ArgConfig{
		Name:        "search-str",
		Description: "address or name to filter regions",
	})

	return root
}

type VmmapCommand struct {
	session   apicompat.Session
	searchStr string
}

func (o *VmmapCommand) run(ctx context.Context) (fx.CommandResult, error) {
	process := o.session.SharedState().Progctl

	regions, err := process.Regions(ctx)
	if err != nil {
		return nil, err
	}

	if o.searchStr != "" {
		return o.search(ctx, regions)
	}

	return o.list(ctx, regions)
}

func (o *VmmapCommand) search(ctx context.Context, regions memory.Regions) (fx.CommandResult, error) {
	if strings.HasPrefix(o.searchStr, "0x") {
		ptr, err := memory.CreatePointerFromString(o.searchStr)
		if err != nil {
			return nil, err
		}

		resolvedPtr, err := o.session.SharedState().Progctl.ResolvePointer(ctx, ptr)
		if err != nil {
			return nil, err
		}

		region, foundRegion := regions.HasAddr(resolvedPtr)
		if foundRegion {
			return fx.NewHumanCommandResult(region.String()), nil
		}

		return nil, fmt.Errorf("address not found for %s", o.searchStr)
	}

	var out bytes.Buffer

	err := regions.IterObjects(func(object memory.Object) error {
		if !object.NameOrPathContains(o.searchStr) {
			return nil
		}

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
		return nil, fmt.Errorf("failed to find object matching: %q", o.searchStr)
	}

	return fx.NewHumanCommandResult(out.String()), nil
}

func (o *VmmapCommand) list(ctx context.Context, regions memory.Regions) (fx.CommandResult, error) {
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
		return fx.NewHumanCommandResult(out.String()), nil
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

	return fx.NewHumanCommandResult(out.String()), nil
}
