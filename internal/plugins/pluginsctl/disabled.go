//go:build plugins_disabled

package pluginsctl

import (
	"github.com/SeungKang/memshonk/internal/plugins"
)

func New(todoProcessPlaceholder interface{}) (plugins.Ctl, error) {
	return nil, plugins.ErrPluginsDisabled
}
