//go:build !plugins_execreload

package libplugin

import (
	"context"

	"github.com/SeungKang/memshonk/internal/plugins"
)

func execReload(ctx context.Context, config plugins.PluginConfig) error {
	return plugins.ErrExecOnReloadDisabled
}
