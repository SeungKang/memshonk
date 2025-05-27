package progctl

import (
	"errors"
	"fmt"
	"os"
	"sync"
	"syscall"

	"github.com/Andoryuuta/kiwi"
	"github.com/SeungKang/memshonk/internal/kernel32"
	"github.com/SeungKang/memshonk/internal/memory"
)

var (
	programExitedNormallyErr = errors.New("program exited without error")
)

func newProcess(exeName string, pid int) (*process, error) {
	kiwiProc, err := kiwi.GetProcessByPID(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to get process by PID - %w", err)
	}

	runningProgram := &process{
		pid:  pid,
		proc: kiwiProc,
		done: make(chan struct{}),
	}

	objects, err := getModules(uintptr(kiwiProc.Handle))
	if err != nil {
		runningProgram.Stop()
		return nil, fmt.Errorf("failed to get required modules - %w", err)
	}

	exeObj, hasIt := objects.Has(exeName)
	if !hasIt {
		runningProgram.Stop()
		return nil, fmt.Errorf("failed to get mapped object for exe: %q - %w",
			exeName, err)
	}

	runningProgram.exeObj = exeObj

	is32Bit, err := kernel32.IsProcess32Bit(syscall.Handle(kiwiProc.Handle))
	if err != nil {
		runningProgram.Stop()
		return nil, fmt.Errorf("failed to determine if process is 32 bit - %w", err)
	}
	runningProgram.is32b = is32Bit

	if is32Bit {
		runningProgram.addrFn = func(u uintptr) (uintptr, error) {
			data, err := kiwiProc.ReadUint32(u)
			return uintptr(data), err
		}
	} else {
		runningProgram.addrFn = func(u uintptr) (uintptr, error) {
			data, err := kiwiProc.ReadUint64(u)
			return uintptr(data), err
		}
	}

	// TODO: We will need to find an alternative on Unix-like systems.
	// This will not work for non-child processes.
	proc, err := os.FindProcess(int(kiwiProc.PID))
	if err != nil {
		runningProgram.Stop()
		return nil, fmt.Errorf("failed to find process with PID: %d - %w", kiwiProc.PID, err)
	}

	go func() {
		_, err := proc.Wait()
		if err == nil {
			err = programExitedNormallyErr
		}

		runningProgram.exited(err)
	}()

	return runningProgram, nil
}

type process struct {
	exeObj memory.MappedObject
	is32b  bool
	addrFn func(uintptr) (uintptr, error)
	pid    int
	proc   kiwi.Process
	once   sync.Once
	done   chan struct{}
	err    error
}

func (o *process) objects() (memory.MappedObjects, error) {
	return getModules(uintptr(o.proc.Handle))
}

func (o *process) regions() (memory.Regions, error) {
	return getRegions(uintptr(o.proc.Handle))
}

func (o *process) read(pointer memory.Pointer, size uint64) ([]byte, error) {
	addr, err := o.resolvePointer(pointer)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve pointer for read: %q - %w",
			pointer.String(), err)
	}

	// TODO do something about this type conversion
	data, err := o.proc.ReadBytes(addr, int(size))
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

	err = o.proc.WriteBytes(addr, data)
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

	addr, err := lookupAddr(baseAddr, pointer, o.addrFn)
	if err != nil {
		return 0, fmt.Errorf("failed to lookup address - %w",
			err)
	}

	return addr, nil
}

func (o *process) Stop() {
	o.exited(errors.New("stopped"))
}

func (o *process) Done() <-chan struct{} {
	if o == nil {
		return nil
	}

	return o.done
}

func (o *process) Err() error {
	return o.err
}

func (o *process) exited(err error) {
	o.once.Do(func() {
		_ = syscall.CloseHandle(syscall.Handle(o.proc.Handle))
		o.err = err
		close(o.done)
	})
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
