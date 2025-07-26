package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SeungKang/memshonk/internal/hexdump"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/termkit"
	"github.com/buger/goterm"
)

const (
	watchCommandName = "watch"
)

func WatchCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      watchCommandName,
		ShortHelp: "watch data at an address for changes",
		NonFlags: []NonFlagSchema{
			{
				Name:     "size",
				Desc:     "number of bytes to read",
				DataType: uint64(0),
			},
			{
				Name:     "addrs",
				Desc:     "one or more addresses to watch",
				DataType: []string{},
			},
		},
		CreateFn: func(c CommandConfig) (Command, error) {
			return WatchCommand{
				AddrStrs:  c.NonFlags.StringList("addrs"),
				SizeBytes: c.NonFlags.Uint64("size"),
			}, nil
		},
	}
}

type WatchCommand struct {
	SizeBytes uint64
	AddrStrs  []string
}

func (o WatchCommand) Name() string {
	return watchCommandName
}

func (o WatchCommand) Run(ctx context.Context, inOut IO, s Session) (CommandResult, error) {
	var cancelFn func()
	ctx, cancelFn = context.WithCancel(ctx)
	defer cancelFn()

	exeInfo, err := s.Process().ExeInfo(ctx)
	if err != nil {
		return nil, err
	}

	rows := make([]watchCommandRow, len(o.AddrStrs))

	reads := make(chan watchReadEvent, len(o.AddrStrs))

	for i, addrStr := range o.AddrStrs {
		ptr, err := memory.CreatePointerFromString(addrStr)
		if err != nil {
			return nil, err
		}

		watcher, err := s.Process().Watch(ctx, ptr, o.SizeBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to create memory watcher for %s - %w",
				addrStr, err)
		}

		row := &rows[i]

		row.rowIndex = i
		row.addr = watcher.Addr()

		go func() {
			for read := range watcher.Results() {
				select {
				case <-ctx.Done():
					return
				case reads <- watchReadEvent{
					row:  row,
					data: read.Data,
				}:
				}
			}

			select {
			case <-ctx.Done():
			case reads <- watchReadEvent{
				row: row,
				err: watcher.Err(),
			}:
			}
		}()
	}

	var src bytes.Buffer

	var dst bytes.Buffer

	hexdumpConfig := hexdump.Config{
		Src:          &src,
		Dst:          &dst,
		Colors:       hexdump.NewColors(),
		OptTitle:     "placeholder",
		OptOffColPad: exeInfo.Bits / 4, // 32 == 8, 64 == 16.
	}

	_, numLinesPerHexdump, err := hexdumpConfig.OutputLen(o.SizeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hexdump output length per watcher - %w",
			err)
	}

	// The previous function does not include the trailing newline
	// we will add below.
	numLinesPerHexdump++

	resized := termkit.NewResizedMonitor(ctx)

	width := goterm.Width()
	height := goterm.Height()

	goterm.Clear()
	goterm.Flush()

loop:
	goterm.MoveCursor(1, 1)

	select {
	case <-ctx.Done():
		return nil, nil
	case resize := <-resized.Events():
		width = resize.Width
		height = resize.Height

		goterm.Clear()
		goterm.Flush()

		for _, row := range rows {
			row.write(numLinesPerHexdump, width, height)
		}
	case read := <-reads:
		hexdumpConfig.OptStartOff = uint64(read.row.addr)

		if read.err != nil {
			if errors.Is(read.err, context.Canceled) {
				return nil, nil
			}

			header := fmt.Sprintf("%#x (invalidated at %s - %s)\n",
				read.row.addr, time.Now().Format(time.TimeOnly), err)

			after := strings.Index(read.row.output, "\n")

			if after > -1 {
				read.row.output = header + read.row.output[after+1:]
			} else {
				read.row.output = header
			}
		} else {
			hexdumpConfig.OptTitle = fmt.Sprintf("%#x (valid)",
				read.row.addr)

			src.Write(read.data)

			err := hexdump.Dump(context.Background(), hexdumpConfig)
			if err != nil {
				return nil, err
			}

			read.row.output = dst.String() + "\n"

			dst.Reset()
		}

		read.row.write(numLinesPerHexdump, width, height)
	}

	goto loop
}

type watchReadEvent struct {
	row  *watchCommandRow
	data []byte
	err  error
}

type watchCommandRow struct {
	rowIndex int
	addr     uintptr
	output   string
}

func (o watchCommandRow) write(numLinesPerHexdump int, width int, height int) {
	y := o.y(numLinesPerHexdump)

	if y > height {
		return
	}

	goterm.MoveCursor(1, y)

	// y = 10: 10+3 > 12
	if y+numLinesPerHexdump > height {
		i := len(o.output) - 1
		numLines := numLinesPerHexdump

		for {
			i = strings.LastIndex(o.output[0:i], "\n")
			if i <= 0 {
				return
			}

			numLines--

			if y+numLines < height {
				goterm.Print(o.output[0 : i+1])
				break
			}
		}
	} else {
		goterm.Print(o.output)
	}

	goterm.Flush()
}

func (o watchCommandRow) y(numLinesPerHexdump int) int {
	// (rowIndex * numLines) + 1
	//
	// rowIndex = 2, numLines = 2 | (2*2)+1 == 5
	//
	// 0:  -
	//     -
	// 1:  -
	//     -
	// 2:  * <---
	//     -
	//     -
	//
	// rowIndex = 0, numLines = 2 | (0*2)+1 == 1
	//
	// 0:  * <---
	//     -
	// 1:  -
	//     -
	// 2:  -
	//     -
	//     -
	return (o.rowIndex * numLinesPerHexdump) + 1
}
