package commands

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/progctl"
)

const (
	ScanCommandName = "scan"
)

func NewScanCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &ScanCommand{
		session: config.Session,
		stderr:  config.Stderr,
	}

	root := fx.NewCommand(ScanCommandName, "search process memory for values or byte patterns", cmd.run)

	root.FlagSet.StringFlag(&cmd.datatype, "pattern", fx.ArgConfig{
		Name:        "datatype",
		Description: "Specify datatype of the search string " + datatypesTopicReferStr,
	})

	root.FlagSet.StringFlag(&cmd.inputFormat, rawEncoding, fx.ArgConfig{
		Name:        "input-format",
		Description: "Specify the input `format` of the search string " + formatsTopicReferStr,
	})

	root.FlagSet.StringSliceNf(&cmd.pattern, fx.ArgConfig{
		Name:        "search-value",
		Description: "Value to search for",
		Required:    true,
	})

	return root
}

type ScanCommand struct {
	session     apicompat.Session
	datatype    string
	inputFormat string
	pattern     []string
	stderr      io.Writer
}

func (o *ScanCommand) run(ctx context.Context) (fx.CommandResult, error) {
	start := time.Now()

	var parsedPattern memory.ParsedPattern
	var err error
	stringList := strings.Join(o.pattern, " ")

	data, err := decodeDataStr(o.inputFormat, stringList)
	if err != nil {
		return nil, err
	}

	searchStr := string(data)

	switch o.datatype {
	case rawDataType, stringDataType, stringleDataType, utf8DataType, utf8leDataType:
		parsedPattern, err = memory.ParsePatternFromUtf8(searchStr)
	case stringbeDataType, utf8beDataType, cstringbeDataType:
		return nil, fmt.Errorf("TODO: %q needs to be implemented", o.datatype)
	case wstringleDataType, utf16leDataType, wstringDataType, utf16DataType:
		parsedPattern, err = memory.ParsePatternFromUtf16(searchStr, binary.LittleEndian)
	case wstringbeDataType, utf16beDataType:
		parsedPattern, err = memory.ParsePatternFromUtf16(searchStr, binary.BigEndian)
	case cstringDataType, cstringleDataType:
		// This is kind of half-ass, but whatever.
		searchStr += "\x00"

		parsedPattern, err = memory.ParsePatternFromUtf8(searchStr)
	case uint16DataType, uint16leDataType, uint16beDataType:
		var endian binary.ByteOrder = binary.LittleEndian

		if o.datatype == uint16beDataType {
			endian = binary.BigEndian
		}

		var buf bytes.Buffer

		for _, str := range o.pattern {
			v, err := stringWithBasePrefixToUint(str, 16)
			if err != nil {
				return nil, fmt.Errorf("failed to parse 16-bit uint: %q - %w",
					str, err)
			}

			final := uint16(v)

			err = binary.Write(&buf, endian, final)
			if err != nil {
				return nil, fmt.Errorf("failed to convert 16-bit uint %v to binary - %w",
					v, err)
			}
		}

		parsedPattern = memory.PatternForRawBytes(buf.Bytes())
	case uint32DataType, uint32leDataType, uint32beDataType:
		var endian binary.ByteOrder = binary.LittleEndian

		if o.datatype == uint32beDataType {
			endian = binary.BigEndian
		}

		var buf bytes.Buffer

		for _, str := range o.pattern {
			v, err := stringWithBasePrefixToUint(str, 32)
			if err != nil {
				return nil, fmt.Errorf("failed to parse 32-bit uint: %q - %w",
					str, err)
			}

			final := uint32(v)

			err = binary.Write(&buf, endian, final)
			if err != nil {
				return nil, fmt.Errorf("failed to convert 32-bit uint %v to binary - %w",
					v, err)
			}
		}

		parsedPattern = memory.PatternForRawBytes(buf.Bytes())
	case uint64DataType, uint64leDataType, uint64beDataType:
		var endian binary.ByteOrder = binary.LittleEndian

		if o.datatype == uint64beDataType {
			endian = binary.BigEndian
		}

		var buf bytes.Buffer

		for _, str := range o.pattern {
			v, err := stringWithBasePrefixToUint(str, 64)
			if err != nil {
				return nil, fmt.Errorf("failed to parse 64-bit uint: %q - %w",
					str, err)
			}

			err = binary.Write(&buf, endian, v)
			if err != nil {
				return nil, fmt.Errorf("failed to convert 64-bit uint %v to binary - %w",
					v, err)
			}
		}

		parsedPattern = memory.PatternForRawBytes(buf.Bytes())
	case float32DataType, float32leDataType, float32beDataType:
		var endian binary.ByteOrder = binary.LittleEndian

		if o.datatype == float32beDataType {
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
	case float64DataType, float64leDataType, float64beDataType:
		var endian binary.ByteOrder = binary.LittleEndian

		if o.datatype == float64beDataType {
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
	case patternDataType:
		parsedPattern, err = memory.ParsePattern(searchStr)
	default:
		return nil, fmt.Errorf("unknown data type: %q", o.datatype)
	}
	if err != nil {
		return nil, err
	}

	regions, err := o.session.SharedState().Progctl.Regions(ctx)
	if err != nil {
		return nil, err
	}

	process := o.session.SharedState().Progctl

	var matches ScanCommandResult

	numReadable := 0
	regions.Iter(func(_ int, region memory.Region) error {
		if region.Readable {
			numReadable++
		}
		return nil
	})

	const barWidth = 25
	printProgress := func(processed int) {
		var pct, filled int
		if numReadable > 0 {
			pct = processed * 100 / numReadable
			filled = processed * barWidth / numReadable
		}
		bar := strings.Repeat("=", filled)
		if filled < barWidth {
			bar += ">"
			bar += strings.Repeat(" ", barWidth-filled-1)
		}
		fmt.Fprintf(o.stderr, "\rscanning [%s] %3d%% (%d/%d)",
			bar, pct, processed, numReadable)
	}

	printProgress(0)

	type workerResult struct {
		matches []memory.ScanResult
		err     error
	}

	numWorkers := runtime.NumCPU()
	regionCh := make(chan memory.Region)
	resultCh := make(chan workerResult, numWorkers)

	scanCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < numWorkers; i++ {
		wg.Add(1)

		go func() {
			defer wg.Done()

			for region := range regionCh {
				m, err := o.searchRegion(scanCtx, parsedPattern, region, process)
				resultCh <- workerResult{matches: m, err: err}
			}
		}()
	}

	go func() {
		defer close(regionCh)

		regions.Iter(func(_ int, region memory.Region) error {
			if !region.Readable {
				return nil
			}

			select {
			case regionCh <- region:
				return nil
			case <-scanCtx.Done():
				return scanCtx.Err()
			}
		})
	}()

	var firstErr error
	var collectWg sync.WaitGroup
	collectWg.Add(1)

	go func() {
		defer collectWg.Done()
		processed := 0

		for result := range resultCh {
			if result.err != nil && firstErr == nil {
				firstErr = result.err
				cancel()
			}

			matches.results = append(matches.results, result.matches...)
			processed++
			printProgress(processed)
		}
	}()

	wg.Wait()
	close(resultCh)
	collectWg.Wait()

	fmt.Fprintln(o.stderr, "")
	fmt.Fprintf(o.stderr, "found: %d | total: %s\n",
		len(matches.results), time.Since(start).Round(time.Millisecond))

	err = firstErr
	if err != nil {
		return nil, err
	}

	if len(matches.results) == 0 {
		return nil, nil
	}

	return fx.NewSerialCommandResult(matches), nil
}

func (o *ScanCommand) searchRegion(ctx context.Context, parsedPattern memory.ParsedPattern, region memory.Region, process progctl.Process) ([]memory.ScanResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Keep going.
	}

	// TODO move this logic into memory.FindAllReader
	if !parsedPattern.HasWildcards() {
		data, _, err := process.ReadFromAddr(ctx, memory.AbsoluteAddrPointer(region.BaseAddr), region.Size)
		if err != nil {
			return nil, nil
		}
		return memory.FindAllBytes(data, parsedPattern.RawBytes(), region.BaseAddr), nil
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

	return matches, nil
}

type ScanCommandResult struct {
	results []memory.ScanResult
}

func (o ScanCommandResult) Serialize() []byte {
	buf := bytes.Buffer{}

	for i, u := range o.results {
		buf.WriteString(u.Addr.String())

		if i < len(o.results)-1 {
			buf.WriteString(" ")
		}
	}

	return buf.Bytes()
}

func stringWithBasePrefixToUint(str string, bitSize int) (uint64, error) {
	base := 10

	switch {
	case strings.HasPrefix(str, "0b"):
		base = 2
		str = str[2:]
	case strings.HasPrefix(str, "0o"):
		base = 8
		str = str[2:]
	case strings.HasPrefix(str, "0x"):
		base = 16
		str = str[2:]
	}

	return strconv.ParseUint(str, base, bitSize)
}
