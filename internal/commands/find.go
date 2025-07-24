package commands

import (
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/progctl"
)

const (
	findCommandName = "find"
)

func FindCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      findCommandName,
		Aliases:   []string{"f"},
		ShortHelp: "find a pattern in memory",
		Flags: []FlagSchema{
			{
				Short:      "e",
				Long:       "encoding",
				Desc:       "Optional: Specify encoding format of pattern",
				DataType:   "",
				DefaultVal: "pattern",
			},
		},
		NonFlags: []NonFlagSchema{
			{
				Name:     "pattern",
				Desc:     "byte pattern to search for",
				DataType: []string{},
				DefValue: nil,
			},
		},
		CreateFn: func(c CommandConfig) (Command, error) {
			return NewFindCommand(FindCommandArgs{
				EncodingFormat: c.Flags.String("encoding"),
				Pattern:        c.NonFlags.StringList("pattern"),
			}), nil
		},
	}
}

type FindCommandArgs struct {
	EncodingFormat string
	Pattern        []string
}

func NewFindCommand(args FindCommandArgs) FindCommand {
	return FindCommand{
		args: args,
	}
}

type FindCommand struct {
	args FindCommandArgs
}

func (o FindCommand) Name() string {
	return findCommandName
}

func (o FindCommand) Run(ctx context.Context, inOut IO, s Session) (CommandResult, error) {
	var parsedPattern memory.ParsedPattern
	var err error
	stringList := strings.Join(o.args.Pattern, " ")

	// TODO: Document encoding formats
	encodingFormat := o.args.EncodingFormat
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

	err = s.Process().Suspend(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to suspend process - %w", err)
	}
	defer func() {
		deferCtx, doneFn := context.WithTimeout(context.Background(), time.Second)
		defer doneFn()

		err := s.Process().Resume(deferCtx)
		if err != nil {
			fmt.Fprintf(inOut.Stderr, "failed to resume process - %s\n", err)
		}
	}()

	regions, err := s.Process().Regions(ctx)
	if err != nil {
		return nil, err
	}

	process := s.Process()

	var matches MemoryPointerListCommandResult

	fmt.Fprint(inOut.Stderr, "searching")

	err = regions.Iter(func(i int, region memory.Region) error {
		if !region.Readable {
			return nil
		}

		matchedAddrs, err := o.searchRegion(parsedPattern, region, inOut, process)
		if err != nil {
			return err
		}

		// print 70 "." to show search progress
		step := regions.Len() / 70
		if step == 0 {
			step = 1
		}

		if i%step == 0 {
			_, err = fmt.Fprint(inOut.Stderr, ".")
			if err != nil {
				return err
			}
		}

		matches = append(matches, matchedAddrs...)

		return nil
	})

	fmt.Fprintln(inOut.Stderr, "")

	if err != nil {
		return nil, err
	}

	return matches, nil
}

func (o FindCommand) searchRegion(parsedPattern memory.ParsedPattern, region memory.Region, inOut IO, process progctl.Process) ([]memory.Pointer, error) {
	reader, err := memory.NewBufferedReader(
		process,
		memory.AbsoluteAddrPointer(region.BaseAddr),
		region.Size)
	if err != nil {
		return nil, err
	}

	matches, err := memory.FindAllReader(parsedPattern, reader)
	if err != nil {
		// TODO ignoring error
		return nil, nil
	}

	if len(matches) > 0 {
		return matches, nil
	}

	return nil, nil
}
