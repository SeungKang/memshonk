package memory

import "errors"

const (
	MemTypeUnknown MemoryType = iota
	MemImage
	MemMapped
	MemPrivate
)

type MemoryType int

const (
	MemStateUnknown MemoryState = iota
	MemCommit
	MemFree
	MemReserve
)

type MemoryState int

type Regions struct {
	regions []Region
}

func (o *Regions) Add(region Region) {
	o.regions = append(o.regions, region)
}

func (o *Regions) Iter(fn func(i int, region Region) error) error {
	for i, region := range o.regions {
		err := fn(i, region)
		if err != nil {
			if errors.Is(err, ErrStopIterating) {
				return nil
			}

			return err
		}
	}

	return nil
}

type Region struct {
	BaseAddress    uintptr
	AllocationBase uintptr
	Size           uint64
	State          MemoryState
	Type           MemoryType

	Readable   bool
	Writeable  bool
	Executable bool
	Copyable   bool
}
