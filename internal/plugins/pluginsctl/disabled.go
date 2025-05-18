//go:build plugins_disabled

package pluginsctl

import (
	"github.com/SeungKang/memshonk/internal/plugins"
)

func New(plugins.CtlConfig) (plugins.Ctl, error) {
	return nil, plugins.ErrPluginsDisabled
}
