package memory

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

func (o *Regions) Iter(fn func(i int, region Region)) error {
	for i, region := range o.regions {
		fn(i, region)
	}

	return nil
}

type Region struct {
	BaseAddress    uintptr
	AllocationBase uintptr
	RegionSize     uint64
	State          MemoryState
	Type           MemoryType

	Readable   bool
	Writeable  bool
	Executable bool
}
