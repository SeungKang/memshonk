//go:build windows

package progctl

import (
	"encoding/binary"
	"errors"
	"fmt"
	"sort"
	"syscall"

	"github.com/SeungKang/memshonk/internal/kernel32"
	"github.com/SeungKang/memshonk/internal/memory"

	"golang.org/x/sys/windows"
)

func attach(config attachConfig) (*process, error) {
	handle, err := syscall.OpenProcess(
		kernel32.ProcessReadWriteRights,
		false,
		uint32(config.pid))
	if err != nil {
		return nil, fmt.Errorf("failed to get read-write handle - %w",
			err)
	}

	is32bit, err := kernel32.IsProcess32Bit(handle)
	if err != nil {
		_ = syscall.CloseHandle(handle)

		return nil, fmt.Errorf("failed to determine if process is 32 bit - %w",
			err)
	}

	var bits uint8
	if is32bit {
		bits = 32
	} else {
		bits = 64
	}

	regions, err := regionsForProcHandle(handle)
	if err != nil {
		_ = syscall.CloseHandle(handle)

		return nil, fmt.Errorf("failed to get memory regions - %w",
			err)
	}

	exeObj, err := regions.FirstObjectMatching(config.exeName)
	if err != nil {
		_ = syscall.CloseHandle(handle)

		return nil, fmt.Errorf("failed to get mapped object for exe - %w",
			err)
	}

	proc := &process{
		config: config,
		handle: handle,
		exeInfo: ExeInfo{
			Bits: bits,
			Obj:  exeObj,
		},
	}

	err = notifyOnExit(handle, config.exitMon)
	if err != nil {
		_ = syscall.CloseHandle(handle)

		return nil, fmt.Errorf("failed to setup process exit monitor - %w",
			err)
	}

	return proc, nil
}

func notifyOnExit(handle syscall.Handle, exitMon *ExitMonitor) error {
	cancelEvent, err := windows.CreateEvent(nil, 0, 0, nil)
	if err != nil {
		return fmt.Errorf("failed to create cancel event - %w", err)
	}

	go func() {
		<-exitMon.Done()

		windows.SetEvent(cancelEvent)
	}()

	go func() {
		processHandle := windows.Handle(handle)

		handles := []windows.Handle{processHandle, cancelEvent}

		event, err := windows.WaitForMultipleObjects(handles, false, windows.INFINITE)
		if err != nil {
			exitMon.SetExited(&ExitMonitorProcExitErr{
				Source:        "wait-for-multiple-objects",
				OptMonitorErr: fmt.Errorf("WaitForMultipleObjects failed - %w", err),
			})

			return
		}

		switch event {
		case windows.WAIT_OBJECT_0:
			// Process exited.
			var exitCode uint32

			err := windows.GetExitCodeProcess(processHandle, &exitCode)
			if err == nil {
				status := int64(exitCode)

				exitMon.SetExited(&ExitMonitorProcExitErr{
					Source:        "wait-for-multiple-objects",
					OptExitStatus: &status,
				})
			} else {
				exitMon.SetExited(&ExitMonitorProcExitErr{
					Source: "wait-for-multiple-objects",
				})
			}
		case windows.WAIT_OBJECT_0 + 1:
			// Cancelled.
		}
	}()

	return nil
}

type process struct {
	handle  syscall.Handle
	config  attachConfig
	exeInfo ExeInfo
}

func (o *process) SetMemoryMode(modeName string) error {
	o.config.memoryModeName = modeName
	return nil
}

func (o *process) isAlive() error {
	var exitStatus uint32
	err := syscall.GetExitCodeProcess(o.handle, &exitStatus)
	if err != nil {
		return fmt.Errorf("failed get exit code process - %s", err)
	}

	// https://learn.microsoft.com/en-us/windows/win32/api/processthreadsapi/nf-processthreadsapi-getexitcodeprocess
	// 259 STILL_ACTIVE
	if exitStatus == 259 {
		return nil
	}

	return fmt.Errorf("process exited with status: %d", exitStatus)
}

func (o *process) ExeInfo() ExeInfo {
	return o.exeInfo
}

func (o *process) ReadBytes(addr uintptr, sizeBytes uint64) ([]byte, error) {
	if o.config.memoryModeName != kernel32MemoryMode {
		return nil, unsupportedMemoryModeError(o.config.memoryModeName)
	}

	return kernel32.ReadProcessMemory(o.handle, addr, uintptr(sizeBytes))
}

func (o *process) WriteBytes(b []byte, addr uintptr) error {
	if o.config.memoryModeName != kernel32MemoryMode {
		return unsupportedMemoryModeError(o.config.memoryModeName)
	}

	return kernel32.WriteProcessMemory(o.handle, addr, b)
}

func (o *process) ReadPtr(at uintptr) (uintptr, error) {
	if o.config.memoryModeName != kernel32MemoryMode {
		return 0, unsupportedMemoryModeError(o.config.memoryModeName)
	}

	if o.exeInfo.Bits == 32 {
		return kernel32.ReadPtr(o.handle, at, 4, binary.LittleEndian)
	} else {
		return kernel32.ReadPtr(o.handle, at, 8, binary.LittleEndian)
	}
}

