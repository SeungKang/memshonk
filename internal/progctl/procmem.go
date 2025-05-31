package progctl

import "github.com/SeungKang/memshonk/internal/memory"

type procMem interface {
	ExitMonitor() *ExitMonitor

	PID() int

	ExeObj() memory.Object

	ReadBytes(addr uintptr, sizeBytes uint64) ([]byte, error)

	WriteBytes(b []byte, addr uintptr) error

	ReadPtr(at uintptr) (uintptr, error)

	Regions() (memory.Regions, error)

	Close() error
}
