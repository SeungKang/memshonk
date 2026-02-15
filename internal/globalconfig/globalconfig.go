package globalconfig

import (
	"fmt"
	"os"
	"path/filepath"
)

func Setup() (Config, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return Config{}, fmt.Errorf("failed to get home dir - %v", err)
	}

	memshonkDir := filepath.Join(homeDir, ".memshonk")
	err = os.MkdirAll(memshonkDir, 0o700)
	if err != nil {
		return Config{}, fmt.Errorf("failed to create memshonk directory - %v", err)
	}

	return Config{
		DirPath: memshonkDir,

		WorkspacesDirPath: filepath.Join(memshonkDir, "workspaces"),

		HistoryFileEnabled: true, // TODO: Make configurable.
	}, nil
}

type Config struct {
	DirPath string

	WorkspacesDirPath string

	HistoryFileEnabled bool
}

func (o Config) Workspaces() ([]WorkspaceConfig, error) {
	projectsDirEntries, err := os.ReadDir(o.WorkspacesDirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workspaces dir entries - %w", err)
	}

	var wsConfigs []WorkspaceConfig

	for _, projectsEntry := range projectsDirEntries {
		if !projectsEntry.IsDir() {
			continue
		}

		wsConfig := o.ProjectWorkspaceConfig(projectsEntry.Name())

		wsConfigs = append(wsConfigs, wsConfig)
	}

	return wsConfigs, nil
}
