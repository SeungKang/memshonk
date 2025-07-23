package commands

import (
	"bytes"
	"context"
	"errors"
	"fmt"

	"github.com/SeungKang/memshonk/internal/hexdump"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/buger/goterm"
)

const (
	watchCommandName = "watch"
)

func WatchCommandSchema() CommandSchema {
	return CommandSchema{
		Name:      watchCommandName,
		Aliases:   []string{"w"},
		ShortHelp: "watch data at an address for changes",
		NonFlags: []NonFlagSchema{
			{
				Name:     "size",
				Desc:     "number of bytes to read",
				DataType: uint64(0),
			},
			{
				Name:     "addr",
				Desc:     "address to watch",
				DataType: "",
			},
		},
		CreateFn: func(c CommandConfig) (Command, error) {
			return WatchCommand{
				AddrStr:   c.NonFlags.String("addr"),
				SizeBytes: c.NonFlags.Uint64("size"),
			}, nil
		},
	}
}

type WatchCommand struct {
	AddrStr   string
	SizeBytes uint64
}

func (o WatchCommand) Name() string {
	return watchCommandName
}

func (o WatchCommand) Run(ctx context.Context, inOut IO, s Session) (CommandResult, error) {
	ptr, err := memory.CreatePointerFromString(o.AddrStr)
	if err != nil {
		return nil, err
	}

	watcher, err := s.Process().Watch(ctx, ptr, o.SizeBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to create memory watcher - %w", err)
	}
	defer watcher.Close()

	var src bytes.Buffer

	var dst bytes.Buffer

	hexdumpConfig := hexdump.Config{
		Src:    &src,
		Dst:    &dst,
		Colors: hexdump.NewColors(),
	}

	goterm.Clear()

	for read := range watcher.Results() {
		// By moving cursor to top-left position we ensure that console output
		// will be overwritten each time, instead of adding new.
		goterm.MoveCursor(1, 1)

		src.Write(read.Data)

		err := hexdump.Dump(ctx, hexdumpConfig)
		if err != nil {
			return nil, err
		}

		goterm.Println(dst.String())

		dst.Reset()

		goterm.Flush()
	}

	err = watcher.Err()

	if errors.Is(err, context.Canceled) {
		return nil, nil
	}

	return nil, err
}
