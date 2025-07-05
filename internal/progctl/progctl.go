package progctl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/SeungKang/memshonk/internal/events"
	"github.com/SeungKang/memshonk/internal/memory"
	"github.com/mitchellh/go-ps"
)

var _ Process = (*Ctl)(nil)

var (
	ErrNotAttached    = errors.New("not attached")
	ErrDetached       = errors.New("detached")
	ErrExitedNormally = errors.New("process exited without error")
)

type Process interface {
	Attach(ctx context.Context) (int, error)

	ExeObject(ctx context.Context) (memory.Object, error)

	Regions(ctx context.Context) (memory.Regions, error)

	ResolvePointer(ctx context.Context, ptr memory.Pointer) (uintptr, error)

	ReadFromAddr(ctx context.Context, addr memory.Pointer, sizeBytes uint64) ([]byte, error)

	WriteToAddr(ctx context.Context, p []byte, addr memory.Pointer) error

	Suspend(ctx context.Context) error

	Resume(ctx context.Context) error

	Detach(ctx context.Context) error
}

type attachedProcess interface {
	ExitMonitor() *ExitMonitor

	PID() int

	ExeObj() memory.Object

	ReadBytes(addr uintptr, sizeBytes uint64) ([]byte, error)

	WriteBytes(b []byte, addr uintptr) error

	ReadPtr(at uintptr) (uintptr, error)

	Regions() (memory.Regions, error)

	Suspend() error

	Resume() error

	Close() error
}

func NewCtl(exeName string, eventGroups *events.Groups) *Ctl {
	return &Ctl{
		exeName: exeName,
		events:  eventGroups,
	}
}

type Ctl struct {
	exeName string
	events  *events.Groups
	rwMu    sync.RWMutex
	current *processThread
}

func (o *Ctl) Attach(ctx context.Context) (int, error) {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.current != nil {
		select {
		case <-o.current.ExitMonitor().Done():
			// Go ahead with reattach.
		default:
			return 0, fmt.Errorf("already attached to pid: %d", o.current.PID())
		}
	}

	processes, err := ps.Processes()
	if err != nil {
		return 0, fmt.Errorf("failed to get active processes - %w", err)
	}

	possiblePID := -1
	var exeName string
	for _, psProc := range processes {
		if strings.ToLower(psProc.Executable()) == strings.ToLower(o.exeName) {
			possiblePID = psProc.Pid()
			exeName = psProc.Executable()
			break
		}
	}

	if possiblePID == -1 {
		return 0, fmt.Errorf("failed to find a matching process for: %q",
			o.exeName)
	}

	exitPub := events.NewPublisher[ProcessExitedEvent](o.events)
	exitMon := newExitMonitor(exitPub)

	proc, err := newProcessThread(exeName, possiblePID, exitMon)
	if err != nil {
		return 0, fmt.Errorf("failed to attach to process %d (%q) - %w",
			possiblePID, exeName, err)
	}

	o.current = proc

	return possiblePID, nil
}

func (o *Ctl) ExeObject(ctx context.Context) (memory.Object, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return memory.Object{}, ErrNotAttached
	}

	return o.current.ExeObj(), nil
}

func (o *Ctl) Regions(ctx context.Context) (memory.Regions, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return memory.Regions{}, ErrNotAttached
	}

	return o.regions(ctx)
}

func (o *Ctl) regions(ctx context.Context) (memory.Regions, error) {
	var regions memory.Regions
	err := o.current.Do(ctx, func(process attachedProcess) error {
		var err error
		regions, err = process.Regions()
		return err
	})

	return regions, err
}

func (o *Ctl) ResolvePointer(ctx context.Context, ptr memory.Pointer) (uintptr, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return 0, ErrNotAttached
	}

	return o.resolvePointer(ctx, ptr)
}

func (o *Ctl) resolvePointer(ctx context.Context, ptr memory.Pointer) (uintptr, error) {
	if ptr.Type() == memory.AbsoluteAddrPointerType {
		return ptr.FirstAddr(), nil
	}

	baseAddr := o.current.ExeObj().BaseAddr

	if ptr.OptModule != "" {
		regions, err := o.regions(ctx)
		if err != nil {
			return 0, fmt.Errorf("failed to get memeory regions - %w",
				err)
		}

		// TODO: This is a non-exact match. Will that
		// lead to unexpected things happening? Or do
		// we want that level of convenience?
		object, err := regions.FirstObjectMatching(ptr.OptModule)
		if err != nil {
			return 0, fmt.Errorf("failed to resolve object name - %w",
				err)
		}

		baseAddr = object.BaseAddr
	}

	var addr uintptr
	err := o.current.Do(ctx, func(process attachedProcess) error {
		var err error
		addr, err = resolvePointerChain(baseAddr, ptr, process.ReadPtr)
		return err
	})
	if err != nil {
		return 0, fmt.Errorf("failed to resolve pointer chain - %w",
			err)
	}

	return addr, nil
}

func (o *Ctl) ReadFromAddr(ctx context.Context, from memory.Pointer, sizeBytes uint64) ([]byte, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return nil, ErrNotAttached
	}

	addr, err := o.resolvePointer(ctx, from)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve pointer - %w", err)
	}

	var buf []byte
	err = o.current.Do(ctx, func(process attachedProcess) error {
		var err error
		buf, err = process.ReadBytes(addr, sizeBytes)
		return err
	})

	return buf, err
}

func (o *Ctl) WriteToAddr(ctx context.Context, data []byte, to memory.Pointer) error {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return ErrNotAttached
	}

	addr, err := o.resolvePointer(ctx, to)
	if err != nil {
		return fmt.Errorf("failed to resolve pointer - %w", err)
	}

	return o.current.Do(ctx, func(process attachedProcess) error {
		return process.WriteBytes(data, addr)
	})
}

func (o *Ctl) Suspend(ctx context.Context) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.current == nil {
		return nil
	}

	return o.current.Do(ctx, func(process attachedProcess) error {
		return process.Suspend()
	})
}

func (o *Ctl) Resume(ctx context.Context) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.current == nil {
		return nil
	}

	return o.current.Do(ctx, func(process attachedProcess) error {
		return process.Resume()
	})
}

func (o *Ctl) Detach(ctx context.Context) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.current == nil {
		return nil
	}

	err := o.current.Close(ctx)

	o.current = nil

	return err
}

// resolvePointerChain resolves a chain-type memory.Pointer.
//
// Note: This code assumes the Pointer is a chain and that the upstream
// code has ensured that is the case.
func resolvePointerChain(baseAddr uintptr, ptr memory.Pointer, addrFn func(uintptr) (uintptr, error)) (uintptr, error) {
	addr := baseAddr
	var offsets = ptr.Addrs()
	var err error

	// We are purposely skipping the last offset.
	for i, offset := range offsets[:len(offsets)-1] {
		currentTarget := addr + offset

		addr, err = addrFn(currentTarget)
		if err != nil {
			return 0, fmt.Errorf("failed to read offset index %d from process (offset: %#x | addr: %#x) - %w",
				i, offset, currentTarget, err)
		}
	}

	// sfox: I believe this is based on what Cheat Engine does.
	addr += offsets[len(offsets)-1]

	return addr, nil
}
