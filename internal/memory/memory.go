package memory

import (
	"strconv"
	"strings"
)

const (
	addrSep = ","
)

// CreatePointerFromString parses a string definition into a Pointer.
//
// Examples:
//
//   - "0xd5a351"               – absolute address
//   - "0x20,0x5,0xC0"          – pointer chain relative to the executable
//   - "buh.dll:0x20,0x5,0xC0"	– pointer chain relative to a specified module
func CreatePointerFromString(ptrDefinition string) (Pointer, error) {
	var ptr Pointer
	var ptrChain string

	before, after, found := strings.Cut(ptrDefinition, ":")
	if found {
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

	return ptr, nil
}

func AbsoluteAddrPointer(addr uintptr) Pointer {
	return Pointer{
		Addrs: []uintptr{addr},
	}
}

// Pointer represents a memory address or a chain of offsets to one
// Name is a user-defined label for identification
// Addrs is a slice of addresses:
//   - A single element represents an absolute address
//   - Multiple elements form a pointer chain via offsets
//
// OptModule specifies an optional module (e.g. a DLL) to use instead of the
// default executable base
type Pointer struct {
	Name      string
	Addrs     []uintptr
	OptModule string
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
		buf += o.OptModule + ":"
	}

	for i, addr := range o.Addrs {
		buf += "0x" + strconv.FormatUint(uint64(addr), 16)

		if i != len(o.Addrs)-1 {
			buf += addrSep
		}
	}

	return buf
}
