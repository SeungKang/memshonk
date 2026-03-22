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

	root.FlagSet.BoolFlag(&cmd.showWindowsInaccesible, false, fx.ArgConfig{
		Name:        "show-windows-inaccessible",
		Description: "include Windows regions with no access protections (e.g. reserved, free)",
	})

	root.FlagSet.BoolFlag(&cmd.flat, false, fx.ArgConfig{
		Name:        "flat",
		Description: "show all regions in consecutive address order",
	})

	return root
}

type VmmapCommand struct {
	session                apicompat.Session
	searchStr              string
	showWindowsInaccesible bool
	flat                   bool
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

	if o.flat {
		return o.listFlat(ctx, regions)
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
		if region.NoPermissions() && !o.showWindowsInaccesible {
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

type flatRow struct {
	region    memory.Region
	objectStr string
}

// allocbColWidth is the fixed width of the "(allocb: 0x000000000000) " column.
const allocbColWidth = 25

func (o *VmmapCommand) listFlat(ctx context.Context, regions memory.Regions) (fx.CommandResult, error) {
	var rows []flatRow
	hasAnyAllocb := false
	maxTypeStateLen := 0

	err := regions.Iter(func(i int, region memory.Region) error {
		if region.NoPermissions() && !o.showWindowsInaccesible {
			return nil
		}

		row := flatRow{region: region}

		if region.Parent.IsSet {
			row.objectStr = region.NameOrPath() + " (id: " + region.Parent.ID.String() + ")"
		}

		if region.AllocBase > 0 {
			hasAnyAllocb = true
		}

		typeStateLen := 2 + len(region.Type.String()) + 2 + len(region.State.String()) + 1 // "(type, state)"
		if typeStateLen > maxTypeStateLen {
			maxTypeStateLen = typeStateLen
		}

		rows = append(rows, row)

		return nil
	})
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer

	for _, row := range rows {
		if out.Len() > 0 {
			out.WriteByte('\n')
		}

		r := row.region

		fmt.Fprintf(&out, "%#012x-%#012x ", r.BaseAddr, r.EndAddr)

		if r.AllocBase > 0 {
			fmt.Fprintf(&out, "(allocb: %#012x) ", r.AllocBase)
		} else if hasAnyAllocb {
			out.WriteString(strings.Repeat(" ", allocbColWidth))
		}

		writePerm := func(b bool, on, off byte) {
			if b {
				out.WriteByte(on)
			} else {
				out.WriteByte(off)
			}
		}

		writePerm(r.Readable, 'r', '-')
		writePerm(r.Writeable, 'w', '-')
		writePerm(r.Executable, 'x', '-')
		out.WriteByte(' ')
		writePerm(r.Copyable, 'C', '-')
		writePerm(r.Shared, 'S', '-')

		fmt.Fprintf(&out, " %#012x ", r.Size)

		typeStateStr := fmt.Sprintf("(%s, %s)", r.Type.String(), r.State.String())
		out.WriteString(typeStateStr)

		if row.objectStr != "" {
			out.WriteString(strings.Repeat(" ", maxTypeStateLen-len(typeStateStr)))
			out.WriteString("  ")
			out.WriteString(row.objectStr)
		}
	}

	return fx.NewHumanCommandResult(out.String()), nil
}
