package plugins

import "errors"

var (
	ErrPluginsDisabled = errors.New("plugins are disabled")
)

type Ctl interface {
	Load(filePath string) (Plugin, error)

	Get(name string) (Plugin, error)

	PrettyString(indent string) string
}

type Plugin interface {
	Name() string

	FilePath() string

	Version() uint16

	EnableDebug()

	DisableDebug()

	Parser(name string) (ParserPlugin, bool)

	PrettyString(indent string) string
}

type ParserPlugin interface {
	Run(addr uintptr) ([]byte, error)
}

type CtlConfig struct {
	InitialPlugins []string

	Process Process
}

type Process interface {
	ReadFromAddr(addr uintptr, size uint64) ([]byte, error)
}
