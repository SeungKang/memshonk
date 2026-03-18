package commands

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/progctl"
)

const (
	FindCommandName = "find"
)

// Various find command search encoding formats.
const (
	stringFindEncoding    = "string"
	utf8FindEncoding      = "utf8"
	wstringleFindEncoding = "wstringle"
	utf16leFindEncoding   = "utf16le"
	wstringFindEncoding   = "wstring"
	utf16FindEncoding     = "utf16"
	wstringbeFindEncoding = "wstringbe"
	utf16beFindEncoding   = "utf16be"
	float32FindEncoding   = "float32"
	float32leFindEncoding = "float32le"
	float32beFindEncoding = "float32be"
	float64FindEncoding   = "float64"
	float64leFindEncoding = "float64le"
	float64beFindEncoding = "float64be"
	patternFindEncoding   = "pattern"
)

func NewFindCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &FindCommand{
		session: config.Session,
		stderr:  config.Stderr,
	}

	root := fx.NewCommand(FindCommandName, "find data in a process' memory", cmd.run)

	root.OptLongDesc = `ENCODING TYPES
  ` + utf8FindEncoding + `      - UTF-8 string
  ` + stringFindEncoding + `    - Alias to ` + utf8FindEncoding + `
  ` + utf16leFindEncoding + `   - UTF-16 string in little endian byte order
  ` + utf16FindEncoding + `     - Alias to ` + utf16leFindEncoding + `
  ` + wstringleFindEncoding + ` - Alias to ` + utf16leFindEncoding + `
  ` + wstringFindEncoding + `   - Alias to ` + utf16leFindEncoding + `
  ` + utf16beFindEncoding + `   - UTF-16 string in big endian byte order
  ` + wstringbeFindEncoding + ` - Alias to ` + utf16beFindEncoding + `
  ` + float32leFindEncoding + ` - A 32-bit float in little endian byte order
  ` + float32FindEncoding + `   - Alias to ` + float32leFindEncoding + `
  ` + float32beFindEncoding + ` - A 32-bit float in big endian byte order
  ` + float64leFindEncoding + ` - A 64-bit float in little endian byte order
  ` + float64FindEncoding + `   - Alias to ` + float64leFindEncoding + `
  ` + float64beFindEncoding + ` - A 64-bit float in big endian byte order
  ` + patternFindEncoding + `   - Pattern string (refer to "help pattern" for details)
`

	root.FlagSet.StringFlag(&cmd.encodingFormat, "pattern", fx.ArgConfig{
		Name:        "encoding",
		Description: "Optional: Specify encoding format of the search string (refer to help page for all possible values)",
	})

	root.FlagSet.StringSliceNf(&cmd.pattern, fx.ArgConfig{
		Name:        "search-value",
		Description: "Value to search for",
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

	encodingFormat := o.encodingFormat
	switch encodingFormat {
	case stringFindEncoding, utf8FindEncoding:
		parsedPattern, err = memory.ParsePatternFromUtf8(stringList)
	case wstringleFindEncoding, utf16leFindEncoding, wstringFindEncoding, utf16FindEncoding:
		parsedPattern, err = memory.ParsePatternFromUtf16(stringList, binary.LittleEndian)
	case wstringbeFindEncoding, utf16beFindEncoding:
		parsedPattern, err = memory.ParsePatternFromUtf16(stringList, binary.BigEndian)
	case float32FindEncoding, float32leFindEncoding, float32beFindEncoding:
		var endian binary.ByteOrder = binary.LittleEndian

		if encodingFormat == float32beFindEncoding {
			endian = binary.BigEndian
		}

		var buf bytes.Buffer

		for _, str := range o.pattern {
			f, err := strconv.ParseFloat(str, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse 32-bit float: %q - %w",
					str, err)
			}

			final := float32(f)

			err = binary.Write(&buf, endian, final)
			if err != nil {
				return nil, fmt.Errorf("failed to convert 32-bit float %v to binary - %w",
					f, err)
			}
		}

		parsedPattern = memory.PatternForRawBytes(buf.Bytes())
	case float64FindEncoding, float64leFindEncoding, float64beFindEncoding:
		var endian binary.ByteOrder = binary.LittleEndian

		if encodingFormat == float64beFindEncoding {
			endian = binary.BigEndian
		}

		var buf bytes.Buffer

		for _, str := range o.pattern {
			f, err := strconv.ParseFloat(str, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse 64-bit float: %q - %w",
					str, err)
			}

			err = binary.Write(&buf, endian, f)
			if err != nil {
				return nil, fmt.Errorf("failed to convert 64-bit float %v to binary - %w",
					f, err)
			}
		}

		parsedPattern = memory.PatternForRawBytes(buf.Bytes())
	case patternFindEncoding:
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
