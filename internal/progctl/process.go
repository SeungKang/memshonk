package progctl

import (
	"fmt"
	"sync"

	"github.com/SeungKang/memshonk/internal/memory"
)

func newProcess(exeName string, pid int) (*process, error) {
	mem, err := attachProcMem(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to attach to process memory - %w", err)
	}

	objects, err := mem.Objects()
	if err != nil {
		return nil, fmt.Errorf("failed to get required modules - %w", err)
	}

	exeObj, hasIt := objects.Has(exeName)
	if !hasIt {
		return nil, fmt.Errorf("failed to get mapped object for exe: %q - %w",
			exeName, err)
	}

	return &process{
		pid:    pid,
		mem:    mem,
		exeObj: exeObj,
	}, nil
}

type process struct {
	pid    int
	mem    procMem
	exeObj memory.MappedObject
	once   sync.Once
}

func (o *process) exitMonitor() *ExitMonitor {
	return o.mem.ExitMonitor()
}

func (o *process) objects() (memory.MappedObjects, error) {
	return o.mem.Objects()
}

func (o *process) regions() (memory.Regions, error) {
	return o.mem.Regions()
}

func (o *process) read(pointer memory.Pointer, size uint64) ([]byte, error) {
	addr, err := o.resolvePointer(pointer)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve pointer for read: %q - %w",
			pointer.String(), err)
	}

	// TODO do something about this type conversion
	data, err := o.mem.ReadBytes(addr, int(size))
	if err != nil {
		return nil, fmt.Errorf("failed to read from 0x%x - %w",
			addr, err)
	}

	return data, nil
}

func (o *process) write(data []byte, pointer memory.Pointer) error {
	addr, err := o.resolvePointer(pointer)
	if err != nil {
		return fmt.Errorf("failed to resolve pointer for write: %q - %w",
			pointer.String(), err)
	}

	err = o.mem.WriteBytes(addr, data)
	if err != nil {
		return fmt.Errorf("failed to write to 0x%x - %w",
			addr, err)
	}

	return nil
}

func (o *process) resolvePointer(pointer memory.Pointer) (uintptr, error) {
	baseAddr := o.exeObj.BaseAddr

	if pointer.OptModule != "" {
		objs, err := o.objects()
		if err != nil {
			return 0, err
		}

		module, hasIt := objs.Has(pointer.OptModule)
		if !hasIt {
			return 0, fmt.Errorf("unknown memory-mapped object: %q",
				pointer.OptModule)
		}

		baseAddr = module.BaseAddr
	}

	addr, err := lookupAddr(baseAddr, pointer, o.mem.ReadPtr)
	if err != nil {
		return 0, fmt.Errorf("failed to lookup address - %w",
			err)
	}

	return addr, nil
}

func (o *process) Close() error {
	return o.mem.Close()
}

func lookupAddr(base uintptr, ptr memory.Pointer, addrFn func(uintptr) (uintptr, error)) (uintptr, error) {
	start := ptr.Addrs[0]
	// treat as absolute address
	if len(ptr.Addrs) == 1 {
		return start, nil
	}

	addr, err := addrFn(base + start)
	if err != nil {
		return 0, fmt.Errorf("failed to read from target process at 0x%x - %w",
			addr, err)
	}

	var offsets = ptr.Addrs[1:]
	for _, offset := range offsets[:len(offsets)-1] {
		addr, err = addrFn(addr + offset)
		if err != nil {
			return 0, fmt.Errorf("failed to read from target process at 0x%x - %w",
				addr, err)
		}
	}

	addr += offsets[len(offsets)-1]

	return addr, nil
}

func newExitMonitor() *ExitMonitor {
	return &ExitMonitor{
		c: make(chan struct{}),
	}
}

type ExitMonitor struct {
	c    chan struct{}
	once sync.Once
	err  error
}

func (o *ExitMonitor) Done() <-chan struct{} {
	return o.c
}

func (o *ExitMonitor) Err() error {
	return o.err
}

func (o *ExitMonitor) SetExited(err error) {
	o.once.Do(func() {
		if err == nil {
			err = ErrExitedNormally
		}

		o.err = err

		close(o.c)
	})
}
