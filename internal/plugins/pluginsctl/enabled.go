//go:build !plugins_disabled

package pluginsctl

import (
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/plugins/libplugin"
)

func New(args plugins.CtlConfig) (*libplugin.Ctl, error) {
	return libplugin.NewCtl(args)
}
