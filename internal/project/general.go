package project

import (
	"github.com/SeungKang/memshonk/internal/ini"
)

const (
	exePathParam = "ExePath"
)

type General struct {
	ExePath string
}

type generalSchema struct {
	general *General
}

func (o *generalSchema) RequiredParams() []string {
	return []string{
		exePathParam,
	}
}

func (o *generalSchema) OnParam(paramName string) (func(*ini.Param) error, ini.SchemaRule) {
	if o.general == nil {
		o.general = &General{}
	}

	switch paramName {
	case exePathParam:
		return func(p *ini.Param) error {
			o.general.ExePath = p.Value

			return nil
		}, ini.SchemaRule{Limit: 1}
	default:
		return nil, ini.SchemaRule{}
	}
}

func (o *generalSchema) Validate() error {
	return nil
}
