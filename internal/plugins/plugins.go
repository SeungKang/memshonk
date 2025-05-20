package plugins

import "errors"

var (
	ErrPluginsDisabled = errors.New("plugins are disabled")
	ErrPluginNotLoaded = errors.New("plugin is not loaded (please check that its name is correct)")
)

type Ctl interface {
	Load(filePath string) (Plugin, error)

	Plugin(name string) (Plugin, error)

	Unload(Plugin) error

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

type CtlConfig struct {
	InitialPlugins []string

	Process Process
}

type Process interface {
	ReadFromAddr(addr uintptr, size uint64) ([]byte, error)
}
