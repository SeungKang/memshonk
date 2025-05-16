package project

import (
	"fmt"

	"github.com/SeungKang/memshonk/internal/ini"
	"github.com/SeungKang/memshonk/internal/shvars"
)

type Variables struct {
	vars *shvars.Variables
}

type variablesSchema struct {
	variables *Variables
}

func (o *variablesSchema) RequiredParams() []string {
	return nil
}

func (o *variablesSchema) OnParam(paramName string) (func(*ini.Param) error, ini.SchemaRule) {
	if o.variables == nil {
		o.variables = &Variables{}
	}

	if o.variables.vars == nil {
		o.variables.vars = &shvars.Variables{}
	}

	return func(p *ini.Param) error {
		err := o.variables.vars.Set(p.Name, p.Value)
		if err != nil {
			return fmt.Errorf("failed to set project variable for %q - %w",
				p.Name, err)
		}

		return nil
	}, ini.SchemaRule{Limit: 1}
}

func (o *variablesSchema) Validate() error {
	return nil
}
