package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/fx"
	"github.com/SeungKang/memshonk/internal/hexdump"
	"github.com/SeungKang/memshonk/internal/vendored/goterm"
)

const (
	WatchCommandName = "watch"
)

func NewWatchCommand(config apicompat.NewCommandConfig) *fx.Command {
	cmd := &WatchCommand{
		session: config.Session,
	}

	root := fx.NewCommand(WatchCommandName, "watch data at an address for changes", cmd.run)

	root.FlagSet.Uint64Flag(&cmd.sizeBytes, 0, fx.ArgConfig{
		Name:         "size",
		Description:  "number of bytes to read",
		Required:     true,
		OptShortName: "s",
	})

	root.FlagSet.StringSliceNf(&cmd.addrStrs, fx.ArgConfig{
		Name:        "addrs",
		Description: "one or more addresses to watch " + addressTopicReferStr,
		Required:    true,
	})

	return root
}

type WatchCommand struct {
	session   apicompat.Session
	sizeBytes uint64
	addrStrs  []string
}

func (o *WatchCommand) run(ctx context.Context) (fx.CommandResult, error) {
	var cancelFn func()
	ctx, cancelFn = context.WithCancel(ctx)
	defer cancelFn()

	exeInfo, err := o.session.SharedState().Progctl.ExeInfo(ctx)
	if err != nil {
		return nil, err
	}

	terminal, hasTerm := o.session.Terminal()
	if !hasTerm {
		return nil, errCommandNeedsTerminal
	}

	screen := goterm.NewScreen(terminal)

	rows := make([]watchCommandRow, len(o.addrStrs))

	reads := make(chan watchReadEvent, len(o.addrStrs))

	for i, addrStr := range o.addrStrs {
		watcher, err := o.session.SharedState().Progctl.WatchLookup(ctx, addrStr, o.sizeBytes)
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
		Src:           &src,
		Dst:           &dst,
		OptStyle:      hexdump.DefaultStyle{Colors: hexdump.NewByteColors()},
		OptTitle:      "placeholder",
		OptOffsetBits: exeInfo.Bits,
	}

	_, numLinesPerHexdump, err := hexdumpConfig.OutputLen(o.sizeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hexdump output length per watcher - %w",
			err)
	}

	// The previous function does not include the trailing newline
	// we will add below.
	numLinesPerHexdump++

	resized, unsubFn := terminal.OnResize()
	defer unsubFn()

	size, err := terminal.Size()
	if err != nil {
		return nil, fmt.Errorf("failed to get terminal size - %w", err)
	}

	width := size.Cols
	height := size.Rows

	screen.Clear()
	screen.Flush()

loop:
	screen.MoveCursor(1, 1)

	select {
	case <-ctx.Done():
		return nil, nil
	case resize := <-resized:
		width = resize.NewSize.Cols
		height = resize.NewSize.Rows

		screen.Clear()
		screen.Flush()

		for _, row := range rows {
			row.write(numLinesPerHexdump, width, height, screen)
		}
	case read := <-reads:
		hexdumpConfig.OptStartOffset = uint64(read.row.addr)

		if read.err != nil {
			if errors.Is(read.err, context.Canceled) {
				return nil, nil
			}

			header := fmt.Sprintf("%#x (invalidated at %s - %s)\n",
				read.row.addr, time.Now().Format(time.TimeOnly), read.err)

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

		read.row.write(numLinesPerHexdump, width, height, screen)
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

func (o watchCommandRow) write(numLinesPerHexdump int, width int, height int, screen *goterm.Screen) {
	y := o.y(numLinesPerHexdump)

	if y > height {
		return
	}

	screen.MoveCursor(1, y)

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
				screen.Print(o.output[0 : i+1])

				break
			}
		}
	} else {
		screen.Print(o.output)
	}

	screen.Flush()
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
