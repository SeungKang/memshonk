package globalconfig

import (
	"fmt"
	"os"
	"path/filepath"
)

func (o *Config) SetupWorkspace(config *Config, projectName string) (WorkspaceConfig, error) {
	wsConfig := WorkspaceConfig{
		config: config,
	}

	err := wsConfig.socketFilePath(projectName)
	if err != nil {
		return wsConfig, fmt.Errorf("failed to setup socket file path - %v", err)
	}

	err = wsConfig.historyDir(projectName)
	if err != nil {
		return wsConfig, fmt.Errorf("failed to setup history dir - %v", err)
	}

	return wsConfig, nil
}

type WorkspaceConfig struct {
	DirPath        string
	SocketFilePath string
	HistoryDirPath string
	config         *Config
}

func (o *WorkspaceConfig) socketFilePath(projectName string) error {
	workspaceDir, err := o.createWorkspacesDir(projectName)
	if err != nil {
		return err
	}

	socketDir := filepath.Join(workspaceDir, "sockets")
	err = os.MkdirAll(socketDir, 0o700)
	if err != nil {
		return fmt.Errorf("failed to create sockets directory at %q - %v", socketDir, err)
	}

	o.SocketFilePath = filepath.Join(socketDir, "socket.sock")

	return nil
}

func (o *WorkspaceConfig) historyDir(projectName string) error {
	workspaceDir, err := o.createWorkspacesDir(projectName)
	if err != nil {
		return err
	}

	historyDir := filepath.Join(workspaceDir, "history")
	err = os.MkdirAll(historyDir, 0o700)
	if err != nil {
		return fmt.Errorf("failed to create history directory at %q - %v", workspaceDir, err)
	}

	o.HistoryDirPath = historyDir

	return nil
}

func (o *WorkspaceConfig) createWorkspacesDir(projectName string) (string, error) {
	workspaceDir := filepath.Join(o.config.DirPath, "workspaces", projectName)
	err := os.MkdirAll(workspaceDir, 0o700)
	if err != nil {
		return "", fmt.Errorf("failed to create workspaces directory - %v", err)
	}

	o.DirPath = workspaceDir

	return workspaceDir, nil
}
