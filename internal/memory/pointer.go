package memory

import (
	"strconv"
	"strings"
)

const (
	addrSep   = ","
	objectSep = ":"
)

// CreatePointerFromString parses a string definition into a Pointer.
//
// Examples:
//
//   - "0xd5a351"              – absolute address
//   - "0x20,0x5,0xC0"         – pointer chain relative to the executable
//   - "buh.dll:0x20,0x5,0xC0" – pointer chain relative to a specified object
func CreatePointerFromString(ptrDefinition string) (Pointer, error) {
	var ptr Pointer
	var ptrChain string

	before, after, hasObjectDelim := strings.Cut(ptrDefinition, objectSep)
	if hasObjectDelim {
		ptr.OptModule = before
		ptrChain = after
	} else {
		ptrChain = before
	}

	ptrParts := strings.Split(ptrChain, addrSep)
	ptr.Addrs = make([]uintptr, len(ptrParts))
	for i, part := range ptrParts {
		addr, err := strconv.ParseUint(strings.TrimPrefix(part, "0x"), 16, 64)
		if err != nil {
			return Pointer{}, err
		}

		ptr.Addrs[i] = uintptr(addr)
	}

	if len(ptr.Addrs) > 1 {
		ptr._type = ChainPointerType
	} else {
		ptr._type = AbsoluteAddrPointerType
	}

	return ptr, nil
}

func AbsoluteAddrPointer(addr uintptr) Pointer {
	return Pointer{
		Addrs: []uintptr{addr},
		_type: AbsoluteAddrPointerType,
	}
}

// Pointer represents a memory address or a chain of offsets to one.
type Pointer struct {
	// Name is a user-defined label for identification.
	Name string

	// Addrs is a slice of values that represents several
	// possible values:
	//
	//   - A single element represents two possible values:
	//     - If OptModule is empty, then the first value
	//       is treated as an absolute address (i.e., points
	//       directly to the target memory)
	//     - If OptModule is *not empty*, the first value
	//       is treated as an offset from the base address
	//       of that object
	//   - Multiple elements form a pointer chain via offsets
	//     similar to that of CheatEngine's pointer feature
	Addrs []uintptr

	// OptModule specifies an optional object (e.g. a DLL)
	// to use as the base address. Refer to Addrs for details.
	OptModule string

	_type PointerType
}

type PointerType int

const (
	UnknownPointerType PointerType = iota
	AbsoluteAddrPointerType
	ChainPointerType
)

func (o Pointer) Type() PointerType {
	return o._type
}

func (o Pointer) FirstAddr() uintptr {
	if len(o.Addrs) > 0 {
		return o.Addrs[0]
	}

	return 0
}

// MutAdvance advances the Pointer by the given value.
func (o *Pointer) MutAdvance(by uint64) {
	if by == 0 {
		return
	}

	lastIndex := len(o.Addrs) - 1

	if lastIndex < 0 {
		return
	}

	last := o.Addrs[lastIndex]
	o.Addrs[lastIndex] = last + uintptr(by)
}

// Advance returns a copy of the Pointer with the last address increased by the
// given value.
func (o Pointer) Advance(by uint64) Pointer {
	if by == 0 {
		return o
	}

	lastIndex := len(o.Addrs) - 1

	if lastIndex < 0 {
		return o
	}

	cloned := o.Clone()

	last := cloned.Addrs[lastIndex]
	cloned.Addrs[lastIndex] = last + uintptr(by)

	return cloned
}

// Offset returns a copy of the Pointer with the last address adjusted by the
// given signed offset.
func (o Pointer) Offset(by int64) Pointer {
	if by == 0 {
		return o
	}

	lastIndex := len(o.Addrs) - 1

	if lastIndex < 0 {
		return o
	}

	cloned := o.Clone()

	last := cloned.Addrs[lastIndex]

	if by < 0 {
		u := uintptr((by * -1))
		cloned.Addrs[lastIndex] = last - u
	} else {
		cloned.Addrs[lastIndex] = last + uintptr(by)
	}

	return cloned
}

// Clone returns a copy of the Pointer.
func (o Pointer) Clone() Pointer {
	cloned := Pointer{
		Name:      o.Name,
		Addrs:     make([]uintptr, len(o.Addrs)),
		OptModule: o.OptModule,
		_type:     o._type,
	}

	copy(cloned.Addrs, o.Addrs)

	return cloned
}

// String returns the Pointer as a hexadecimal string representation.
func (o Pointer) String() string {
	var buf string

	if o.Name != "" {
		buf += o.Name + " "
	}

	if o.OptModule != "" {
		buf += o.OptModule + objectSep
	}

	for i, addr := range o.Addrs {
		buf += "0x" + strconv.FormatUint(uint64(addr), 16)

		if i != len(o.Addrs)-1 {
			buf += addrSep
		}
	}

	return buf
}
