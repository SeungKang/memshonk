package progctl

import "github.com/SeungKang/memshonk/internal/memory"

type procMem interface {
	ExitMonitor() *ExitMonitor

	ReadBytes(addr uintptr, num int) ([]byte, error)

	WriteBytes(addr uintptr, b []byte) error

	ReadPtr(at uintptr) (uintptr, error)

	Regions() (memory.Regions, error)

	Close() error
}
