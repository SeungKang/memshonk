package commands

import (
	"context"
	"io"

	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/progctl"
)

func BuiltinCommands() []CommandSchema {
	return []CommandSchema{
		PluginsCommandSchema(),
	}
}

type CommandSchema struct {
	Name      string
	Aliases   []string
	ShortHelp string
	LongHelp  string
	NonFlags  []NonFlagSchema
	Flags     []FlagSchema
	CreateFn  func(CommandConfig) (Command, error)
}

type FlagSchema struct {
	Short       string
	Long        string
	Description string
	DataType    interface{}
	DefaultVal  interface{}
}

type NonFlagSchema struct {
	Name     string
	Desc     string
	DataType interface{}
	DefValue interface{}
}

type ArgDataTypeSchema struct {
	Type interface{}
}

type CommandConfig struct {
	NonFlags ArgFetcher
	Flags    ArgFetcher
}

type ArgFetcher interface {
	String(argID string) string
	Int(argID string) int
	Int64(argID string) int64
	Uint(argID string) uint
	Uint64(argID string) uint64
}

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
