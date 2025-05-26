package project

import (
	"strings"

	"github.com/SeungKang/memshonk/internal/ini"
	"github.com/SeungKang/memshonk/internal/plugins"
)

const (
	libraryPathParam        = "Library"
	execOnPluginReloadParam = "ExecOnReload"
)

type Plugins struct {
	Libraries []plugins.PluginConfig
}

func (o Plugins) LibraryPaths() []string {
	paths := make([]string, len(o.Libraries))

	for i := range o.Libraries {
		paths[i] = o.Libraries[i].FilePath
	}

	return paths
}

type PluginConfig struct {
	FilePath     string
	ExecOnReload []string
}

type pluginsSchema struct {
	plugins *Plugins
	current plugins.PluginConfig
}

func (o *pluginsSchema) RequiredParams() []string {
	return nil
}

func (o *pluginsSchema) OnParam(paramName string) (func(*ini.Param) error, ini.SchemaRule) {
	if o.plugins == nil {
		o.plugins = &Plugins{}
	}

	switch paramName {
	case libraryPathParam:
		return func(p *ini.Param) error {
			pathStr, err := replaceMagicStrings(p.Value)
			if err != nil {
				return err
			}

			o.current.FilePath = pathStr

			return nil
		}, ini.SchemaRule{Limit: 1}
	case execOnPluginReloadParam:
		return func(p *ini.Param) error {
			replaced, err := replaceMagicStrings(p.Value)
			if err != nil {
				return err
			}

			o.current.ExecOnReload = strings.Split(replaced, " ")

			return nil
		}, ini.SchemaRule{Limit: 1}
	default:
		return nil, ini.SchemaRule{}
	}
}

func (o *pluginsSchema) Validate() error {
	o.plugins.Libraries = append(o.plugins.Libraries, o.current)

	return nil
}
