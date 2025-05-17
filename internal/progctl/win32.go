package progctl

import (
	"fmt"
	"syscall"

	"github.com/SeungKang/memshonk/internal/kernel32"
	"github.com/SeungKang/memshonk/internal/memory"
)

func getRegions(procHandle uintptr) (memory.Regions, error) {
	var regions memory.Regions

	err := kernel32.IterVirtualMemory(syscall.Handle(procHandle),
		func(i int, info kernel32.MEMORY_BASIC_INFORMATION) error {
			region := memory.Region{
				BaseAddress:    uintptr(info.BaseAddress),
				AllocationBase: uintptr(info.AllocationBase),
				RegionSize:     uint64(info.RegionSize),
			}

			switch info.State {
			case kernel32.MemCommit:
				region.State = memory.MemCommit
			case kernel32.MemReserve:
				region.State = memory.MemReserve
			case kernel32.MemFree:
				region.State = memory.MemFree
			}

			switch info.Type {
			case kernel32.MemImage:
				region.Type = memory.MemImage
			case kernel32.MemMapped:
				region.Type = memory.MemMapped
			case kernel32.MemPrivate:
				region.Type = memory.MemPrivate
			}

			switch info.AllocationProtect {

			}

			regions.Add(region)
			return nil
		})

	if err != nil {
		return memory.Regions{}, fmt.Errorf("failed to iterate over virtual memory - %w", err)
	}

	return regions, nil
}
