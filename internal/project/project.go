package project

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/SeungKang/memshonk/internal/globalconfig"
	"github.com/SeungKang/memshonk/internal/ini"
	"github.com/SeungKang/memshonk/internal/shvars"
)

type ProjectConfig struct {
	GlobalConf globalconfig.Config
	GlobalVars *shvars.Variables
}

func EmptyForExePath(exeFilePath string, projConfig ProjectConfig) (*Project, error) {
	absExeFilePath, err := exeAbsPath(exeFilePath)
	if err != nil {
		return nil, err
	}

	projectName := filepath.Base(absExeFilePath)

	wsConfig, err := projConfig.GlobalConf.SetupWorkspace(projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to setup workspace - %v", err)
	}

	project := &Project{
		name:     projectName,
		wsConfig: wsConfig,
		general: General{
			ExePath: absExeFilePath,
		},
		variables: Variables{
			vars: projConfig.GlobalVars,
		},
	}

	return project, nil
}

func FromFilePath(projectFilePath string, projConfig ProjectConfig) (*Project, error) {
	projectName := filepath.Base(projectFilePath)
	dotIndex := strings.LastIndex(projectName, ".")
	if dotIndex > 0 {
		projectName = projectName[:dotIndex]
	}

	wsConfig, err := projConfig.GlobalConf.SetupWorkspace(projectName)
	if err != nil {
		return nil, fmt.Errorf("failed to setup workspace - %v", err)
	}

	srcFn := func() (io.ReadCloser, error) {
		return os.Open(projectFilePath)
	}

	file, err := srcFn()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	schemea := &projectSchema{
		project: &Project{
			name:     projectName,
			wsConfig: wsConfig,
			src:      srcFn,
			variables: Variables{
				vars: projConfig.GlobalVars,
			},
		},
	}

	err = ini.ParseSchema(file, schemea)
	if err != nil {
		return nil, fmt.Errorf("failed to parse project - %w", err)
	}

	exeAbsFilePath, err := exeAbsPath(schemea.project.general.ExePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get exe absolute path - %w", err)
	}

	schemea.project.general.ExePath = exeAbsFilePath

	return schemea.project, nil
}

func exeAbsPath(exeFilePath string) (string, error) {
	absExePath, err := filepath.Abs(exeFilePath)
	if err != nil {
		return "", err
	}

	resolved, readLinkErr := os.Readlink(absExePath)
	if readLinkErr == nil {
		resolvedAbs, err := filepath.Abs(resolved)
		if err != nil {
			return "", fmt.Errorf("executable file path is a symlink which points to: '%s' - please specify that path instead of the symlink",
				resolved)

		}

		if resolvedAbs != absExePath {
			return "", fmt.Errorf("executable file path is a symlink which points to: '%s' - please specify that path instead of the symlink",
				resolvedAbs)
		}
	}

	return absExePath, nil
}

type Project struct {
	name      string
	wsConfig  globalconfig.WorkspaceConfig
	src       func() (io.ReadCloser, error)
	rwMu      sync.RWMutex
	general   General
	variables Variables
	plugins   Plugins
}

func (o *Project) Name() string {
	return o.name
}

func (o *Project) WorkspaceConfig() globalconfig.WorkspaceConfig {
	return o.wsConfig
}

func (o *Project) Reload(context.Context) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.src == nil {
		return errors.New("project does not support reloading")
	}

	src, err := o.src()
	if err != nil {
		return fmt.Errorf("failed to get project io.Reader - %w", err)
	}
	defer src.Close()

	next := &Project{
		src: o.src,
	}

	schemea := &projectSchema{project: next}

	err = ini.ParseSchema(src, schemea)
	if err != nil {
		return fmt.Errorf("failed to re-parse project - %w", err)
	}

	o.general = schemea.project.general
	o.variables = schemea.project.variables
	o.plugins = schemea.project.plugins

	return nil
}

func (o *Project) General() General {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return o.general
}

func (o *Project) Variables() *shvars.Variables {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return o.variables.vars
}

func (o *Project) Plugins() Plugins {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	return o.plugins
}
