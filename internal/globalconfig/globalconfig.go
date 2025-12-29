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
	err = os.MkdirAll(memshonkDir, 0700)
	if err != nil {
		return Config{}, fmt.Errorf("failed to create memshonk directory - %v", err)
	}

	return Config{
		DirPath: memshonkDir,
	}, nil
}

type Config struct {
	DirPath string
}
