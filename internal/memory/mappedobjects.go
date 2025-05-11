package memory

import (
	"errors"
	"fmt"
)

type MappedObjects struct {
	namesToObjects map[string]MappedObject
}

func (o *MappedObjects) Add(object MappedObject) error {
	if object.Filename == "" {
		return errors.New("object name is empty string")
	}

	_, hasIt := o.Has(object.Filename)
	if hasIt {
		return fmt.Errorf("object already present: %q", object.Filename)
	}

	if o.namesToObjects == nil {
		o.namesToObjects = make(map[string]MappedObject)
	}

	o.namesToObjects[object.Filename] = object

	return nil
}

func (o *MappedObjects) Has(name string) (MappedObject, bool) {
	obj, hasIt := o.namesToObjects[name]

	return obj, hasIt
}

type MappedObject struct {
	Filepath string
	Filename string
	BaseAddr uintptr
	EndAddr  uintptr
	Size     uint64
}
