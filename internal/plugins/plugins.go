package plugins

import (
	"context"
	"errors"

	"github.com/SeungKang/memshonk/internal/events"
)

var (
	ErrPluginsDisabled      = errors.New("plugins are disabled")
	ErrPluginNotLoaded      = errors.New("plugin is not loaded (please check that its name is correct)")
	ErrPluginUnloaded       = errors.New("plugin was unloaded")
	ErrExecOnReloadDisabled = errors.New("exec on reload is disabled")
)

type CtlConfig struct {
	Events  *events.Groups
	Process Process
}

type Ctl interface {
	Load(config PluginConfig) (Plugin, error)

	Plugin(name string) (Plugin, error)

	Reload(ctx context.Context, name string) error

	Unload(name string) error

	PrettyString(indent string) string
}

type Plugin interface {
	Name() string

	FilePath() string

	Version() uint16

	Description() string

	EnableDebug()

	DisableDebug()

	IterParsers(func(Parser) error) error

	IterCommands(func(Command) error) error

	PrettyString(indent string) string
}

type Parser interface {
	Name() string

	Run(ctx context.Context, targetAddr uintptr) ([]byte, error)
}

type Command interface {
	Name() string

	Run(ctx context.Context, args []string) ([]byte, error)
}

type PluginConfig struct {
	FilePath     string
	ExecOnReload []string
}

type Process interface {
	ReadFromAddr(addr uintptr, size uint64) ([]byte, error)
	WriteToAddr(addr uintptr, data []byte) error
}
