package memory

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
)

const (
	MemTypeUnknown MemoryType = iota
	MemImage
	MemMapped
	MemPrivate
)

type MemoryType int

func (o MemoryType) String() string {
	switch o {
	case MemImage:
		return "image"
	case MemMapped:
		return "mapped"
	case MemPrivate:
		return "private"
	default:
		return "unknown"
	}
}

func (o MemoryType) Letter() byte {
	switch o {
	case MemImage:
		return 'I'
	case MemMapped:
		return 'M'
	case MemPrivate:
		return 'P'
	default:
		return 'U'
	}
}

const (
	MemStateUnknown MemoryState = iota
	MemCommit
	MemFree
	MemReserve
)

type MemoryState int

func (o MemoryState) String() string {
	switch o {
	case MemCommit:
		return "commit"
	case MemFree:
		return "free"
	case MemReserve:
		return "reserve"
	default:
		return "unknown"
	}
}

func (o MemoryState) Letter() byte {
	switch o {
	case MemCommit:
		return 'c'
	case MemFree:
		return 'f'
	case MemReserve:
		return 'r'
	default:
		return 'u'
	}
}

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

func (o Regions) Len() int {
	return len(o.regions)
}

func (o Regions) Less(i, j int) bool {
	return o.regions[i].BaseAddress < o.regions[j].EndAddr
}

func (o Regions) Swap(i, j int) {
	o.regions[i], o.regions[j] = o.regions[j], o.regions[i]
}

func (o Regions) Sort() {
	sort.Sort(o)
}

func (o Regions) ByAddr(addr uintptr) (*Region, bool) {
	// This code is based on work by Stackoverflow user OneOfOne:
	// https://stackoverflow.com/a/39750394
	ln := o.Len()

	i := sort.Search(ln, func(i int) bool {
		return addr <= o.regions[i].EndAddr
	})

	if i < ln {
		it := &o.regions[i]
		if addr >= it.BaseAddress && addr <= it.EndAddr {
			return it, true
		}
	}

	return nil, false
}

type Region struct {
	BaseAddress    uintptr
	EndAddr        uintptr
	AllocationBase uintptr
	Size           uint64
	State          MemoryState
	Type           MemoryType

	Readable   bool
	Writeable  bool
	Executable bool
	Copyable   bool
}

func (o Region) Unaccessible() bool {
	return !o.Readable && !o.Writeable && !o.Executable
}

func (o Region) String() string {
	buf := bytes.Buffer{}

	buf.WriteString(fmt.Sprintf("%#012x-%#012x (allocb: %#012x) ",
		o.BaseAddress,
		o.EndAddr,
		o.AllocationBase))

	if o.Readable {
		buf.WriteByte('r')
	} else {
		buf.WriteByte('-')
	}

	if o.Writeable {
		buf.WriteByte('w')
	} else {
		buf.WriteByte('-')
	}

	if o.Executable {
		buf.WriteByte('x')
	} else {
		buf.WriteByte('-')
	}

	buf.WriteByte(' ')

	if o.Copyable {
		buf.WriteByte('C')
	} else {
		buf.WriteByte('-')
	}

	buf.WriteString(fmt.Sprintf(" %#012x (%s, %s)",
		o.Size,
		o.Type.String(),
		o.State.String()))

	return buf.String()
}
