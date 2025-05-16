package project

import (
	"github.com/SeungKang/memshonk/internal/ini"
)

// Various section names.
const (
	generalSectionName        = "General"
	variablesSectionName      = "Variables"
	variablesSectionNameShort = "Vars"
)

type projectSchema struct {
	project *Project
}

func (o *projectSchema) Rules() ini.ParserRules {
	return ini.ParserRules{
		AllowGlobalParams: false,
		RequiredSections: []string{
			generalSectionName,
		},
	}
}

func (o *projectSchema) OnGlobalParam(paramName string) (func(*ini.Param) error, ini.SchemaRule) {
	return nil, ini.SchemaRule{}
}

func (o *projectSchema) OnSection(sectionName string, canconicalName string) (func() (ini.SectionSchema, error), ini.SchemaRule) {
	switch sectionName {
	case generalSectionName:
		return func() (ini.SectionSchema, error) {
			return &generalSchema{
				general: &o.project.general,
			}, nil
		}, ini.SchemaRule{Limit: 1}
	case variablesSectionName, variablesSectionNameShort:
		return func() (ini.SectionSchema, error) {
			return &variablesSchema{
				variables: &o.project.variables,
			}, nil
		}, ini.SchemaRule{Limit: 1}
	default:
		return nil, ini.SchemaRule{}
	}
}

func (o *projectSchema) Validate() error {
	return nil
}
