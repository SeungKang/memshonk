package dl

import (
	"errors"
	"fmt"

	"github.com/ebitengine/purego"
)

func Open(libraryFilePath string) (*Library, error) {
	handle, err := open(libraryFilePath)
	if err != nil {
		return nil, err
	}

	return &Library{
		handle: handle,
	}, nil
}

type Library struct {
	handle uintptr
}

func (o *Library) Func(funcName string, ptrToGoFn interface{}) error {
	ptr, err := o.Sym(funcName)
	if err != nil {
		return fmt.Errorf("sym failed - %w", err)
	}

	if ptr == 0 {
		return errors.New("symbol points to null")
	}

	purego.RegisterFunc(ptrToGoFn, ptr)

	return nil
}

func (o *Library) Sym(symbolName string) (uintptr, error) {
	return sym(o.handle, symbolName)
}

func (o *Library) Release() error {
	return closel(o.handle)
}

// NewCallback is a wrapper for purego.NewCallback.
func NewCallback(goFn interface{}) (uintptr, error) {
	return purego.NewCallback(goFn), nil
}