func (o *process) Regions() (memory.Regions, error) {
	return regionsForProcHandle(o.handle)
}

func regionsForProcHandle(procHandle syscall.Handle) (memory.Regions, error) {
	objs, err := objectsForProcHandle(procHandle)
	if err != nil {
		return memory.Regions{}, fmt.Errorf("failed to get objects - %w", err)
	}

	var regions memory.Regions

	err = kernel32.IterVirtualMemory(
		procHandle,
		func(i int, info kernel32.MEMORY_BASIC_INFORMATION) error {
			region := memBasicInfoToRegion(info)

			obj, hasIt := objs.ContainsRegion(region.BaseAddr, region.EndAddr)
			if hasIt {
				region.Parent = obj.ToMeta()

				obj.used = true
			}

			regions.Add(region)

			return nil
		})
	if err != nil {
		return memory.Regions{}, fmt.Errorf("failed to iterate over virtual memory - %w", err)
	}

	err = objs.IterUnused(func(obj MappedObject) error {
		regions.Add(memory.Region{
			BaseAddr:   obj.BaseAddr,
			EndAddr:    obj.EndAddr,
			State:      memory.MemCommit,
			Type:       memory.MemImage,
			Size:       obj.Size,
			Readable:   true,
			Writeable:  true,
			Executable: true,
			Copyable:   true,
			Parent:     obj.ToMeta(),
		})

		return nil
	})
	if err != nil {
		return memory.Regions{}, fmt.Errorf("failed to iterate over unused objects - %w", err)
	}

	return regions, nil
}

func objectsForProcHandle(procHandle syscall.Handle) (MappedObjects, error) {
	objs := MappedObjects{}

	objectID := memory.ObjectID(0)

	// some modules appear more than once, we are just going to use the first
	// entry that has a non-zero base address :)
	// TODO add option to log weird stuff we are seeing, attach -v
	err := kernel32.IterProcessModules(
		procHandle,
		func(_ int, _ uint, module kernel32.Module) error {
			if module.BaseAddr == 0 {
				return nil
			}

			err := objs.Add(MappedObject{
				ID:       objectID,
				Filepath: module.Filepath,
				Filename: module.Filename,
				BaseAddr: module.BaseAddr,
				EndAddr:  module.EndAddr,
				Size:     module.Size,
			})
			if err != nil {
				return fmt.Errorf("failed to add module to memory mapped objects list - %w", err)
			}

			objectID++

			return nil
		})
	if err != nil {
		return MappedObjects{}, fmt.Errorf("failed to iterate over process modules - %w", err)
	}

	objs.Sort()

	return objs, nil
}

func memBasicInfoToRegion(info kernel32.MEMORY_BASIC_INFORMATION) memory.Region {
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

	info.Protect &= ^(kernel32.PageGuard | kernel32.PageNoCache)

	switch info.Protect {
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

	return region
}

func (o *process) Suspend() error {
	return nil
}

func (o *process) Resume() error {
	return nil
}

func (o *process) Close() error {
	return syscall.CloseHandle(o.handle)
}

type MappedObjects struct {
	objects []MappedObject
}

func (o *MappedObjects) Add(object MappedObject) error {
	if object.Filename == "" {
		return errors.New("object name is empty string")
	}

	o.objects = append(o.objects, object)

	return nil
}

func (o *MappedObjects) Len() int {
	return len(o.objects)
}

func (o *MappedObjects) Less(i, j int) bool {
	return o.objects[i].BaseAddr < o.objects[j].EndAddr
}

func (o *MappedObjects) Swap(i, j int) {
	o.objects[i], o.objects[j] = o.objects[j], o.objects[i]
}

func (o *MappedObjects) Sort() {
	sort.Sort(o)
}

func (o *MappedObjects) ContainsRegion(baseAddr uintptr, endAddr uintptr) (*MappedObject, bool) {
	// This code is based on work by Stackoverflow user OneOfOne:
	// https://stackoverflow.com/a/39750394
	ln := o.Len()

	i := sort.Search(ln, func(i int) bool {
		return endAddr <= o.objects[i].EndAddr
	})

	if i < ln {
		it := &o.objects[i]

		if baseAddr >= it.BaseAddr && endAddr <= it.EndAddr {
			return it, true
		}
	}

	return nil, false
}

func (o *MappedObjects) IterUnused(fn func(MappedObject) error) error {
	for _, obj := range o.objects {
		if !obj.used {
			err := fn(obj)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

type MappedObject struct {
	ID       memory.ObjectID
	Filepath string
	Filename string
	BaseAddr uintptr
	EndAddr  uintptr
	Size     uint64
	used     bool
}

func (o MappedObject) ToMeta() memory.ObjectMeta {
	// TODO: Windows modules also have a size
	// and their own base / end addrs.
	return memory.ObjectMeta{
		IsSet:    true,
		ID:       o.ID,
		FilePath: o.Filepath,
		FileName: o.Filename,
	}
}
