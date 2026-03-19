//go:build plugins_execonreload

package libplugin

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/SeungKang/memshonk/internal/plugins"
)

func execReload(ctx context.Context, args plugins.ReloadPluginArgs, config plugins.PluginConfig) error {
	prog := exec.CommandContext(
		ctx,
		config.ExecOnReload[0],
		config.ExecOnReload[1:]...)

	prog.Stderr = args.Stderr
	prog.Stdout = args.Stdout
	prog.Dir = filepath.Dir(config.FilePath)

	err := prog.Run()
	if err != nil {
		return fmt.Errorf("exec failed for: %q - %w",
			prog.String(), err)
	}

	return nil
}
