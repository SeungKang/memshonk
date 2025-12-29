package project

import (
	"context"
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

func FromFilePath(filePath string, config globalconfig.Config) (*Project, error) {
	name := filepath.Base(filePath)
	dotIndex := strings.LastIndex(name, ".")
	if dotIndex > 0 {
		name = name[:dotIndex]
	}

	wsConfig, err := config.SetupWorkspace(&config, name)
	if err != nil {
		return nil, fmt.Errorf("failed to setup workspace - %v", err)
	}

	srcFn := func() (io.ReadCloser, error) {
		return os.Open(filePath)
	}

	file, err := srcFn()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	schemea := &projectSchema{
		project: &Project{
			name:     name,
			wsConfig: wsConfig,
			src:      srcFn,
		},
	}

	err = ini.ParseSchema(file, schemea)
	if err != nil {
		return nil, fmt.Errorf("failed to parse project - %w", err)
	}

	return schemea.project, nil
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
