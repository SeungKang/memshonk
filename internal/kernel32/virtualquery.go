package kernel32

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	pVirtualQueryEx = kernel32.NewProc("VirtualQueryEx")
)

// IterVirtualMemory iterates over a process's virtual memory.
//
// The process handle must be opened with:
//
//	PROCESS_VM_READ | PROCESS_QUERY_INFORMATION
//
// Based on work by Stackoverflow user Jerry Coffin:
// https://stackoverflow.com/a/3313700
func IterVirtualMemory(handle syscall.Handle, fn func(i int, info MEMORY_BASIC_INFORMATION) error) error {
	var offset uintptr = 0
	i := 0

	for {
		var info MEMORY_BASIC_INFORMATION

		err := VirtualQueryEx(handle, offset, &info)
		if err != nil {
			if i == 0 {
				return fmt.Errorf("VirtualQueryEx failed - %w", err)
			}

			return nil
		}

		err = fn(i, info)
		if err != nil {
			return fmt.Errorf("iterator function returned an error - %w", err)
		}

		offset += uintptr(info.RegionSize)

		i++
	}
}

// VirtualQueryEx calls VirtualQueryEx for a process's handle.
//
// The process handle must be opened with:
//
//	PROCESS_VM_READ | PROCESS_QUERY_INFORMATION
func VirtualQueryEx(hProcess syscall.Handle, lpAddress uintptr, lpBuffer *MEMORY_BASIC_INFORMATION) error {
	dwLength := unsafe.Sizeof(*lpBuffer)

	_, _, err := pVirtualQueryEx.Call(
		uintptr(hProcess),                 // hProcess
		uintptr(lpAddress),                // lpAddress
		uintptr(unsafe.Pointer(lpBuffer)), // lpBuffer
		uintptr(dwLength))                 // dwLength
	if isError(err) {
		return err
	}

	return nil
}

// https://learn.microsoft.com/en-us/windows/win32/api/winnt/ns-winnt-memory_basic_information
type MEMORY_BASIC_INFORMATION struct {
	BaseAddress       unsafe.Pointer // PVOID
	AllocationBase    unsafe.Pointer // PVOID
	AllocationProtect uint32         // DWORD
	PartitionId       uint16         // WORD
	RegionSize        uintptr        // SIZE_T
	State             uint32         // DWORD
	Protect           uint32         // DWORD
	Type              uint32         // DWORD
}
