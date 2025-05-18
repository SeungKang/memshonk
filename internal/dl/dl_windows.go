//go:build windows

package dl

import "syscall"

func open(libraryFilePath string) (uintptr, error) {
	syscallHandle, err := syscall.LoadLibrary(libraryFilePath)
	if err != nil {
		return 0, err
	}

	return uintptr(syscallHandle), nil
}

func sym(libraryHandle uintptr, symbolName string) (uintptr, error) {
	handle, err := syscall.GetProcAddress(
		syscall.Handle(libraryHandle),
		symbolName)
	if err != nil {
		return 0, err
	}

	return uintptr(handle), nil
}

func closel(libraryHandle uintptr) error {
	return syscall.FreeLibrary(syscall.Handle(libraryHandle))
}
