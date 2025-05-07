package commands

import (
	"context"
	"io"
)

type Command interface {
	Run(ctx context.Context, inputOutput IO, s Session) error
}

type Session interface {
	Process() Process
}

type Process interface {
	Attach(ctx context.Context) (int, error)

	ReadFromAddr(ctx context.Context, addr uint64, size uint) ([]byte, error)

	WriteToAddr(ctx context.Context, p []byte, addr uint64) error

	Detach(ctx context.Context) error
}

type IO struct {
	Stdout io.Writer

	Stderr io.Writer
}
