package progctl

import (
	"fmt"
	"os"
	"syscall"

	"github.com/SeungKang/memshonk/internal/kernel32"
	"github.com/SeungKang/memshonk/internal/memory"

	"github.com/Andoryuuta/kiwi"
)

var _ procMem = (*windowsProcMem)(nil)

func attachProcMem(pid int) (*windowsProcMem, error) {
	kiwiProc, err := kiwi.GetProcessByPID(pid)
	if err != nil {
		return nil, err
	}

	handle := syscall.Handle(kiwiProc.Handle)

	is32Bit, err := kernel32.IsProcess32Bit(handle)
	if err != nil {
		return nil, fmt.Errorf("failed to determine if process is 32 bit - %w",
			err)
	}

	osProc, err := os.FindProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to find process with PID: %d - %w",
			pid, err)
	}

	exitMon := newExitMonitor()

	go func() {
		_, err := osProc.Wait()
		exitMon.SetExited(err)
	}()

	return &windowsProcMem{
		kiwiProc: kiwiProc,
		handle:   handle,
		is32b:    is32Bit,
		exitMon:  exitMon,
	}, nil
}

type windowsProcMem struct {
	kiwiProc kiwi.Process
	handle   syscall.Handle
	is32b    bool
	exitMon  *ExitMonitor
}

func (o *windowsProcMem) ExitMonitor() *ExitMonitor {
	return o.exitMon
}

func (o *windowsProcMem) ReadBytes(addr uintptr, size int) ([]byte, error) {
	return o.kiwiProc.ReadBytes(addr, size)
}

func (o *windowsProcMem) WriteBytes(addr uintptr, b []byte) error {
	return o.kiwiProc.WriteBytes(addr, b)
}

func (o *windowsProcMem) ReadPtr(at uintptr) (uintptr, error) {
	if o.is32b {
		u32, err := o.kiwiProc.ReadUint32(at)
		return uintptr(u32), err
	} else {
		u64, err := o.kiwiProc.ReadUint64(at)
		return uintptr(u64), err
	}
}

func (o *windowsProcMem) Objects() (memory.MappedObjects, error) {
	objs := memory.MappedObjects{}

	// some modules appear more than once, we are just going to use the first
	// entry that has a non-zero base address :)
	// TODO add option to log weird stuff we are seeing, attach -v
	err := kernel32.IterProcessModules(
		o.handle,
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
		return memory.MappedObjects{}, fmt.Errorf("failed to iterate over process modules - %w", err)
	}

	objs.Sort()

	return objs, nil
}

func (o *windowsProcMem) Regions() (memory.Regions, error) {
	var regions memory.Regions

	err := kernel32.IterVirtualMemory(
		o.handle,
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
				region.Readable = true
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
				region.Readable = true
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

func (o *windowsProcMem) Close() error {
	o.exitMon.SetExited(ErrDetached)

	return syscall.CloseHandle(o.handle)
}
