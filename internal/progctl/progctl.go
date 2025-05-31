package progctl

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/SeungKang/memshonk/internal/memory"

	"github.com/mitchellh/go-ps"
)

var _ Process = (*Ctl)(nil)

var (
	ErrNotAttached    = errors.New("not attached")
	ErrDetached       = errors.New("detached")
	ErrExitedNormally = errors.New("process exited without error")
)

type Notifier interface {
	ProgramStarted(exename string)
	ProgramStopped(exename string, err error)
}

type Process interface {
	Attach(ctx context.Context) (int, error)

	ExeObject(ctx context.Context) (memory.Object, error)

	Regions(ctx context.Context) (memory.Regions, error)

	ResolvePointer(ctx context.Context, ptr memory.Pointer) (uintptr, error)

	ReadFromAddr(ctx context.Context, addr memory.Pointer, sizeBytes uint64) ([]byte, error)

	WriteToAddr(ctx context.Context, p []byte, addr memory.Pointer) error

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

	Close() error
}

func NewCtl(exeName string) *Ctl {
	return &Ctl{
		Notif:   nil,
		exeName: exeName,
	}
}

type Ctl struct {
	Notif   Notifier
	exeName string
	rwMu    sync.RWMutex
	current attachedProcess
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

	proc, err := attach(exeName, possiblePID)
	if err != nil {
		return 0, fmt.Errorf("failed to attach to process %d (%q) - %w",
			possiblePID, exeName, err)
	}

	o.current = proc
	if o.Notif != nil {
		o.Notif.ProgramStarted(o.exeName)
	}

	return proc.PID(), nil
}

func (o *Ctl) ExeObject(ctx context.Context) (memory.Object, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return memory.Object{}, ErrNotAttached
	}

	return o.current.ExeObj(), nil
}

func (o *Ctl) Regions(context.Context) (memory.Regions, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return memory.Regions{}, ErrNotAttached
	}

	return o.current.Regions()
}

func (o *Ctl) ResolvePointer(ctx context.Context, ptr memory.Pointer) (uintptr, error) {
	o.rwMu.RLock()
	defer o.rwMu.RUnlock()

	if o.current == nil {
		return 0, ErrNotAttached
	}

	return o.resolvePointer(ctx, ptr)
}

func (o *Ctl) resolvePointer(_ context.Context, ptr memory.Pointer) (uintptr, error) {
	baseAddr := o.current.ExeObj().BaseAddr

	if ptr.OptModule != "" {
		regions, err := o.current.Regions()
		if err != nil {
			return 0, err
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

	addr, err := lookupAddr(baseAddr, ptr, o.current.ReadPtr)
	if err != nil {
		return 0, fmt.Errorf("failed to lookup address - %w",
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

	return o.current.ReadBytes(addr, sizeBytes)
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

	return o.current.WriteBytes(data, addr)
}

func (o *Ctl) Detach(ctx context.Context) error {
	o.rwMu.Lock()
	defer o.rwMu.Unlock()

	if o.current == nil {
		return nil
	}

	err := o.current.Close()

	o.current = nil

	return err
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
