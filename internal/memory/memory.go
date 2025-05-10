package memory

import (
	"strconv"
	"strings"
)

type Pointer struct {
	Name      string
	Addrs     []uintptr
	OptModule string
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

	ptrParts := strings.Split(ptrChain, ",")
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
