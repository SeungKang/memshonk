package commands

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/SeungKang/memshonk/internal/apicompat"
	"github.com/SeungKang/memshonk/internal/memory"
)

var (
	errCommandNeedsTerminal = errors.New("this command requires a terminal, but the session does not provide a terminal")
)

func BuiltinCommands() []CommandSchema {
	return []CommandSchema{
		AttachCommandSchema(),
		DetachCommandSchema(),
		ReadCommandSchema(),
		WriteCommandSchema(),
		FindCommandSchema(),
		VmmapCommandSchema(),
		PluginsCommandSchema(),
		WatchCommandSchema(),
	}
}

type CommandSchema struct {
	Name      string
	Aliases   []string
	ShortHelp string
	LongHelp  string
	NonFlags  []NonFlagSchema
	Flags     []FlagSchema
	CreateFn  func(CommandConfig) (apicompat.Command, error)
}

type FlagSchema struct {
	Short      string
	Long       string
	Desc       string
	DataType   interface{}
	DefaultVal interface{}
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
	Flags    FlagFetcher
}

type ArgFetcher interface {
	FlagFetcher
	StringList(argID string) []string
}

type FlagFetcher interface {
	Bool(argID string) bool
	String(argID string) string
	Int(argID string) int
	Int64(argID string) int64
	Uint(argID string) uint
	Uint64(argID string) uint64
}

type HumanCommandResult string

func (o HumanCommandResult) String() string {
	return string(o)
}

func (o HumanCommandResult) Serialize() []byte {
	return []byte(o)
}

type UintptrCommandResult uintptr

func (o UintptrCommandResult) Uintptr() uintptr {
	return uintptr(o)
}

func (o UintptrCommandResult) Serialize() []byte {
	return []byte(fmt.Sprintf("%#x", o))
}

type UintptrListCommandResult []uintptr

func (o UintptrListCommandResult) Uintptrs() []uintptr {
	return []uintptr(o)
}

func (o UintptrListCommandResult) Serialize() []byte {
	buf := bytes.Buffer{}

	for i, u := range o {
		buf.WriteString(fmt.Sprintf("%#x", u))

		if i < len(o)-1 {
			buf.WriteString(", ")
		}
	}

	return buf.Bytes()
}

type MemoryPointerListCommandResult []memory.Pointer

func (o MemoryPointerListCommandResult) Pointers() []memory.Pointer {
	return []memory.Pointer(o)
}

func (o MemoryPointerListCommandResult) Serialize() []byte {
	buf := bytes.Buffer{}

	for i, u := range o {
		buf.WriteString(u.String())

		if i < len(o)-1 {
			buf.WriteString(", ")
		}
	}

	return buf.Bytes()
}
