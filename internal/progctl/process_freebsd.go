//go:build freebsd

package progctl

import (
	"fmt"

	"github.com/SeungKang/memshonk/internal/fbsdmaps"
	"github.com/SeungKang/memshonk/internal/memory"
)

func (o *unixProcess) Regions() (memory.Regions, error) {
	needToResume := false

	if !o.stopped {
		needToResume = true

		err := o.Suspend()
		if err != nil {
			return memory.Regions{}, fmt.Errorf("failed to suspend process prior to getting regions - %w", err)
		}
	}

	regions, regionsErr := fbsdmaps.Vmmap(o.ptrace)

	if needToResume {
		err := o.Resume()
		if err != nil {
			return memory.Regions{}, fmt.Errorf("failed to resume process after getting regions - %w",
				err)
		}
	}

	return regions, regionsErr
}
