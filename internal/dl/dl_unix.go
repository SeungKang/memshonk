package dl

import "github.com/ebitengine/purego"

func open(libFilePath string) (uintptr, error) {
	return purego.Dlopen(libFilePath, purego.RTLD_GLOBAL|purego.RTLD_NOW)
}

func sym(libraryHandle uintptr, symbolName string) (uintptr, error) {
	return purego.Dlsym(libraryHandle, symbolName)
}

func closel(libraryHandle uintptr) error {
	return purego.Dlclose(libraryHandle)
}
