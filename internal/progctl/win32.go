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
				BaseAddr:  uintptr(info.BaseAddress),
				EndAddr:   uintptr(info.BaseAddress) + info.RegionSize,
				AllocBase: uintptr(info.AllocationBase),
				Size:      uint64(info.RegionSize),
			}

			switch info.Type {
			case kernel32.MemImage:
				region.Type = memory.MemImage
			case kernel32.MemMapped:
				region.Type = memory.MemMapped
			case kernel32.MemPrivate:
				region.Type = memory.MemPrivate
			}

			switch info.State {
			case kernel32.MemCommit:
				region.State = memory.MemCommit
			case kernel32.MemReserve:
				region.State = memory.MemReserve
			case kernel32.MemFree:
				region.State = memory.MemFree
			}

			info.AllocationProtect &= ^(kernel32.PageGuard | kernel32.PageNoCache)

			switch info.AllocationProtect {
			case kernel32.PageExecute:
				region.Executable = true
			case kernel32.PageExecuteRead:
				region.Executable = true
				region.Readable = true
			case kernel32.PageExecuteReadWrite:
				region.Executable = true
				region.Readable = true
				region.Writeable = true
			case kernel32.PageExecuteWriteCopy:
				region.Executable = true
				region.Writeable = true
				region.Copyable = true
			case kernel32.PageNoAccess:
				// no access
			case kernel32.PageReadOnly:
				region.Readable = true
			case kernel32.PageReadWrite:
				region.Readable = true
				region.Writeable = true
			case kernel32.PageWriteCopy:
				region.Writeable = true
				region.Copyable = true
			}

			regions.Add(region)
			return nil
		})

	if err != nil {
		return memory.Regions{}, fmt.Errorf("failed to iterate over virtual memory - %w", err)
	}

	return regions, nil
}
