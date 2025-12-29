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
//   - "0xd5a351"                  – absolute address
//   - "0xd5a351,0x20,0x5,0xC0"    – pointer chain relative to the executable
//   - "buh.dll:0x20,0x5,0xC0"     – pointer chain relative to a specified object
//
// Refer to the Pointer documentation for additional details.
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
	ptr.addrs = make([]uintptr, len(ptrParts))
	for i, part := range ptrParts {
		addr, err := strconv.ParseUint(strings.TrimPrefix(part, "0x"), 16, 64)
		if err != nil {
			return Pointer{}, err
		}

		ptr.addrs[i] = uintptr(addr)
	}

	if len(ptr.addrs) > 1 {
		ptr._type = ChainPointerType
	} else {
		ptr._type = AbsoluteAddrPointerType
	}

	return ptr, nil
}

// AbsoluteAddrPointer creates a Pointer to an arbitrary memory address.
func AbsoluteAddrPointer(addr uintptr) Pointer {
	return Pointer{
		addrs: []uintptr{addr},
		_type: AbsoluteAddrPointerType,
	}
}

// Pointer represents a memory address or a chain of offsets to one
// similar to Cheat Engine's pointer feature.
//
// The []uintptr returned by the Addrs method represents several
// possible values depending on the PointerType:
//
//   - AbsoluteAddrPointerType: Only the first element in the slice is
//     used and it is treated as a pointer to arbitrary memory
//   - ChainPointerType: The elements are treated as offsets from
//     one of the following addresses:
//   - The executable's base address if OptModule is empty
//   - A memory-mapped object's base address if OptModule
//     is not empty
type Pointer struct {
	// Name is a user-defined label for identification.
	Name string

	// addrs stores the address or offset(s) to an address.
	// Refer to the type's top-level documentation for details.
	addrs []uintptr

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

func (o Pointer) Addrs() []uintptr {
	return o.addrs
}

func (o Pointer) FirstAddr() uintptr {
	if len(o.addrs) > 0 {
		return o.addrs[0]
	}

	return 0
}

// MutAdvance advances the Pointer by the given value.
func (o *Pointer) MutAdvance(by uint64) {
	if by == 0 {
		return
	}

	lastIndex := len(o.addrs) - 1

	if lastIndex < 0 {
		return
	}

	last := o.addrs[lastIndex]
	o.addrs[lastIndex] = last + uintptr(by)
}

// Advance returns a copy of the Pointer with the last address increased by the
// given value.
func (o Pointer) Advance(by uint64) Pointer {
	if by == 0 {
		return o
	}

	lastIndex := len(o.addrs) - 1

	if lastIndex < 0 {
		return o
	}

	cloned := o.Clone()

	last := cloned.addrs[lastIndex]
	cloned.addrs[lastIndex] = last + uintptr(by)

	return cloned
}

// Offset returns a copy of the Pointer with the last address adjusted by the
// given signed offset.
func (o Pointer) Offset(by int64) Pointer {
	if by == 0 {
		return o
	}

	lastIndex := len(o.addrs) - 1

	if lastIndex < 0 {
		return o
	}

	cloned := o.Clone()

	last := cloned.addrs[lastIndex]

	if by < 0 {
		u := uintptr((by * -1))
		cloned.addrs[lastIndex] = last - u
	} else {
		cloned.addrs[lastIndex] = last + uintptr(by)
	}

	return cloned
}

// Clone returns a copy of the Pointer.
func (o Pointer) Clone() Pointer {
	cloned := Pointer{
		Name:      o.Name,
		addrs:     make([]uintptr, len(o.addrs)),
		OptModule: o.OptModule,
		_type:     o._type,
	}

	copy(cloned.addrs, o.addrs)

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

	for i, addr := range o.addrs {
		buf += "0x" + strconv.FormatUint(uint64(addr), 16)

		if i != len(o.addrs)-1 {
			buf += addrSep
		}
	}

	return buf
}
