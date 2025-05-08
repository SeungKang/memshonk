package memory

type Pointer struct {
	Name      string
	Addrs     []uintptr
	OptModule string
}
