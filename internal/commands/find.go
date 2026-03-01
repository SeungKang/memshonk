package commands

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strings"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/progctl"
)

const (
	FindCommandName = "find"
)

func NewFindCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &FindCommand{
		session: config.Session,
		stderr:  config.Stderr,
	}

	root := fx.NewCommand(FindCommandName, "find a pattern in memory", cmd.run)

	root.FlagSet.StringFlag(&cmd.encodingFormat, "pattern", fx.ArgConfig{
		Name:        "encoding",
		Description: "Optional: Specify encoding format of pattern",
	})

	root.FlagSet.StringSliceNf(&cmd.pattern, fx.ArgConfig{
		Name:        "pattern",
		Description: "Byte pattern to search for",
		Required:    true,
	})

	return root
}

type FindCommand struct {
	session        apicompat.Session
	encodingFormat string
	pattern        []string
	stderr         io.Writer
}

func (o *FindCommand) run(ctx context.Context) (fx.CommandResult, error) {
	var parsedPattern memory.ParsedPattern
	var err error
	stringList := strings.Join(o.pattern, " ")

	// TODO: Document encoding formats
	encodingFormat := o.encodingFormat
	switch encodingFormat {
	case "string", "utf8":
		parsedPattern, err = memory.ParsePatternFromUtf8(stringList)
	case "wstringle", "utf16le", "wstring", "utf16":
		parsedPattern, err = memory.ParsePatternFromUtf16(stringList, binary.LittleEndian)
	case "wstringbe", "utf16be":
		parsedPattern, err = memory.ParsePatternFromUtf16(stringList, binary.BigEndian)
	case "pattern":
		parsedPattern, err = memory.ParsePattern(stringList)
	default:
		return nil, fmt.Errorf("unknown encoding format: %q", encodingFormat)
	}
	if err != nil {
		return nil, err
	}

	regions, err := o.session.SharedState().Progctl.Regions(ctx)
	if err != nil {
		return nil, err
	}

	process := o.session.SharedState().Progctl

	var matches FindCommandResult

	fmt.Fprint(o.stderr, "searching")

	err = regions.Iter(func(i int, region memory.Region) error {
		if !region.Readable {
			return nil
		}

		matchedAddrs, err := o.searchRegion(ctx, parsedPattern, region, process)
		if err != nil {
			return err
		}

		// print 70 "." to show search progress
		step := regions.Len() / 70
		if step == 0 {
			step = 1
		}

		if i%step == 0 {
			_, err = fmt.Fprint(o.stderr, ".")
			if err != nil {
				return err
			}
		}

		matches.results = append(matches.results, matchedAddrs...)

		return nil
	})

	fmt.Fprintln(o.stderr, "")

	if err != nil {
		return nil, err
	}

	return fx.NewSerialCommandResult(matches), nil
}

func (o *FindCommand) searchRegion(ctx context.Context, parsedPattern memory.ParsedPattern, region memory.Region, process progctl.Process) ([]memory.FindResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Keep going.
	}

	reader, err := memory.NewBufferedReader(
		process,
		memory.AbsoluteAddrPointer(region.BaseAddr),
		region.Size)
	if err != nil {
		return nil, err
	}

	matches, err := memory.FindAllReader(ctx, parsedPattern, reader)
	if err != nil {
		// TODO ignoring error
		return nil, nil
	}

	if len(matches) > 0 {
		return matches, nil
	}

	return nil, nil
}

type FindCommandResult struct {
	results []memory.FindResult
}

func (o FindCommandResult) Serialize() []byte {
	buf := bytes.Buffer{}

	for i, u := range o.results {
		buf.WriteString(u.Addr.String())

		if i < len(o.results)-1 {
			buf.WriteString(" ")
		}
	}

	return buf.Bytes()
}
