package memory

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrStopIterating = errors.New("stop iterating")
)

const (
	MemTypeUnknown MemoryType = iota
	MemImage
	MemMapped
	MemPrivate
	MemStack
	MemHeap
	MemVvar
	MemVdso
	MemAnon
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
	case MemStack:
		return "stack"
	case MemHeap:
		return "heap"
	case MemVvar:
		return "vvar"
	case MemVdso:
		return "vdso"
	case MemAnon:
		return "anon"
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
	objects []Object
	nonobjs []*Region
}

type Object struct {
	ID       ObjectID
	BaseAddr uintptr
	EndAddr  uintptr
	Name     string
	Path     string
	Regions  []*Region
}

func (o Object) Matches(str string) bool {
	if len(o.Regions) == 0 {
		return false
	}

	return strings.Contains(o.Regions[0].Parent.FileName, str) ||
		strings.Contains(o.Regions[0].Parent.FilePath, str)
}

func (o Object) String() string {
	buf := bytes.Buffer{}

	if len(o.Regions) == 0 {
		buf.WriteString(o.ID.String())

		buf.WriteString(" <empty-object>")

		return buf.String()
	}

	buf.WriteString(o.Regions[0].NameOrPath())

	buf.WriteString(" (id: ")

	buf.WriteString(o.ID.String())

	buf.WriteString(")")

	for _, region := range o.Regions {
		buf.WriteString("\n|-- ")

		buf.WriteString(region.StringWithoutObject())
	}

	return buf.String()
}

type ObjectMeta struct {
	IsSet    bool
	ID       ObjectID
	FilePath string
	FileName string
}

type ObjectID uint64

func (o ObjectID) String() string {
	return strconv.FormatUint(uint64(o), 10)
}

func (o *Regions) Add(region Region) {
	o.regions = append(o.regions, region)

	if region.Parent.IsSet {
		index, hasIt := o.objectIndex(region.Parent.ID)
		if hasIt {
			obj := &o.objects[index]
			obj.Regions = append(obj.Regions, &region)

			if region.BaseAddr < obj.BaseAddr {
				obj.BaseAddr = region.BaseAddr
			}

			if region.EndAddr > obj.EndAddr {
				obj.EndAddr = region.EndAddr
			}
		} else {
			o.objects = append(o.objects, Object{
				ID:       region.Parent.ID,
				BaseAddr: region.BaseAddr,
				EndAddr:  region.EndAddr,
				Name:     region.Parent.FileName,
				Path:     region.Parent.FilePath,
				Regions:  []*Region{&region},
			})
		}
	} else {
		o.nonobjs = append(o.nonobjs, &region)
	}
}

func (o *Regions) ObjectByID(id ObjectID) (Object, bool) {
	index, hasIt := o.objectIndex(id)
	if hasIt {
		return o.objects[index], true
	}

	return Object{}, false
}

func (o *Regions) objectIndex(id ObjectID) (int, bool) {
	for i, obj := range o.objects {
		if obj.ID == id {
			return i, true
		}
	}

	return 0, false
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

func (o *Regions) Len() int {
	return len(o.regions)
}

func (o *Regions) ObjectsLen() int {
	return len(o.objects)
}

func (o *Regions) NonObjectsLen() int {
	return len(o.nonobjs)
}

func (o *Regions) Less(i, j int) bool {
	return o.regions[i].BaseAddr < o.regions[j].EndAddr
}

func (o *Regions) Swap(i, j int) {
	o.regions[i], o.regions[j] = o.regions[j], o.regions[i]
}

func (o *Regions) Sort() {
	sort.Sort(o)
}

func (o *Regions) HasAddr(addr uintptr) (*Region, bool) {
	// This code is based on work by Stackoverflow user OneOfOne:
	// https://stackoverflow.com/a/39750394
	ln := o.Len()

	i := sort.Search(ln, func(i int) bool {
		return addr < o.regions[i].EndAddr
	})

	if i < ln {
		it := &o.regions[i]

		if addr >= it.BaseAddr && addr < it.EndAddr {
			return it, true
		}
	}

	return nil, false
}

func (o *Regions) FirstObjectMatching(str string) (Object, error) {
	var match Object
	var found bool

	err := o.IterObjectsMatching(str, func(obj Object) error {
		found = true
		match = obj

		return ErrStopIterating
	})
	if err != nil {
		return Object{}, err
	}

	if !found {
		return Object{}, fmt.Errorf("failed to find a match for: %q", str)
	}

	return match, nil
}

func (o *Regions) IterObjectsMatching(str string, fn func(Object) error) error {
	return o.IterObjects(func(obj Object) error {
		if obj.Matches(str) {
			return fn(obj)
		}

		return nil
	})
}

func (o *Regions) IterObjects(fn func(Object) error) error {
	for _, object := range o.objects {
		err := fn(object)
		if err != nil {
			if errors.Is(err, ErrStopIterating) {
				return nil
			}

			return err
		}
	}

	return nil
}

func (o *Regions) IterNonObjects(fn func(*Region) error) error {
	for _, region := range o.nonobjs {
		err := fn(region)
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
	BaseAddr  uintptr
	EndAddr   uintptr
	AllocBase uintptr
	Size      uint64
	State     MemoryState
	Type      MemoryType

	Readable   bool
	Writeable  bool
	Executable bool
	Copyable   bool
	Shared     bool

	Parent ObjectMeta
}

func (o Region) NoPermissions() bool {
	return !o.Readable && !o.Writeable && !o.Executable
}

func (o Region) String() string {
	buf := bytes.Buffer{}

	buf.WriteString(o.StringWithoutObject())

	if o.Parent.IsSet {
		buf.WriteByte(' ')

		buf.WriteString(o.NameOrPath())

		buf.WriteString(" (id: " + o.Parent.ID.String() + ")")
	}

	return buf.String()
}

func (o Region) StringWithoutObject() string {
	buf := bytes.Buffer{}

	buf.WriteString(fmt.Sprintf("%#012x-%#012x ",
		o.BaseAddr,
		o.EndAddr))

	if o.AllocBase > 0 {
		buf.WriteString(fmt.Sprintf("(allocb: %#012x) ",
			o.AllocBase))
	}

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

	if o.Shared {
		buf.WriteByte('S')
	} else {
		buf.WriteByte('-')
	}

	buf.WriteString(fmt.Sprintf(" %#012x (%s, %s)",
		o.Size,
		o.Type.String(),
		o.State.String()))

	return buf.String()
}

func (o Region) NameOrPath() string {
	switch {
	case o.Parent.FilePath != "":
		return o.Parent.FilePath
	case o.Parent.FileName != "":
		return o.Parent.FileName
	default:
		return "<no-name>"
	}
}
