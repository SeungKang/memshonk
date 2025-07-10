package plugins

import (
	"context"
	"errors"

	"github.com/SeungKang/memshonk/internal/events"
)

var (
	ErrPluginsDisabled      = errors.New("plugins are disabled")
	ErrPluginNotLoaded      = errors.New("plugin is not loaded (please check that its name is correct)")
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

	EnableDebug()

	DisableDebug()

	RunParser(id string, targetAddr uintptr) ([]byte, error)

	PrettyString(indent string) string
}

type PluginConfig struct {
	FilePath     string
	ExecOnReload []string
}

type Process interface {
	ReadFromAddr(addr uintptr, size uint64) ([]byte, error)
	WriteToAddr(addr uintptr, data []byte) error
}
