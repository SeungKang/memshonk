//go:build freebsd

package progctl

import (
	"context"
	"fmt"

	"github.com/SeungKang/memshonk/internal/fbsdmaps"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/ptrace"
)

func (o *process) Regions() (memory.Regions, error) {
	if o.optPtrace == nil {
		return memory.Regions{}, errPtraceNotEnabled
	}

	needToResume := false

	if !o.stopped {
		needToResume = true

		err := o.Suspend()
		if err != nil {
			return memory.Regions{}, fmt.Errorf("failed to suspend process prior to getting regions - %w", err)
		}
	}

	var regions memory.Regions

	regionsErr := o.optPtrace.Do(context.Background(), func(_ context.Context, pt *ptrace.Tracer) error {
		var err error
		regions, err = fbsdmaps.Vmmap(pt)
		return err
	})

	if needToResume {
		err := o.Resume()
		if err != nil {
			return memory.Regions{}, fmt.Errorf("failed to resume process after getting regions - %w",
				err)
		}
	}

	return regions, regionsErr
}
