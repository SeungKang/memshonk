package progctl

import (
	"fmt"
	"syscall"

	"github.com/SeungKang/memshonk/internal/kernel32"
	"github.com/SeungKang/memshonk/internal/memory"
)

func getModules(exeName string, procHandle uintptr) (memory.MappedObject, memory.MappedObjects, error) {
	objs := memory.MappedObjects{}

	// some modules appear more than once, we are just going to use the first
	// entry that has a non-zero base address :)
	// TODO add option to log weird stuff we are seeing, attach -v
	err := kernel32.IterProcessModules(
		syscall.Handle(procHandle),
		func(_ int, _ uint, module kernel32.Module) error {
			if module.BaseAddr == 0 {
				return nil
			}

			_, alreadyPresent := objs.Has(module.Filename)
			if alreadyPresent {
				return nil
			}

			err := objs.Add(memory.MappedObject{
				Filepath: module.Filepath,
				Filename: module.Filename,
				BaseAddr: module.BaseAddr,
				EndAddr:  module.EndAddr,
				Size:     module.Size,
			})
			if err != nil {
				return fmt.Errorf("failed to add module to memory mapped objects list - %w", err)
			}

			return nil
		})
	if err != nil {
		return memory.MappedObject{}, memory.MappedObjects{}, fmt.Errorf("failed to iterate over process modules - %w", err)
	}

	exeModule, found := objs.Has(exeName)
	if !found {
		return memory.MappedObject{}, memory.MappedObjects{}, fmt.Errorf("failed to find exe module for: %q", exeName)
	}

	objs.Sort()

	return exeModule, objs, nil
}

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
