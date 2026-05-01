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

	root.FlagSet.StringFlag(&cmd.match, "", fx.ArgConfig{
		Name:        "match",
		Description: "filter regions by name/path",
	})

	root.FlagSet.StringFlag(&cmd.addr, "", fx.ArgConfig{
		Name:        "addr",
		Description: "find the region containing the given address",
	})

	root.FlagSet.StringFlag(&cmd.region, "", fx.ArgConfig{
		Name:        "region",
		Description: "filter regions by name (e.g. heap, stack, or filename)",
	})

	root.FlagSet.StringFlag(&cmd.property, "", fx.ArgConfig{
		Name:        "property",
		Description: "comma-separated columns to display (name, base, end, alloc-base, perm, size, type, state)",
	})

	root.FlagSet.BoolFlag(&cmd.showWindowsInaccessible, false, fx.ArgConfig{
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
	session                 apicompat.Session
	match                   string
	addr                    string
	region                  string
	property                string
	showWindowsInaccessible bool
	flat                    bool
}

func (o *VmmapCommand) run(ctx context.Context) (fx.CommandResult, error) {
	process := o.session.SharedState().Progctl

	regions, err := process.Regions(ctx)
	if err != nil {
		return nil, err
	}

	if o.addr != "" {
		return o.searchAddr(ctx, regions)
	}

	if o.property != "" {
		return o.listProperties(ctx, regions)
	}

	if o.match != "" {
		return o.searchMatch(ctx, regions)
	}

	if o.region != "" {
		return o.searchFilter(ctx, regions)
	}

	if o.flat {
		return o.listFlat(ctx, regions)
	}

	return o.list(ctx, regions)
}

func (o *VmmapCommand) searchAddr(ctx context.Context, regions memory.Regions) (fx.CommandResult, error) {
	ptr, err := memory.CreatePointerFromString(o.addr)
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

	return nil, fmt.Errorf("address not found for %s", o.addr)
}

func (o *VmmapCommand) searchMatch(ctx context.Context, regions memory.Regions) (fx.CommandResult, error) {
	var out bytes.Buffer

	err := regions.IterObjects(func(object memory.Object) error {
		if !object.NameOrPathContains(o.match) {
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
		return nil, fmt.Errorf("failed to find object matching: %q", o.match)
	}

	return fx.NewHumanCommandResult(out.String()), nil
}

func (o *VmmapCommand) searchFilter(ctx context.Context, regions memory.Regions) (fx.CommandResult, error) {
	region := strings.ToLower(o.region)
	var out bytes.Buffer

	err := regions.IterObjects(func(object memory.Object) error {
		if !strings.Contains(strings.ToLower(object.Name), region) {
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

	err = regions.IterNonObjects(func(r *memory.Region) error {
		if !strings.Contains(strings.ToLower(r.Type.String()), region) {
			return nil
		}

		if out.Len() > 0 {
			out.WriteByte('\n')
		}

		out.WriteString(r.String())

		return nil
	})
	if err != nil {
		return nil, err
	}

	if out.Len() == 0 {
		return nil, fmt.Errorf("failed to find region matching: %q", o.region)
	}

	return fx.NewHumanCommandResult(out.String()), nil
}

func (o *VmmapCommand) listProperties(ctx context.Context, regions memory.Regions) (fx.CommandResult, error) {
	props := strings.Split(o.property, ",")
	for i, p := range props {
		props[i] = strings.TrimSpace(strings.ToLower(p))
	}

	regionFilter := strings.ToLower(o.region)
	matchFilter := strings.ToLower(o.match)

	var out bytes.Buffer

	err := regions.Iter(func(_ int, r memory.Region) error {
		if r.NoPermissions() && !o.showWindowsInaccessible {
			return nil
		}

		if regionFilter != "" {
			fileName := strings.ToLower(r.Parent.FileName)
			typeName := strings.ToLower(r.Type.String())
			if !strings.Contains(fileName, regionFilter) && !strings.Contains(typeName, regionFilter) {
				return nil
			}
		}

		if matchFilter != "" {
			nameOrPath := strings.ToLower(r.NameOrPath())
			if !strings.Contains(nameOrPath, matchFilter) {
				return nil
			}
		}

		if out.Len() > 0 {
			out.WriteByte('\n')
		}

		writeByte := func(b bool, on, off byte) {
			if b {
				out.WriteByte(on)
			} else {
				out.WriteByte(off)
			}
		}

		for i, prop := range props {
			if i > 0 {
				out.WriteByte(' ')
			}

			switch prop {
			case "base":
				fmt.Fprintf(&out, "%#012x", r.BaseAddr)
			case "end":
				fmt.Fprintf(&out, "%#012x", r.EndAddr)
			case "size":
				fmt.Fprintf(&out, "%#012x", r.Size)
			case "perm":
				writeByte(r.Readable, 'r', '-')
				writeByte(r.Writeable, 'w', '-')
				writeByte(r.Executable, 'x', '-')
				out.WriteByte(' ')
				writeByte(r.Copyable, 'C', '-')
				writeByte(r.Shared, 'S', '-')
			case "type":
				out.WriteString(r.Type.String())
			case "state":
				out.WriteString(r.State.String())
			case "name":
				out.WriteString(r.NameOrPath())
			case "alloc-base":
				fmt.Fprintf(&out, "%#012x", r.AllocBase)
			default:
				return fmt.Errorf("unknown property: %q", prop)
			}
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if out.Len() == 0 {
		return nil, fmt.Errorf("no regions matched")
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
		if region.NoPermissions() && !o.showWindowsInaccessible {
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
		if region.NoPermissions() && !o.showWindowsInaccessible {
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
