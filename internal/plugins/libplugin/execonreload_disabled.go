//go:build !plugins_execonreload

package libplugin

import (
	"context"

	"github.com/SeungKang/memshonk/internal/plugins"
)

func execReload(context.Context, plugins.ReloadPluginArgs, plugins.PluginConfig) error {
	return plugins.ErrExecOnReloadDisabled
}
