package project

import (
	"fmt"
	"os"
	"strings"

	"github.com/SeungKang/memshonk/internal/ini"
)

// Various section names.
const (
	generalSectionName        = "General"
	variablesSectionName      = "Variables"
	variablesSectionNameShort = "Vars"
	pluginsSectionName        = "Plugins"
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
	case pluginsSectionName:
		return func() (ini.SectionSchema, error) {
			return &pluginsSchema{
				plugins: &o.project.plugins,
			}, nil
		}, ini.SchemaRule{Limit: 1}
	default:
		return nil, ini.SchemaRule{}
	}
}

func (o *projectSchema) Validate() error {
	return nil
}

func replaceMagicStrings(str string) (string, error) {
	str, err := innerReplaceMagicStrings(str)
	if err != nil {
		return "", fmt.Errorf("failed to replace magic strings - %w", err)
	}

	return str, nil
}

// innerReplaceMagicStrings exists so we can write a useful error message
// in a wrapper function and not need to repeat that error over and over.
func innerReplaceMagicStrings(str string) (string, error) {
	if strings.HasPrefix(str, "~") {
		homeDirPath, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory - %w", err)
		}

		str = homeDirPath + str[1:]
	}

	return str, nil
}
