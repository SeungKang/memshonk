package commands

import (
	"context"
	"io"

	"github.com/SeungKang/memshonk/internal/memory"
)

type Command interface {
	Run(ctx context.Context, inputOutput IO, s Session) error
}

type Session interface {
	Process() Process
}

type Process interface {
	Attach(ctx context.Context) (int, error)

	ReadFromAddr(ctx context.Context, addr memory.Pointer, size uint) ([]byte, error)

	WriteToAddr(ctx context.Context, p []byte, addr memory.Pointer) error

	Detach(ctx context.Context) error
}

type IO struct {
	Stdout io.Writer

	Stderr io.Writer
}
