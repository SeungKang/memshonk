package project

import (
	"github.com/SeungKang/memshonk/internal/ini"
)

const (
	libraryPathParam = "Library"
)

type Plugins struct {
	Libraries []string
}

type pluginsSchema struct {
	plugins *Plugins
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

			o.plugins.Libraries = append(o.plugins.Libraries, pathStr)

			return nil
		}, ini.SchemaRule{}
	default:
		return nil, ini.SchemaRule{}
	}
}

func (o *pluginsSchema) Validate() error {
	return nil
}
