package pluginscompat

import (
	"context"

	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/SeungKang/memshonk/internal/plugins"
	"github.com/SeungKang/memshonk/internal/progctl"
)

var _ plugins.Process = (*ProcessCompatLayer)(nil)

func WrapProcess(progCtl progctl.Process) *ProcessCompatLayer {
	return &ProcessCompatLayer{
		proc: progCtl,
	}
}

type ProcessCompatLayer struct {
	proc progctl.Process
}

func (o ProcessCompatLayer) ReadFromAddr(addr uintptr, size uint64) ([]byte, error) {
	return o.proc.ReadFromAddr(context.Background(), memory.AbsoluteAddrPointer(addr), size)
}
