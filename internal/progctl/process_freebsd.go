//go:build freebsd

package progctl

import (
	"fmt"

	"github.com/SeungKang/memshonk/internal/fbsdmaps"
	"github.com/SeungKang/memshonk/internal/memory"
)

func (o *unixProcess) Regions() (memory.Regions, error) {
	err := o.stopAndPtrace()
	if err != nil {
		return memory.Regions{}, fmt.Errorf("failed to ptrace process prior to getting regions - %w", err)
	}
	defer func() {
		o.ptrace.Detach()
	}()

	return fbsdmaps.Vmmap(o.ptrace)
}
