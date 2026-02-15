package globalconfig

import (
	"fmt"
	"os"
	"path/filepath"
)

func (o *Config) SetupWorkspace(projectName string) (WorkspaceConfig, error) {
	wsConfig := o.ProjectWorkspaceConfig(projectName)

	err := os.MkdirAll(wsConfig.DirPath, 0o700)
	if err != nil {
		return WorkspaceConfig{}, fmt.Errorf("failed to create workspace directory at %q - %v",
			wsConfig.DirPath, err)
	}

	err = os.MkdirAll(wsConfig.SocketsDirPath, 0o700)
	if err != nil {
		return WorkspaceConfig{}, fmt.Errorf("failed to create sockets directory at %q - %v",
			wsConfig.SocketsDirPath, err)
	}

	err = os.MkdirAll(wsConfig.HistoryDirPath, 0o700)
	if err != nil {
		return WorkspaceConfig{}, fmt.Errorf("failed to create history directory at %q - %v",
			wsConfig.HistoryDirPath, err)
	}

	return wsConfig, nil
}

func (o *Config) ProjectWorkspaceConfig(projectName string) WorkspaceConfig {
	workspaceDirPath := filepath.Join(o.WorkspacesDirPath, projectName)

	socketsDirPath := filepath.Join(workspaceDirPath, "sockets")

	return WorkspaceConfig{
		DirPath:        workspaceDirPath,
		SocketsDirPath: socketsDirPath,
		SocketFilePath: filepath.Join(socketsDirPath, "socket.sock"),
		HistoryDirPath: filepath.Join(workspaceDirPath, "history"),
		globalConfig:   o,
	}
}

type WorkspaceConfig struct {
	DirPath        string
	SocketsDirPath string
	SocketFilePath string
	HistoryDirPath string
	globalConfig   *Config
}

func (o *WorkspaceConfig) HistoryFilePath(sessionID string) (string, bool) {
	if !o.globalConfig.HistoryFileEnabled {
		return "", false
	}

	var suffix string

	if sessionID != "default" {
		suffix = "-" + sessionID
	}

	return filepath.Join(o.HistoryDirPath, "history"+suffix), true
}
