//go:build !plugins_disabled

package pluginsctl

import "github.com/SeungKang/memshonk/internal/plugins/libplugin"

func New(todoProcessPlaceholder interface{}) (*libplugin.Ctl, error) {
	return libplugin.NewCtl(todoProcessPlaceholder)
}
