//go:build linux

package progctl

import (
	"github.com/SeungKang/memshonk/internal/linuxmaps"
	"github.com/SeungKang/memshonk/internal/memory"
)

func (o *unixProcess) Regions() (memory.Regions, error) {
	return linuxmaps.Vmmap(o.pid)
}
