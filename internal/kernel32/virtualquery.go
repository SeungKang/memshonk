package kernel32

import (
	"fmt"
	"syscall"
	"unsafe"
)

// Various memory type constants.
//
// See MEMORY_BASIC_INFORMATION documentation for details.
const (
	MemImage   uint32 = 0x1000000
	MemMapped  uint32 = 0x40000
	MemPrivate uint32 = 0x20000
)

// Various memory state constants.
//
// See MEMORY_BASIC_INFORMATION documentation for details.
const (
	MemCommit  uint32 = 0x1000
	MemFree    uint32 = 0x10000
	MemReserve uint32 = 0x2000
)

// Various memory protection constants.
//
// See also:
// https://learn.microsoft.com/en-us/windows/win32/memory/memory-protection-constants
const (
	PageExecute          uint32 = 0x10
	PageExecuteRead      uint32 = 0x20
	PageExecuteReadWrite uint32 = 0x40
	PageExecuteWriteCopy uint32 = 0x80
	PageNoAccess         uint32 = 0x01
	PageReadOnly         uint32 = 0x02
	PageReadWrite        uint32 = 0x04
	PageWriteCopy        uint32 = 0x08
	PageTargetsInvalid   uint32 = 0x40000000
	PageTargetsNoUpdate  uint32 = 0x40000000
)

const (
	PageGuard        uint32 = 0x100
	PageNoCache      uint32 = 0x200
	PageWriteCombine uint32 = 0x400
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
