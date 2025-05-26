//go:build plugins_execreload

package libplugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/SeungKang/memshonk/internal/plugins"
)

func execReload(ctx context.Context, config plugins.PluginConfig) error {
	if len(config.ExecOnReload) == 0 {
		return nil
	}

	prog := exec.CommandContext(
		ctx,
		config.ExecOnReload[0],
		config.ExecOnReload[1:]...)

	prog.Stderr = os.Stderr
	prog.Stdout = os.Stdout
	prog.Dir = filepath.Dir(config.FilePath)

	err := prog.Run()
	if err != nil {
		return fmt.Errorf("exec failed for: %q - %w",
			prog.Args, err)
	}

	return nil
}
