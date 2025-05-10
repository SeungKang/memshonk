package memory

import (
	"strconv"
	"strings"
)

const (
	addrSep = ","
)

type Pointer struct {
	Name      string
	Addrs     []uintptr
	OptModule string
}

func (o Pointer) Advance(by uint64) Pointer {
	if by == 0 {
		return o
	}

	lastIndex := len(o.Addrs) - 1

	if lastIndex < 0 {
		return o
	}

	last := o.Addrs[lastIndex]

	o.Addrs[lastIndex] = last + uintptr(by)

	return o
}

func (o Pointer) Offset(by int64) Pointer {
	if by == 0 {
		return o
	}

	lastIndex := len(o.Addrs) - 1

	if lastIndex < 0 {
		return o
	}

	last := o.Addrs[lastIndex]

	if by < 0 {
		u := uintptr((by * -1))
		o.Addrs[lastIndex] = last - u
	} else {
		o.Addrs[lastIndex] = last + uintptr(by)
	}

	return o
}

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
