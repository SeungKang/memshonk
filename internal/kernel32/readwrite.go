package kernel32

import (
	"encoding/binary"
	"fmt"
	"golang.org/x/sys/windows"
	"syscall"
)

// NeededProcessAccess is the combined windows process open flags needed for read and write functionality.
const NeededProcessAccess = windows.PROCESS_VM_READ | windows.PROCESS_VM_WRITE | windows.PROCESS_VM_OPERATION | windows.PROCESS_QUERY_INFORMATION

// GetReadWriteHandle returns a handle to the process with teh specified PID,
// granting read and write access
func GetReadWriteHandle(pid int) (syscall.Handle, error) {
	hnd, err := syscall.OpenProcess(NeededProcessAccess, false, uint32(pid))
	// hnd, ok := w32.OpenProcess(NeededProcessAccess, false, uint32(pid))
	if err != nil {
		return 0, err
	}

	return hnd, nil
}

func ReadProcessMemory(handle syscall.Handle, addr uintptr, byteSize uintptr) ([]byte, error) {
	if byteSize == 0 {
		return nil, nil
	}

	byteSlice := make([]byte, byteSize)
	numberOfBytesRead := uintptr(0)

	err := windows.ReadProcessMemory(
		windows.Handle(handle),
		addr,
		&byteSlice[0],
		byteSize,
		&numberOfBytesRead,
	)
	if err != nil {
		return nil, err
	}

	if numberOfBytesRead != byteSize {
		return nil, fmt.Errorf("ReadProcessMemory returned %d bytes instead of %d",
			numberOfBytesRead,
			byteSize)
	}

	return byteSlice, nil
}

func ReadPtr(handle syscall.Handle, addr uintptr, sizeBytes uint, order binary.ByteOrder) (uintptr, error) {
	switch sizeBytes {
	case 4:
		data, err := ReadProcessMemory(handle, addr, uintptr(sizeBytes))
		if err != nil {
			return 0, err
		}

		return uintptr(order.Uint32(data)), nil
	case 8:
		data, err := ReadProcessMemory(handle, addr, uintptr(sizeBytes))
		if err != nil {
			return 0, err
		}

		return uintptr(order.Uint64(data)), nil
	default:
		return 0, fmt.Errorf("unsupported number of bytes: %d", sizeBytes)
	}
}

func WriteProcessMemory(handle syscall.Handle, addr uintptr, data []byte) error {
	numberOfBytesWritten := uintptr(0)
	byteSize := uintptr(len(data))

	if byteSize == 0 {
		return fmt.Errorf("cannot write to zero byte slice")
	}

	err := windows.WriteProcessMemory(
		windows.Handle(handle),
		addr,
		&data[0],
		byteSize,
		&numberOfBytesWritten,
	)
	if err != nil {
		return err
	}

	if numberOfBytesWritten != byteSize {
		return fmt.Errorf("WriteProcessMemory returned %d bytes instead of %d",
			numberOfBytesWritten,
			byteSize)
	}

	return nil
}
