package commands

import (
	"context"
	"io"

	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/progctl"
)

type Command interface {
	Run(context.Context, IO, Session) error
}

type Session interface {
	Process() progctl.Process

	Plugins() (plugins.Ctl, bool)
}

type IO struct {
	Stdout io.Writer

	Stderr io.Writer
}
