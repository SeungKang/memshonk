package memory

import (
	"errors"
	"fmt"
	"sort"
)

type MappedObjects struct {
	objects []MappedObject
	byName  map[string]int
}

func (o *MappedObjects) Add(object MappedObject) error {
	if object.Filename == "" {
		return errors.New("object name is empty string")
	}

	_, hasIt := o.Has(object.Filename)
	if hasIt {
		return fmt.Errorf("object already present: %q", object.Filename)
	}

	if o.byName == nil {
		o.byName = make(map[string]int)
	}

	o.objects = append(o.objects, object)
	o.byName[object.Filename] = len(o.objects) - 1

	return nil
}

func (o *MappedObjects) Has(name string) (MappedObject, bool) {
	i, hasIt := o.byName[name]
	if hasIt {
		return o.objects[i], true
	}

	return MappedObject{}, hasIt
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

func (o *MappedObjects) HasAddr(addr uintptr) (*MappedObject, bool) {
	// This code is based on work by Stackoverflow user OneOfOne:
	// https://stackoverflow.com/a/39750394
	ln := o.Len()

	i := sort.Search(ln, func(i int) bool {
		return addr < o.objects[i].EndAddr
	})

	if i < ln {
		it := &o.objects[i]

		if addr >= it.BaseAddr && addr < it.EndAddr {
			return it, true
		}
	}

	return nil, false
}

func (o *MappedObjects) ContainsRegion(region Region) (*MappedObject, bool) {
	// This code is based on work by Stackoverflow user OneOfOne:
	// https://stackoverflow.com/a/39750394
	ln := o.Len()

	i := sort.Search(ln, func(i int) bool {
		return region.EndAddr <= o.objects[i].EndAddr
	})

	if i < ln {
		it := &o.objects[i]

		if region.BaseAddr >= it.BaseAddr && region.EndAddr <= it.EndAddr {
			return it, true
		}
	}

	return nil, false
}

func (o *MappedObjects) IterObjects(fn func(MappedObject) error) error {
	for _, obj := range o.objects {
		obj := obj

		err := fn(obj)
		if err != nil {
			if errors.Is(err, ErrStopIterating) {
				return nil
			}

			return err
		}
	}

	return nil
}

type MappedObject struct {
	Filepath string
	Filename string
	BaseAddr uintptr
	EndAddr  uintptr
	Size     uint64
}

func (o *MappedObject) ContainsAddr(addr uintptr) bool {
	return addr >= o.BaseAddr && addr < o.EndAddr
}

func (o *MappedObject) String() string {
	return fmt.Sprintf("%#012x-%#012x %#08x %s",
		o.BaseAddr, o.EndAddr, o.Size, o.Filepath)
}
